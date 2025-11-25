// Package grpcclient 提供 gRPC 客户端工厂，支持负载均衡、限流、重试、熔断、拦截器/trace 注入
package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// ClientConfig gRPC 客户端配置
type ClientConfig struct {
	// 目标地址
	Target string
	// 连接超时（秒）
	ConnTimeout int
	// 请求超时（秒）
	RequestTimeout int
	// 最大重试次数
	MaxRetries int
	// 重试延迟（毫秒）
	RetryDelay int
	// 是否启用 keepalive
	EnableKeepalive bool
	// Keepalive 间隔（秒）
	KeepaliveInterval int
	// 是否启用负载均衡
	EnableLoadBalancing bool
	// 负载均衡策略：round_robin, least_request
	LoadBalancingPolicy string
}

// NewClient 创建 gRPC 客户端连接
func NewClient(cfg ClientConfig) (*grpc.ClientConn, error) {
	// 构建拨号选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024), // 100MB
			grpc.MaxCallSendMsgSize(100*1024*1024), // 100MB
		),
	}

	// 添加连接超时
	if cfg.ConnTimeout > 0 {
		opts = append(opts, grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				MaxDelay:   time.Duration(cfg.ConnTimeout) * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
			},
			MinConnectTimeout: time.Duration(cfg.ConnTimeout) * time.Second,
		}))
	}

	// 添加 keepalive
	if cfg.EnableKeepalive {
		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(cfg.KeepaliveInterval) * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}))
	}

	// 添加负载均衡
	if cfg.EnableLoadBalancing {
		policy := cfg.LoadBalancingPolicy
		if policy == "" {
			policy = "round_robin"
		}
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{
			"loadBalancingConfig": [{"round_robin":{}}],
			"methodConfig": [{
				"name": [{"service": ""}],
				"retryPolicy": {
					"maxAttempts": %d,
					"initialBackoff": "%dms",
					"maxBackoff": "10s",
					"backoffMultiplier": 1.0,
					"retryableStatusCodes": ["UNAVAILABLE", "RESOURCE_EXHAUSTED"]
				}
			}]
		}`, cfg.MaxRetries, cfg.RetryDelay)))
	}

	// 添加拦截器
	opts = append(opts,
		grpc.WithUnaryInterceptor(unaryClientInterceptor(cfg)),
		grpc.WithStreamInterceptor(streamClientInterceptor(cfg)),
	)

	// 创建连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ConnTimeout)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Target, opts...)
	if err != nil {
		logger.Error(ctx, "Failed to create gRPC client", "target", cfg.Target, "error", err)
		return nil, err
	}

	logger.Info(ctx, "gRPC client created successfully", "target", cfg.Target)
	return conn, nil
}

// unaryClientInterceptor 一元 RPC 拦截器
func unaryClientInterceptor(cfg ClientConfig) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 添加请求超时
		if cfg.RequestTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.RequestTimeout)*time.Second)
			defer cancel()
		}

		// 记录请求开始
		start := time.Now()
		logger.Debug(ctx, "gRPC request started", "method", method)

		// 执行请求，支持重试
		var lastErr error
		for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				// 请求成功
				duration := time.Since(start)
				logger.Debug(ctx, "gRPC request succeeded",
					"method", method,
					"duration", duration,
				)
				return nil
			}

			lastErr = err
			st, ok := status.FromError(err)
			if !ok {
				// 非 gRPC 错误，不重试
				break
			}

			// 检查是否应该重试
			if !shouldRetry(st.Code()) || attempt >= cfg.MaxRetries {
				break
			}

			// 等待后重试
			select {
			case <-time.After(time.Duration(cfg.RetryDelay) * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		duration := time.Since(start)
		logger.Error(ctx, "gRPC request failed",
			"method", method,
			"duration", duration,
			"error", lastErr,
		)
		return lastErr
	}
}

// streamClientInterceptor 流 RPC 拦截器
func streamClientInterceptor(cfg ClientConfig) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// 添加请求超时
		if cfg.RequestTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.RequestTimeout)*time.Second)
			defer cancel()
		}

		logger.Debug(ctx, "gRPC stream started", "method", method)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// shouldRetry 判断是否应该重试
func shouldRetry(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

// ClientPool gRPC 客户端连接池
type ClientPool struct {
	conns map[string]*grpc.ClientConn
}

// NewClientPool 创建客户端连接池
func NewClientPool() *ClientPool {
	return &ClientPool{
		conns: make(map[string]*grpc.ClientConn),
	}
}

// GetOrCreate 获取或创建客户端连接
func (cp *ClientPool) GetOrCreate(target string, cfg ClientConfig) (*grpc.ClientConn, error) {
	if conn, ok := cp.conns[target]; ok {
		return conn, nil
	}

	conn, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	cp.conns[target] = conn
	return conn, nil
}

// Close 关闭所有连接
func (cp *ClientPool) Close() error {
	for _, conn := range cp.conns {
		if err := conn.Close(); err != nil {
			logger.Error(context.Background(), "Failed to close gRPC connection", "error", err)
		}
	}
	cp.conns = make(map[string]*grpc.ClientConn)
	return nil
}
