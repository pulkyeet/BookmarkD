package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu sync.Mutex
	limit int
	window time.Duration
}

var limiter = &rateLimiter{
	requests: make(map[string][]time.Time),
	limit: 5,
	window: 1 * time.Minute,
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	windowStart := now.Add(-rl.window)
	
	if timestamps, exists := rl.requests[key]; exists {
		valid := []time.Time{}
		for _, ts := range timestamps {
			if ts.After(windowStart) {
				valid = append(valid, ts)
			}
		}
		rl.requests[key] = valid
	}
	if len(rl.requests[key]) >= rl.limit {
		return false
	}
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
		if !limiter.allow(ip) {
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}