package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/amrrasi/fits/internal/logger"
)

func TestMain(m *testing.M) {
	dir, _ := os.MkdirTemp("", "fitslog")
	_ = logger.Init(logger.Config{Level: "error", LogDir: dir, AppName: "test"})
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestClientIPIgnoresSpoofedHeaderFromUntrustedPeer(t *testing.T) {
	_ = SetTrustedProxies(nil)
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.9:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIP(r); got != "203.0.113.9" {
		t.Fatalf("got %s", got)
	}
}

func TestClientIPTrustedProxy(t *testing.T) {
	_ = SetTrustedProxies([]string{"10.0.0.0/8"})
	defer SetTrustedProxies(nil) //nolint:errcheck
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:1"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 198.51.100.7, 10.0.0.9")
	if got := ClientIP(r); got != "198.51.100.7" {
		t.Fatalf("got %s", got)
	}
}

func TestCORSWildcardNeverWithCredentials(t *testing.T) {
	h := CORS(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Credentials") != "" || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("unsafe CORS headers: %v", w.Header())
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(0.001, 2)
	defer rl.Stop()
	if !rl.allow("a") || !rl.allow("a") || rl.allow("a") {
		t.Fatal("burst of 2 expected")
	}
	if !rl.allow("b") {
		t.Fatal("separate key must have its own bucket")
	}
}

func TestRecoverer(t *testing.T) {
	h := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 {
		t.Fatalf("code %d", w.Code)
	}
}
