package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/amrrasi/fits/internal/logger"
)

type contextKey string

const requestIDKey contextKey = "request_id"

var requestIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID attaches a request id (a client-supplied one is accepted only if it looks sane).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !requestIDRe.MatchString(id) {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Recoverer turns handler panics into a clean 500 instead of a dropped connection.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.S().Errorw("panic recovered", "panic", rec, "path", r.URL.Path, "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"مشکلی در سرور داخلی پیش آمده","code":500}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ── Client IP / trusted proxies ───────────────────────────────────────────────

var (
	trustMu sync.RWMutex
	trusted []*net.IPNet
)

// SetTrustedProxies configures which peers may set X-Forwarded-For (CIDRs or single IPs).
func SetTrustedProxies(cidrs []string) error {
	var nets []*net.IPNet
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if strings.Contains(c, ":") {
				c += "/128"
			} else {
				c += "/32"
			}
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return err
		}
		nets = append(nets, n)
	}
	trustMu.Lock()
	trusted = nets
	trustMu.Unlock()
	return nil
}

func isTrusted(ip net.IP) bool {
	trustMu.RLock()
	defer trustMu.RUnlock()
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP returns the real client address. X-Forwarded-For is honoured only when the
// direct peer is a configured trusted proxy, and is walked right-to-left.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil || !isTrusted(peer) {
		return host
	}
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(parts[i]))
		if ip == nil {
			continue
		}
		if !isTrusted(ip) {
			return ip.String()
		}
	}
	return host
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if p := net.ParseIP(host); p != nil && isTrusted(p) {
		return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	}
	return false
}

// ── Security headers ──────────────────────────────────────────────────────────

const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
	"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", csp)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		if isHTTPS(r) {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// ── CORS ──────────────────────────────────────────────────────────────────────

type CORSConfig struct{ AllowedOrigins []string }

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(cfg.AllowedOrigins))
	allowAll := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[strings.TrimRight(o, "/")] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			h := w.Header()
			h.Add("Vary", "Origin")
			if origin != "" {
				ok := allowed[strings.TrimRight(origin, "/")]
				if ok || allowAll {
					if ok {
						h.Set("Access-Control-Allow-Origin", origin)
						h.Set("Access-Control-Allow-Credentials", "true")
					} else {
						// wildcard is never combined with credentials
						h.Set("Access-Control-Allow-Origin", "*")
					}
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Requested-With")
					h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
					h.Set("Access-Control-Max-Age", "600")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ── Rate limiting ─────────────────────────────────────────────────────────────

type bucket struct {
	tokens float64
	last   time.Time
}

const maxBuckets = 100_000

type RateLimiter struct {
	rate, burst float64
	mu          sync.Mutex
	buckets     map[string]*bucket
	stopOnce    sync.Once
	stop        chan struct{}
}

func NewRateLimiter(rate, burst float64) *RateLimiter {
	rl := &RateLimiter{rate: rate, burst: burst, buckets: map[string]*bucket{}, stop: make(chan struct{})}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				rl.mu.Lock()
				cut := time.Now().Add(-5 * time.Minute)
				for k, b := range rl.buckets {
					if b.last.Before(cut) {
						delete(rl.buckets, k)
					}
				}
				rl.mu.Unlock()
			case <-rl.stop:
				return
			}
		}
	}()
	return rl
}

func (rl *RateLimiter) Stop() { rl.stopOnce.Do(func() { close(rl.stop) }) }

func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		if len(rl.buckets) >= maxBuckets {
			return false // memory guard under a flood of distinct keys
		}
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func tooMany(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"تعداد درخواست‌ها بیش از حد مجاز است؛ کمی صبر کنید","code":429}`))
}

// Limit applies one limiter to everything.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(ClientIP(r)) {
			tooMany(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SplitLimit uses a stricter limiter for /api/auth/* and a general one for the rest.
func SplitLimit(auth, api *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				if !auth.allow(ip) {
					logger.S().Warnw("auth rate limit exceeded", "ip", ip)
					tooMany(w)
					return
				}
			} else if !api.allow(ip) {
				tooMany(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func MaxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(s int)           { rw.status = s; rw.ResponseWriter.WriteHeader(s) }
func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			return
		}
		logger.S().Infow("http",
			"method", r.Method, "path", r.URL.Path, "status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", GetRequestID(r.Context()), "ip", ClientIP(r))
	})
}

// JSONError writes a small JSON error (used by other packages' middleware).
func JSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]interface{}{"error": msg, "code": status})
	_, _ = w.Write(b)
	_ = strconv.Itoa
}
