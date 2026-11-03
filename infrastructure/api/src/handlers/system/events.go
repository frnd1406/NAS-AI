package system

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	domain "github.com/nas-ai/api/src/domain/system"
	"github.com/sirupsen/logrus"
)

const (
	EventTypeUFWBlock    = "ufw_block"
	EventTypeSMBAuthFail = "smb_auth_fail"

	// SystemAlertsTopic is the pub/sub channel live alert updates are pushed to.
	SystemAlertsTopic = "system_alerts"

	eventMaxCount         = 100000
	eventMaxWindowSeconds = 3600
	eventAggregateWindow  = 5 * time.Minute
	eventAuthFailHighAt   = 3
	eventMaxBodyBytes     = 4 << 10

	severityWarningEvt = "WARNING"
	severityHighEvt    = "HIGH"
)

var eventSourceRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// SystemEventRequest is the payload sent by host-side monitors.
type SystemEventRequest struct {
	Source        string `json:"source"`
	Type          string `json:"type"`
	SrcIP         string `json:"src_ip"`
	Count         int    `json:"count"`
	WindowSeconds int    `json:"window_seconds"`
}

// EventAlertStore is the persistence needed to turn events into alerts.
type EventAlertStore interface {
	CreateReturning(ctx context.Context, severity, message string) (*domain.SystemAlert, error)
	FindRecentOpenByPrefix(ctx context.Context, prefix string, since time.Time) (*domain.SystemAlert, error)
	UpdateSeverityMessage(ctx context.Context, id, severity, message string) error
}

// AlertPublisher pushes a payload to a live topic. May be nil (disabled).
type AlertPublisher func(ctx context.Context, topic string, payload []byte) error

func (r *SystemEventRequest) validate() error {
	if r.Type != EventTypeUFWBlock && r.Type != EventTypeSMBAuthFail {
		return fmt.Errorf("type must be one of ufw_block, smb_auth_fail")
	}
	if !eventSourceRe.MatchString(r.Source) {
		return fmt.Errorf("source must match [A-Za-z0-9._-]{1,64}")
	}
	ip := net.ParseIP(r.SrcIP)
	if ip == nil {
		return fmt.Errorf("src_ip is not a valid IP address")
	}
	r.SrcIP = ip.String() // normalised, never raw user text
	if r.Count < 1 || r.Count > eventMaxCount {
		return fmt.Errorf("count must be between 1 and %d", eventMaxCount)
	}
	if r.WindowSeconds < 1 || r.WindowSeconds > eventMaxWindowSeconds {
		return fmt.Errorf("window_seconds must be between 1 and %d", eventMaxWindowSeconds)
	}
	return nil
}

func eventSeverity(typ string, count int) string {
	if typ == EventTypeSMBAuthFail && count >= eventAuthFailHighAt {
		return severityHighEvt
	}
	return severityWarningEvt
}

func eventPrefix(r *SystemEventRequest) string {
	return fmt.Sprintf("[%s src=%s source=%s]", r.Type, r.SrcIP, r.Source)
}

var eventTotalRe = regexp.MustCompile(`total=(\d+)`)

func eventMessage(r *SystemEventRequest, total int) string {
	what := "UFW blocked connections to port 445"
	if r.Type == EventTypeSMBAuthFail {
		what = "failed SMB logins"
	}
	return fmt.Sprintf("%s %s: total=%d (last report: %d in %ds)",
		eventPrefix(r), what, total, r.Count, r.WindowSeconds)
}

// SystemEventsHandler ingests host events (UFW blocks, SMB auth failures) and
// turns them into aggregated system alerts. Auth uses X-Monitoring-Token.
func SystemEventsHandler(store EventAlertStore, publish AlertPublisher, apiKey string, logger *logrus.Logger) gin.HandlerFunc {
	fail := func(c *gin.Context, status int, code, msg string) {
		c.JSON(status, gin.H{"error": gin.H{
			"code": code, "message": msg, "request_id": c.GetString("request_id"),
		}})
	}

	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		token := c.GetHeader("X-Monitoring-Token")
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			logger.WithField("request_id", requestID).Warn("unauthorized system events call")
			fail(c, http.StatusUnauthorized, "unauthorized", "invalid or missing monitoring token")
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, eventMaxBodyBytes)
		var req SystemEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "invalid_request", "invalid payload")
			return
		}
		req.Source = strings.TrimSpace(req.Source)
		if err := req.validate(); err != nil {
			fail(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		ctx := c.Request.Context()
		severity := eventSeverity(req.Type, req.Count)
		prefix := eventPrefix(&req)

		existing, err := store.FindRecentOpenByPrefix(ctx, prefix+" ", time.Now().Add(-eventAggregateWindow))
		if err != nil {
			logger.WithFields(logrus.Fields{"request_id": requestID, "error": err.Error()}).Error("event alert lookup failed")
			fail(c, http.StatusInternalServerError, "internal_error", "failed to store event")
			return
		}

		var alert *domain.SystemAlert
		aggregated := false
		if existing != nil {
			total := req.Count
			if m := eventTotalRe.FindStringSubmatch(existing.Message); m != nil {
				if n, convErr := strconv.Atoi(m[1]); convErr == nil && n > 0 && n <= eventMaxCount*100 {
					total += n
				}
			}
			if existing.Severity == severityHighEvt {
				severity = severityHighEvt // never downgrade
			}
			msg := eventMessage(&req, total)
			if err = store.UpdateSeverityMessage(ctx, existing.ID, severity, msg); err == nil {
				existing.Severity, existing.Message = severity, msg
				alert, aggregated = existing, true
			}
		} else {
			alert, err = store.CreateReturning(ctx, severity, eventMessage(&req, req.Count))
		}
		if err != nil {
			logger.WithFields(logrus.Fields{"request_id": requestID, "error": err.Error()}).Error("failed to store event alert")
			fail(c, http.StatusInternalServerError, "internal_error", "failed to store event")
			return
		}

		if publish != nil {
			if payload, mErr := json.Marshal(gin.H{"event": "alert", "aggregated": aggregated, "alert": alert}); mErr == nil {
				if pErr := publish(ctx, SystemAlertsTopic, payload); pErr != nil {
					logger.WithFields(logrus.Fields{"request_id": requestID, "error": pErr.Error()}).Warn("live alert publish failed")
				}
			}
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":     "created",
			"alert_id":   alert.ID,
			"severity":   alert.Severity,
			"aggregated": aggregated,
			"request_id": requestID,
		})
	}
}
