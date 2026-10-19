package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/metrics"
)

const principalKey = "crosswatc.principal"

// withCorrelation accepts a valid inbound X-Correlation-Id or generates one, and echoes it (spec §39).
func withCorrelation() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(correlation.Header)
		if !correlation.Valid(id) {
			id = correlation.New()
		}
		c.Header(correlation.Header, id)
		c.Request = c.Request.WithContext(correlation.WithID(c.Request.Context(), id))
		c.Next()
	}
}

func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		id, _ := correlation.FromContext(c.Request.Context())
		attrs := []any{"method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(),
			"took", time.Since(start), "correlation.id", id}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "err", c.Errors.String())
			log.Error("request", attrs...)
			return
		}
		log.Info("request", attrs...)
	}
}

func authenticate(a auth.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := a.Authenticate(c.Request)
		if err != nil {
			if !errors.Is(err, auth.ErrUnauthenticated) {
				_ = c.Error(err)
			}
			c.Header("WWW-Authenticate", "Bearer")
			problem(c, http.StatusUnauthorized, "unauthenticated", "Authentication required", "")
			return
		}
		c.Set(principalKey, p)
		c.Next()
	}
}

func principal(c *gin.Context) auth.Principal {
	p, _ := c.Get(principalKey)
	pr, _ := p.(auth.Principal)
	return pr
}

// observe records request latency (SLO: GET /resources/{id} p95 < 400ms) and webhook failures.
func observe() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		code := c.Writer.Status()
		metrics.HTTPDuration.WithLabelValues(c.Request.Method, route, strconv.Itoa(code)).Observe(time.Since(start).Seconds())
		if strings.HasPrefix(route, "/webhooks/") && code >= 400 {
			metrics.WebhookFailures.WithLabelValues(strings.TrimPrefix(route, "/webhooks/"), strconv.Itoa(code)).Inc()
		}
	}
}
