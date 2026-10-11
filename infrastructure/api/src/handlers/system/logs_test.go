package system

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func newLogRouter() (*gin.Engine, *logrustest.Hook) {
	gin.SetMode(gin.TestMode)
	logger, hook := logrustest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	r := gin.New()
	r.POST("/logs", FrontendLogHandler(logger))
	return r, hook
}

func postLog(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestFrontendLogRejectsOversizedBody(t *testing.T) {
	r, hook := newLogRouter()
	w := postLog(r, `{"level":"error","message":"`+strings.Repeat("A", 1<<20)+`"}`)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if len(hook.AllEntries()) != 0 {
		t.Fatal("oversized entry must not be logged")
	}
}

func TestFrontendLogTruncatesFields(t *testing.T) {
	r, hook := newLogRouter()
	w := postLog(r, `{"level":"error","message":"`+strings.Repeat("B", 10000)+`","url":"`+strings.Repeat("u", 5000)+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	e := hook.LastEntry()
	if e == nil {
		t.Fatal("expected a log entry")
	}
	if len(e.Message) > maxFrontendLogMessage+len("...[truncated]") {
		t.Fatalf("message not truncated: %d bytes", len(e.Message))
	}
	if u := e.Data["url"].(string); len(u) > maxFrontendLogURL+len("...[truncated]") {
		t.Fatalf("url not truncated: %d bytes", len(u))
	}
}

func TestFrontendLogAcceptsNormalEntry(t *testing.T) {
	r, hook := newLogRouter()
	if w := postLog(r, `{"level":"warn","message":"boom","url":"/x"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if e := hook.LastEntry(); e == nil || e.Message != "boom" || e.Level != logrus.WarnLevel {
		t.Fatalf("unexpected entry: %+v", e)
	}
}
