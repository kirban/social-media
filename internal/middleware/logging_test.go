package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kirban/social-media/internal/config"
	applogger "github.com/kirban/social-media/internal/logger"
)

// hijackableRecorder is an httptest.ResponseRecorder that also implements
// http.Hijacker, standing in for the real net/http connection.
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

func testLogger(t *testing.T) *applogger.AppLogger {
	t.Helper()

	l, err := applogger.NewAppLogger(&config.Config{Env: "local", LogLevel: "disabled"})
	if err != nil {
		t.Fatalf("build logger: %v", err)
	}
	return l
}

// The logging middleware wraps the ResponseWriter to record the status code.
// That wrapper must not hide http.Hijacker, or every WebSocket upgrade fails
// with "http.ResponseWriter does not implement http.Hijacker" — the response
// being a 501 that looks like an unimplemented route rather than a wrapper bug.
func TestLoggingPreservesHijacker(t *testing.T) {
	var (
		sawHijacker bool
		hijackErr   error
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		sawHijacker = ok
		if ok {
			_, _, hijackErr = hj.Hijack()
		}
	})

	rec := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	Logging(testLogger(t))(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/post/feed/posted", nil))

	if !sawHijacker {
		t.Fatal("wrapped ResponseWriter does not implement http.Hijacker; WebSocket upgrades will fail")
	}
	if hijackErr != nil {
		t.Fatalf("Hijack returned error: %v", hijackErr)
	}
	if !rec.hijacked {
		t.Fatal("Hijack did not reach the underlying ResponseWriter")
	}
}

// When the underlying writer genuinely cannot be hijacked, the wrapper must
// report that rather than panicking.
func TestLoggingHijackFailsCleanlyWhenUnsupported(t *testing.T) {
	var hijackErr error

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("wrapper should always expose Hijacker")
		}
		_, _, hijackErr = hj.Hijack()
	})

	// A plain recorder is not an http.Hijacker.
	Logging(testLogger(t))(next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if hijackErr == nil {
		t.Fatal("expected an error when the underlying writer is not hijackable")
	}
}

func TestLoggingRecordsStatus(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	Logging(testLogger(t))(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}
