package auth

import (
	"math"
	"sync"
	"time"
)

const (
	blockThreshold = 5
	blockDuration  = 15 * time.Minute
	cleanInterval  = 5 * time.Minute
	entryTTL       = 30 * time.Minute
)

type rateLimitEntry struct {
	failures int
	lastFail time.Time
}

// RateLimiter provides IP-based login rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	stop    chan struct{}
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		stop:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Check returns the wait duration and whether the IP is blocked.
func (rl *RateLimiter) Check(ip string) (waitSeconds int, blocked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.entries[ip]
	if !ok {
		return 0, false
	}

	if entry.failures >= blockThreshold {
		elapsed := time.Since(entry.lastFail)
		if elapsed < blockDuration {
			return int(math.Ceil((blockDuration - elapsed).Seconds())), true
		}
		// Block expired
		delete(rl.entries, ip)
		return 0, false
	}

	if entry.failures > 0 {
		// Exponential backoff: 1s, 2s, 4s, 8s
		wait := int(math.Pow(2, float64(entry.failures-1)))
		elapsed := time.Since(entry.lastFail)
		remaining := time.Duration(wait)*time.Second - elapsed
		if remaining > 0 {
			return int(math.Ceil(remaining.Seconds())), false
		}
	}

	return 0, false
}

func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.entries[ip]
	if !ok {
		entry = &rateLimitEntry{}
		rl.entries[ip] = entry
	}
	entry.failures++
	entry.lastFail = time.Now()
}

func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, ip)
}

func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(cleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, entry := range rl.entries {
				if now.Sub(entry.lastFail) > entryTTL {
					delete(rl.entries, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}
