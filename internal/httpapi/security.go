package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// securityHeaders hardens every API response (spec §49).
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}

// userLimiter rate-limits mutations per authenticated user.
type userLimiter struct {
	mu      sync.Mutex
	every   time.Duration
	burst   int
	byUser  map[string]*rate.Limiter
	lastGC  time.Time
	lastUse map[string]time.Time
}

func newUserLimiter(perMinute, burst int) *userLimiter {
	if perMinute <= 0 {
		return nil
	}
	return &userLimiter{every: time.Minute / time.Duration(perMinute), burst: burst,
		byUser: map[string]*rate.Limiter{}, lastUse: map[string]time.Time{}, lastGC: time.Now()}
}

func (l *userLimiter) allow(user string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.lastGC) > 10*time.Minute {
		for u, t := range l.lastUse {
			if now.Sub(t) > 10*time.Minute {
				delete(l.byUser, u)
				delete(l.lastUse, u)
			}
		}
		l.lastGC = now
	}
	lim, ok := l.byUser[user]
	if !ok {
		lim = rate.NewLimiter(rate.Every(l.every), l.burst)
		l.byUser[user] = lim
	}
	l.lastUse[user] = now
	r := lim.ReserveN(now, 1)
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now)
		return false, d
	}
	return true, 0
}

func mutationLimiter(perMinute int) *userLimiter {
	switch {
	case perMinute < 0:
		return nil
	case perMinute == 0:
		perMinute = 10
	}
	return newUserLimiter(perMinute, 5)
}

// limitMutations applies the per-user limiter to non-GET requests.
func limitMutations(l *userLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		if ok, wait := l.allow(principal(c).User); !ok {
			c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			problem(c, http.StatusTooManyRequests, "rate-limited", "Too many change requests", "")
			return
		}
		c.Next()
	}
}
