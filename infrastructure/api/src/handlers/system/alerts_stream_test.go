package system

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestAlertsStreamDeliversEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := logrustest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ch := make(chan []byte, 4)
	cancelled := make(chan struct{})
	var gotTopic string
	sub := func(_ context.Context, topic string) (<-chan []byte, func(), error) {
		gotTopic = topic
		return ch, func() { close(cancelled) }, nil
	}
	r := gin.New()
	r.GET("/stream", SystemAlertsStreamHandler(sub, logger))
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}

	ch <- []byte("{\"alert\":\"a\nb\"}") // newline must not break the frame
	sc := bufio.NewScanner(resp.Body)
	var lines []string
	deadline := time.After(3 * time.Second)
	done := make(chan struct{})
	go func() {
		for sc.Scan() {
			lines = append(lines, sc.Text())
			if strings.HasPrefix(sc.Text(), "data: ") {
				close(done)
				return
			}
		}
	}()
	select {
	case <-done:
	case <-deadline:
		t.Fatal("no data frame received")
	}
	joined := strings.Join(lines, "|")
	if !strings.Contains(joined, "event: alert") || !strings.Contains(joined, `data: {"alert":"a b"}`) {
		t.Fatalf("unexpected frames: %s", joined)
	}
	if gotTopic != SystemAlertsTopic {
		t.Fatalf("topic = %q", gotTopic)
	}

	cancel() // client disconnect must release the subscription
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("subscription not released after disconnect")
	}
}

func TestAlertsStreamUnavailableWithoutSubscriber(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := logrustest.NewNullLogger()
	r := gin.New()
	r.GET("/stream", SystemAlertsStreamHandler(nil, logger))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
