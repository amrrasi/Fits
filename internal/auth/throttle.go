package auth

import (
	"sync"
	"time"
)

type attempt struct {
	fails   int
	first   time.Time
	blocked time.Time
}

// LoginThrottle counts failed logins per (email+ip), per email and per ip (in memory).
type LoginThrottle struct {
	mu     sync.Mutex
	m      map[string]*attempt
	window time.Duration
	block  time.Duration
}

func NewLoginThrottle() *LoginThrottle {
	return &LoginThrottle{m: map[string]*attempt{}, window: 15 * time.Minute, block: 15 * time.Minute}
}

func limits(key string) int {
	switch {
	case len(key) > 4 && key[:4] == "acc:":
		return 5
	case len(key) > 3 && key[:3] == "ip:":
		return 30
	default:
		return 40 // per-email across all IPs (high, so an attacker cannot easily lock a victim out)
	}
}

func keysFor(email, ip string) []string {
	return []string{"acc:" + email + "|" + ip, "ip:" + ip, "em:" + email}
}

// Blocked reports whether any key is currently blocked and for how long.
func (t *LoginThrottle) Blocked(email, ip string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	var worst time.Duration
	for _, k := range keysFor(email, ip) {
		if a := t.m[k]; a != nil && now.Before(a.blocked) {
			if d := a.blocked.Sub(now); d > worst {
				worst = d
			}
		}
	}
	return worst, worst > 0
}

func (t *LoginThrottle) Fail(email, ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if len(t.m) > 50000 {
		t.m = map[string]*attempt{}
	}
	for _, k := range keysFor(email, ip) {
		a := t.m[k]
		if a == nil || now.Sub(a.first) > t.window {
			a = &attempt{first: now}
			t.m[k] = a
		}
		a.fails++
		if a.fails >= limits(k) {
			a.blocked = now.Add(t.block)
			a.fails = 0
			a.first = now
		}
	}
}

func (t *LoginThrottle) Success(email, ip string) {
	t.mu.Lock()
	delete(t.m, "acc:"+email+"|"+ip)
	t.mu.Unlock()
}
