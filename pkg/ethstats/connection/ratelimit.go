package connection

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type RateLimiter struct {
	mu             sync.RWMutex
	connections    map[string]int
	failures       map[string]int
	windowStart    time.Time
	windowDuration time.Duration
	maxConnections int
	maxFailures    int
	log            logrus.FieldLogger
}

func NewRateLimiter(windowDuration time.Duration, maxConn, maxFail int, log logrus.FieldLogger) *RateLimiter {
	return &RateLimiter{
		connections:    make(map[string]int),
		failures:       make(map[string]int),
		windowStart:    time.Now(),
		windowDuration: windowDuration,
		maxConnections: maxConn,
		maxFailures:    maxFail,
		log:            log,
	}
}

func (r *RateLimiter) AddConnection(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanup()

	count := r.connections[ip]
	if count >= r.maxConnections {
		r.log.WithFields(logrus.Fields{
			"ip":    ip,
			"count": count,
			"max":   r.maxConnections,
		}).Warn("Connection limit exceeded for IP")

		return false
	}

	r.connections[ip]++

	return true
}

func (r *RateLimiter) RemoveConnection(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if count, exists := r.connections[ip]; exists && count > 0 {
		r.connections[ip]--
		if r.connections[ip] == 0 {
			delete(r.connections, ip)
		}
	}
}

func (r *RateLimiter) AddFailure(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanup()

	r.failures[ip]++
	failureCount := r.failures[ip]

	if failureCount >= r.maxFailures {
		r.log.WithFields(logrus.Fields{
			"ip":        ip,
			"failures":  failureCount,
			"threshold": r.maxFailures,
		}).Warn("High failure rate detected for IP")

		return false
	}

	return true
}

func (r *RateLimiter) cleanup() {
	now := time.Now()
	if now.Sub(r.windowStart) > r.windowDuration {
		r.failures = make(map[string]int)
		r.windowStart = now
	}
}
