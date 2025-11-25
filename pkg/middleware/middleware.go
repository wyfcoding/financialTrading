// Package middleware 提供 Gin 与 gRPC 的通用中间件（日志、trace、panic recover、限流、鉴权）
package middleware

import (
	"context"
	"time"

	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestIDKey context key for request ID
const RequestIDKey = "request_id"

// TraceIDKey context key for trace ID
const TraceIDKey = "trace_id"

// SpanIDKey context key for span ID
const SpanIDKey = "span_id"

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	traceIDContextKey   contextKey = "trace_id"
	spanIDContextKey    contextKey = "span_id"
)

// GinLoggingMiddleware Gin 日志中间件
func GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成 request ID 和 trace ID
		requestID := uuid.New().String()
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		spanID := uuid.New().String()

		// 将 ID 存储到 context
		c.Set(RequestIDKey, requestID)
		c.Set(TraceIDKey, traceID)
		c.Set(SpanIDKey, spanID)

		// 记录请求开始
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 创建带有 trace info 的 context
		ctx := context.WithValue(c.Request.Context(), traceIDContextKey, traceID)
		ctx = context.WithValue(ctx, spanIDContextKey, spanID)
		ctx = context.WithValue(ctx, requestIDContextKey, requestID)

		logger.Info(ctx, "HTTP request started",
			"request_id", requestID,
			"method", method,
			"path", path,
			"client_ip", clientIP,
		)

		// 处理请求
		c.Next()

		// 记录响应
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		logger.Info(ctx, "HTTP request completed",
			"request_id", requestID,
			"method", method,
			"path", path,
			"status_code", statusCode,
			"response_size", responseSize,
			"duration", duration,
		)
	}
}

// GinRecoveryMiddleware Gin panic 恢复中间件
func GinRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get(RequestIDKey)
				traceID, _ := c.Get(TraceIDKey)
				spanID, _ := c.Get(SpanIDKey)

				ctx := context.WithValue(c.Request.Context(), traceIDContextKey, traceID)
				ctx = context.WithValue(ctx, spanIDContextKey, spanID)
				ctx = context.WithValue(ctx, requestIDContextKey, requestID)

				logger.Error(ctx, "HTTP request panicked",
					"request_id", requestID,
					"panic", err,
				)

				c.JSON(500, gin.H{
					"error":      "Internal server error",
					"request_id": requestID,
				})
			}
		}()
		c.Next()
	}
}

// GinCORSMiddleware Gin CORS 中间件
func GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Trace-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GRPCLoggingInterceptor gRPC 日志拦截器
func GRPCLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 生成 request ID 和 trace ID
		requestID := uuid.New().String()
		traceID := extractTraceID(ctx)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		spanID := uuid.New().String()

		// 将 ID 存储到 context
		ctx = context.WithValue(ctx, requestIDContextKey, requestID)
		ctx = context.WithValue(ctx, traceIDContextKey, traceID)
		ctx = context.WithValue(ctx, spanIDContextKey, spanID)

		// 记录请求开始
		start := time.Now()
		method := info.FullMethod

		logger.Info(ctx, "gRPC request started",
			"request_id", requestID,
			"method", method,
		)

		// 处理请求
		resp, err := handler(ctx, req)

		// 记录响应
		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error(ctx, "gRPC request failed",
				"request_id", requestID,
				"method", method,
				"error_code", st.Code().String(),
				"error_message", st.Message(),
				"duration", duration,
			)
		} else {
			logger.Info(ctx, "gRPC request completed",
				"request_id", requestID,
				"method", method,
				"duration", duration,
			)
		}

		return resp, err
	}
}

// GRPCRecoveryInterceptor gRPC panic 恢复拦截器
func GRPCRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if err := recover(); err != nil {
				requestID := ctx.Value(requestIDContextKey)

				logger.Error(ctx, "gRPC request panicked",
					"request_id", requestID,
					"method", info.FullMethod,
					"panic", err,
				)
			}
		}()
		return handler(ctx, req)
	}
}

// extractTraceID 从 context 中提取 trace ID
func extractTraceID(ctx context.Context) string {
	// 尝试从 context value 获取
	if traceID, ok := ctx.Value(traceIDContextKey).(string); ok {
		return traceID
	}
	// TODO: 尝试从 metadata 获取
	return ""
}

// RateLimitMiddleware 限流中间件（基于令牌桶算法）
type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(rl.maxTokens, rl.tokens+elapsed*rl.refillRate)
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// min 返回两个数中的最小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GinRateLimitMiddleware Gin 限流中间件
func GinRateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(429, gin.H{
				"error": "Too many requests",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GRPCRateLimitInterceptor gRPC 限流拦截器
func GRPCRateLimitInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
