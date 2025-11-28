package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/ratelimit"
)

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(limiter ratelimit.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Use IP as the key for now. Can be extended to use UserID or API Key.
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())
		limit := ratelimit.Limit{
			Rate:   cfg.QPS,
			Period: time.Second,
			Burst:  cfg.Burst,
		}

		res, err := limiter.Allow(c.Request.Context(), key, limit)
		if err != nil {
			// Fail open if rate limiter fails
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Burst))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(int64(res.ResetAfter/time.Second), 10))

		if !res.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(res.RetryAfter/time.Second), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too Many Requests",
				"retry_after": res.RetryAfter.String(),
			})
			return
		}

		c.Next()
	}
}
