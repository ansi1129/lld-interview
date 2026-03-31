package main

import (
	"fmt"
	"sync"
	"time"
)

//////////////////////////////////////////////////////
// INTERFACE
//////////////////////////////////////////////////////

type RateLimiter interface {
	Allow(key string) bool
}

//////////////////////////////////////////////////////
// CONFIG
//////////////////////////////////////////////////////

type RateLimitConfig struct {
	Limit      int
	WindowSize time.Duration
	Algorithm  string
}

//////////////////////////////////////////////////////
// FIXED WINDOW
//////////////////////////////////////////////////////

type FixedWindowLimiter struct {
	limit      int
	window     time.Duration
	counts     map[string]int
	timestamps map[string]time.Time
	mutex      sync.Mutex
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		limit:      limit,
		window:     window,
		counts:     make(map[string]int),
		timestamps: make(map[string]time.Time),
	}
}

func (f *FixedWindowLimiter) Allow(key string) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	now := time.Now()

	if t, ok := f.timestamps[key]; !ok || now.Sub(t) >= f.window {
		f.timestamps[key] = now
		f.counts[key] = 1
		return true
	}

	if f.counts[key] < f.limit {
		f.counts[key]++
		return true
	}

	return false
}

//////////////////////////////////////////////////////
// TOKEN BUCKET
//////////////////////////////////////////////////////

type TokenBucketLimiter struct {
	capacity   int
	refillRate int
	tokens     map[string]int
	lastRefill map[string]time.Time
	mutex      sync.Mutex
}

func NewTokenBucketLimiter(capacity, refillRate int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     make(map[string]int),
		lastRefill: make(map[string]time.Time),
	}
}

func (t *TokenBucketLimiter) Allow(key string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()

	last, exists := t.lastRefill[key]
	if !exists {
		t.tokens[key] = t.capacity
		t.lastRefill[key] = now
	}

	elapsed := int(now.Sub(last).Seconds())
	refill := elapsed * t.refillRate

	if refill > 0 {
		t.tokens[key] = min(t.capacity, t.tokens[key]+refill)
		t.lastRefill[key] = now
	}

	if t.tokens[key] > 0 {
		t.tokens[key]--
		return true
	}

	return false
}

//////////////////////////////////////////////////////
// SLIDING WINDOW COUNTER (OPTIMIZED)
//////////////////////////////////////////////////////

type SlidingWindowLimiter struct {
	limit       int
	window      time.Duration
	current     map[string]int
	previous    map[string]int
	lastUpdated map[string]time.Time
	mutex       sync.Mutex
}

func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:       limit,
		window:      window,
		current:     make(map[string]int),
		previous:    make(map[string]int),
		lastUpdated: make(map[string]time.Time),
	}
}

func (s *SlidingWindowLimiter) Allow(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	last := s.lastUpdated[key]

	if last.IsZero() {
		s.lastUpdated[key] = now
	}

	elapsed := now.Sub(last)

	if elapsed >= s.window {
		s.previous[key] = s.current[key]
		s.current[key] = 0
		s.lastUpdated[key] = now
		elapsed = 0
	}

	weight := float64(s.window-elapsed) / float64(s.window)
	count := float64(s.previous[key])*weight + float64(s.current[key])

	if int(count) < s.limit {
		s.current[key]++
		return true
	}

	return false
}

//////////////////////////////////////////////////////
// FACTORY
//////////////////////////////////////////////////////

type RateLimiterFactory struct{}

func (f *RateLimiterFactory) GetLimiter(cfg RateLimitConfig) RateLimiter {

	switch cfg.Algorithm {

	case "FIXED":
		return NewFixedWindowLimiter(cfg.Limit, cfg.WindowSize)

	case "TOKEN":
		return NewTokenBucketLimiter(cfg.Limit, 1)

	case "SLIDING":
		return NewSlidingWindowLimiter(cfg.Limit, cfg.WindowSize)

	default:
		return NewFixedWindowLimiter(cfg.Limit, cfg.WindowSize)
	}
}

//////////////////////////////////////////////////////
// SERVICE
//////////////////////////////////////////////////////

type RateLimiterService struct {
	limiters map[string]RateLimiter
	factory  *RateLimiterFactory
	mutex    sync.RWMutex
}

func NewRateLimiterService() *RateLimiterService {
	return &RateLimiterService{
		limiters: make(map[string]RateLimiter),
		factory:  &RateLimiterFactory{},
	}
}

func (r *RateLimiterService) Register(key string, cfg RateLimitConfig) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.limiters[key] = r.factory.GetLimiter(cfg)
}

func (r *RateLimiterService) Allow(key string) bool {
	r.mutex.RLock()
	limiter, ok := r.limiters[key]
	r.mutex.RUnlock()

	if !ok {
		return true // no limit
	}

	return limiter.Allow(key)
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {

	service := NewRateLimiterService()

	// Register user rate limit
	service.Register("user1", RateLimitConfig{
		Limit:      5,
		WindowSize: 10 * time.Second,
		Algorithm:  "FIXED",
	})

	// Simulate requests
	for i := 0; i < 15; i++ {
		if service.Allow("user1") {
			fmt.Println("Allowed")
		} else {
			fmt.Println("Blocked")
		}
		time.Sleep(1 * time.Second)
	}
}

//////////////////////////////////////////////////////
// UTILS
//////////////////////////////////////////////////////

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
