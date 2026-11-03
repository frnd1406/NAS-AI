package system

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	domain "github.com/nas-ai/api/src/domain/system"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

const testEventToken = "0123456789abcdef-test-token"

type fakeEventStore struct {
	alerts []*domain.SystemAlert
}

func (f *fakeEventStore) CreateReturning(_ context.Context, sev, msg string) (*domain.SystemAlert, error) {
	a := &domain.SystemAlert{ID: "id-" + string(rune('a'+len(f.alerts))), Severity: sev, Message: msg, CreatedAt: time.Now()}
	f.alerts = append(f.alerts, a)
	return a, nil
}

func (f *fakeEventStore) FindRecentOpenByPrefix(_ context.Context, prefix string, since time.Time) (*domain.SystemAlert, error) {
	for _, a := range f.alerts {
		if !a.IsResolved && !a.CreatedAt.Before(since) && strings.HasPrefix(a.Message, prefix) {
			return a, nil
		}
	}
	return nil, nil
}

func (f *fakeEventStore) UpdateSeverityMessage(_ context.Context, id, sev, msg string) error {
	for _, a := range f.alerts {
		if a.ID == id {
			a.Severity, a.Message = sev, msg
		}
	}
	return nil
}

func newEventRouter(store EventAlertStore, pub AlertPublisher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger, _ := logrustest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	r := gin.New()
	r.POST("/events", SystemEventsHandler(store, pub, testEventToken, logger))
	return r
}

func postEvent(r *gin.Engine, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Monitoring-Token", token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const validUFW = `{"source":"nas","type":"ufw_block","src_ip":"192.168.178.50","count":7,"window_seconds":30}`

func TestEventsRejectsMissingAndWrongToken(t *testing.T) {
	store := &fakeEventStore{}
	r := newEventRouter(store, nil)
	for _, tok := range []string{"", "wrong-token"} {
		if w := postEvent(r, tok, validUFW); w.Code != http.StatusUnauthorized {
			t.Fatalf("token %q: status = %d, want 401", tok, w.Code)
		}
	}
	if len(store.alerts) != 0 {
		t.Fatal("unauthorized call must not create alerts")
	}
}

func TestEventsValidUFWBlockCreatesWarning(t *testing.T) {
	store := &fakeEventStore{}
	var topic string
	r := newEventRouter(store, func(_ context.Context, tp string, _ []byte) error { topic = tp; return nil })
	w := postEvent(r, testEventToken, validUFW)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if len(store.alerts) != 1 || store.alerts[0].Severity != "WARNING" {
		t.Fatalf("unexpected alerts: %+v", store.alerts)
	}
	if topic != SystemAlertsTopic {
		t.Fatalf("published topic = %q, want %q", topic, SystemAlertsTopic)
	}
}

func TestEventsInvalidPayloads(t *testing.T) {
	cases := map[string]string{
		"bad ip":      `{"source":"nas","type":"ufw_block","src_ip":"999.1.1.1","count":1,"window_seconds":30}`,
		"ip injected": `{"source":"nas","type":"ufw_block","src_ip":"1.1.1.1 <script>","count":1,"window_seconds":30}`,
		"bad type":    `{"source":"nas","type":"rm_rf","src_ip":"1.1.1.1","count":1,"window_seconds":30}`,
		"zero count":  `{"source":"nas","type":"ufw_block","src_ip":"1.1.1.1","count":0,"window_seconds":30}`,
		"huge count":  `{"source":"nas","type":"ufw_block","src_ip":"1.1.1.1","count":999999999,"window_seconds":30}`,
		"bad window":  `{"source":"nas","type":"ufw_block","src_ip":"1.1.1.1","count":1,"window_seconds":0}`,
		"bad source":  `{"source":"na s;","type":"ufw_block","src_ip":"1.1.1.1","count":1,"window_seconds":30}`,
		"not json":    `nope`,
	}
	for name, body := range cases {
		store := &fakeEventStore{}
		r := newEventRouter(store, nil)
		if w := postEvent(r, testEventToken, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, w.Code)
		}
		if len(store.alerts) != 0 {
			t.Errorf("%s: invalid payload created an alert", name)
		}
	}
}

func TestEventsSMBAuthFailSeverity(t *testing.T) {
	for count, want := range map[int]string{1: "WARNING", 2: "WARNING", 3: "HIGH", 10: "HIGH"} {
		store := &fakeEventStore{}
		r := newEventRouter(store, nil)
		body := `{"source":"nas","type":"smb_auth_fail","src_ip":"192.168.178.60","count":` + itoa(count) + `,"window_seconds":30}`
		if w := postEvent(r, testEventToken, body); w.Code != http.StatusCreated {
			t.Fatalf("count %d: status %d", count, w.Code)
		}
		if got := store.alerts[0].Severity; got != want {
			t.Errorf("count %d: severity %s, want %s", count, got, want)
		}
	}
}

func itoa(n int) string { return string(appendInt(nil, n)) }

func appendInt(b []byte, n int) []byte {
	if n >= 10 {
		b = appendInt(b, n/10)
	}
	return append(b, byte('0'+n%10))
}

func TestEventsAggregateWithinWindow(t *testing.T) {
	store := &fakeEventStore{}
	r := newEventRouter(store, nil)
	postEvent(r, testEventToken, validUFW)
	w := postEvent(r, testEventToken, validUFW)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
	if len(store.alerts) != 1 {
		t.Fatalf("alerts = %d, want 1 (aggregated)", len(store.alerts))
	}
	if !strings.Contains(store.alerts[0].Message, "total=14") {
		t.Fatalf("message not aggregated: %s", store.alerts[0].Message)
	}
	// different source IP -> separate alert
	postEvent(r, testEventToken, strings.Replace(validUFW, "192.168.178.50", "192.168.178.51", 1))
	if len(store.alerts) != 2 {
		t.Fatalf("different IP must create new alert, got %d", len(store.alerts))
	}
	// different type, same IP -> separate alert
	postEvent(r, testEventToken, strings.Replace(validUFW, "ufw_block", "smb_auth_fail", 1))
	if len(store.alerts) != 3 {
		t.Fatalf("different type must create new alert, got %d", len(store.alerts))
	}
}

func TestEventsNoAggregationAfterWindow(t *testing.T) {
	store := &fakeEventStore{}
	r := newEventRouter(store, nil)
	postEvent(r, testEventToken, validUFW)
	store.alerts[0].CreatedAt = time.Now().Add(-6 * time.Minute)
	postEvent(r, testEventToken, validUFW)
	if len(store.alerts) != 2 {
		t.Fatalf("alerts = %d, want 2 after window expiry", len(store.alerts))
	}
}

func TestEventsAggregationNeverDowngradesSeverity(t *testing.T) {
	store := &fakeEventStore{}
	r := newEventRouter(store, nil)
	high := `{"source":"nas","type":"smb_auth_fail","src_ip":"192.168.178.60","count":5,"window_seconds":30}`
	low := `{"source":"nas","type":"smb_auth_fail","src_ip":"192.168.178.60","count":1,"window_seconds":30}`
	postEvent(r, testEventToken, high)
	postEvent(r, testEventToken, low)
	if store.alerts[0].Severity != "HIGH" || len(store.alerts) != 1 {
		t.Fatalf("got %+v", store.alerts)
	}
}
