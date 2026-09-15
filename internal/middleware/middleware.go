// Package middleware provides HTTP middleware for security, observability, and robustness.
package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/amrrasi/fits/internal/logger"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request ID
// ─────────────────────────────────────────────────────────────────────────────

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID injects a unique request ID into the context and response header.
// Every log line for that request should include this ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Security Headers
// ─────────────────────────────────────────────────────────────────────────────

// SecurityHeaders adds defensive HTTP headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// Only add HSTS in production — don't break local dev
		// h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// CORS
// ─────────────────────────────────────────────────────────────────────────────

// CORSConfig holds CORS settings loaded from config.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins.
	// Use ["*"] only in development. In production list exact origins.
	AllowedOrigins []string
}

// CORS returns a middleware that sets CORS headers based on the config.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool, len(cfg.AllowedOrigins))
	allowAll := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
		}
		allowedSet[strings.TrimRight(o, "/")] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				if allowAll || allowedSet[strings.TrimRight(origin, "/")] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
					w.Header().Set("Access-Control-Allow-Headers",
						"Content-Type, Authorization, X-Request-ID")
					w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.Header().Set("Vary", "Origin")
				}
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Rate Limiter (token bucket per IP)
// ─────────────────────────────────────────────────────────────────────────────

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

// RateLimiter holds state for per-IP rate limiting.
type RateLimiter struct {
	rate     float64 // tokens per second
	burst    float64 // max burst size
	buckets  sync.Map
	stopOnce sync.Once
	stop     chan struct{}
}

// NewRateLimiter creates a limiter that allows `rate` requests/s with a burst of `burst`.
func NewRateLimiter(rate, burst float64) *RateLimiter {
	rl := &RateLimiter{
		rate:  rate,
		burst: burst,
		stop:  make(chan struct{}),
	}
	// Clean up old buckets every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cutoff := time.Now().Add(-5 * time.Minute)
				rl.buckets.Range(func(k, v interface{}) bool {
					b := v.(*bucket)
					b.mu.Lock()
					if b.lastSeen.Before(cutoff) {
						rl.buckets.Delete(k)
					}
					b.mu.Unlock()
					return true
				})
			case <-rl.stop:
				return
			}
		}
	}()
	return rl
}

// Stop shuts down the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
}

func (rl *RateLimiter) allow(ip string) bool {
	now := time.Now()
	v, _ := rl.buckets.LoadOrStore(ip, &bucket{tokens: rl.burst, lastSeen: now})
	b := v.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rate)
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Limit returns middleware that rate-limits requests by client IP.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip) {
			logger.S().Warnw("rate limit exceeded", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprintf(w, `{"error":"too many requests","code":429}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.SplitN(fwd, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ─────────────────────────────────────────────────────────────────────────────
// Request Size Limit
// ─────────────────────────────────────────────────────────────────────────────

// MaxBodySize limits incoming request bodies to n bytes.
func MaxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Request Logger
// ─────────────────────────────────────────────────────────────────────────────

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Logger logs every HTTP request with method, path, status, duration, and request ID.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		// Skip health/ready endpoints to keep logs clean
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			return
		}

		logger.S().Infow("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", GetRequestID(r.Context()),
			"ip", clientIP(r),
		)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────────────────────────────────────

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
