package system

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const alertStreamHeartbeat = 15 * time.Second

// AlertSubscriber subscribes to a live topic. It returns a message channel and
// a cancel func that must be called when the client goes away.
type AlertSubscriber func(ctx context.Context, topic string) (<-chan []byte, func(), error)

// SystemAlertsStreamHandler streams alert updates as Server-Sent Events
// (GET /system/alerts/stream, JWT protected). Works with URLSession (iOS) and
// fetch/EventSource-style clients (Tauri). Each message:
//
//	event: alert
//	data: {"event":"alert","aggregated":false,"alert":{...}}
//
// A ": ping" comment is sent every 15 s to keep proxies/NAT from dropping it.
// Clients should reconnect on disconnect and re-sync via GET /system/alerts.
func SystemAlertsStreamHandler(sub AlertSubscriber, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sub == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"code": "unavailable", "message": "live stream not available", "request_id": c.GetString("request_id"),
			}})
			return
		}
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"code": "internal_error", "message": "streaming unsupported", "request_id": c.GetString("request_id"),
			}})
			return
		}

		ctx := c.Request.Context()
		msgs, cancel, err := sub(ctx, SystemAlertsTopic)
		if err != nil {
			logger.WithError(err).Error("alert stream subscribe failed")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"code": "unavailable", "message": "live stream not available", "request_id": c.GetString("request_id"),
			}})
			return
		}
		defer cancel()

		// The server has a global WriteTimeout; lift it for this long-lived stream.
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetWriteDeadline(time.Time{})

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		fmt.Fprint(c.Writer, ": connected\n\n")
		flusher.Flush()

		ticker := time.NewTicker(alertStreamHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case m, open := <-msgs:
				if !open {
					return
				}
				if _, err := fmt.Fprintf(c.Writer, "event: alert\ndata: %s\n\n", sanitizeSSE(m)); err != nil {
					return
				}
				flusher.Flush()
			case <-ticker.C:
				if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// sanitizeSSE removes newlines so a payload can never break out of its data line.
func sanitizeSSE(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, ch := range b {
		if ch == '\n' || ch == '\r' {
			ch = ' '
		}
		out = append(out, ch)
	}
	return out
}
