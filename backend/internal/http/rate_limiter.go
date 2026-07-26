package http

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type requestRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	count int
	start time.Time
}

const maxRateLimitEntries = 4096

func newRequestRateLimiter(limit int, window time.Duration) *requestRateLimiter {
	return &requestRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateLimitEntry),
	}
}

func (l *requestRateLimiter) Allow(key string, now time.Time) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true
	}
	key = strings.TrimSpace(key)
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, exists := l.entries[key]
	if !exists && len(l.entries) >= maxRateLimitEntries {
		for candidate, candidateEntry := range l.entries {
			if !now.Before(candidateEntry.start.Add(l.window)) {
				delete(l.entries, candidate)
			}
		}
		if len(l.entries) >= maxRateLimitEntries {
			return false
		}
	}
	if entry.start.IsZero() || !now.Before(entry.start.Add(l.window)) {
		l.entries[key] = rateLimitEntry{count: 1, start: now}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func requestClientIP(request *http.Request, trustProxy bool) string {
	if request == nil {
		return ""
	}
	remoteIP := remoteRequestIP(request.RemoteAddr)
	if !trustProxy {
		return remoteIP
	}
	values := request.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return remoteIP
	}
	first := ""
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			candidate = strings.TrimSpace(candidate)
			ip := net.ParseIP(candidate)
			if ip == nil {
				return remoteIP
			}
			if first == "" {
				first = ip.String()
			}
		}
	}
	if first == "" {
		return remoteIP
	}
	return first
}

func remoteRequestIP(remoteAddress string) string {
	remoteAddress = strings.TrimSpace(remoteAddress)
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	if ip := net.ParseIP(remoteAddress); ip != nil {
		return ip.String()
	}
	return remoteAddress
}
