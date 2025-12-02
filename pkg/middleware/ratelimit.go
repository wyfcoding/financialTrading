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

// RateLimitMiddleware 创建 Gin 限流中间件
// limiter: 限流器接口实现
// cfg: 限流配置
func RateLimitMiddleware(limiter ratelimit.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用限流，直接放行
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 使用 IP 作为限流键。可以扩展为使用 UserID 或 API Key。
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
