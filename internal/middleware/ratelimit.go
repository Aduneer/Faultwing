package middleware

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type projectBucket struct {
	tokens     float64
	lastRefill time.Time
}

type ProjectRateLimiter struct {
	mu              sync.Mutex
	tokensPerSecond float64
	burst           float64
	buckets         map[int64]*projectBucket
}

func NewProjectRateLimiter(eventsPerMinute, burst int) *ProjectRateLimiter {
	return &ProjectRateLimiter{
		tokensPerSecond: float64(eventsPerMinute) / 60,
		burst:           float64(burst),
		buckets:         make(map[int64]*projectBucket),
	}
}

func RateLimitByProject(limiter *ProjectRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project, ok := ProjectFromContext(r.Context())
		if !ok {
			writeRateLimitError(w, http.StatusInternalServerError, "authenticated project is missing")
			return
		}

		allowed, retryAfter := limiter.allow(project.ID, time.Now())
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeRateLimitError(w, http.StatusTooManyRequests, "event rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *ProjectRateLimiter) allow(projectID int64, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[projectID]
	if !ok {
		bucket = &projectBucket{tokens: l.burst, lastRefill: now}
		l.buckets[projectID] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(l.burst, bucket.tokens+elapsed*l.tokensPerSecond)
	bucket.lastRefill = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}

	waitSeconds := (1 - bucket.tokens) / l.tokensPerSecond
	return false, max(1, int(math.Ceil(waitSeconds)))
}

func writeRateLimitError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
