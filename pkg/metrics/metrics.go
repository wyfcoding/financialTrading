// Package metrics 提供 Prometheus helper，包含常用 counter/gauge/histogram 模板
package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 指标集合
type Metrics struct {
	// HTTP 请求计数
	HTTPRequestsTotal prometheus.Counter
	// HTTP 请求耗时
	HTTPRequestDuration prometheus.Histogram
	// HTTP 请求大小
	HTTPRequestSize prometheus.Histogram
	// HTTP 响应大小
	HTTPResponseSize prometheus.Histogram

	// gRPC 请求计数
	GRPCRequestsTotal prometheus.Counter
	// gRPC 请求耗时
	GRPCRequestDuration prometheus.Histogram

	// 数据库查询计数
	DBQueriesTotal prometheus.Counter
	// 数据库查询耗时
	DBQueryDuration prometheus.Histogram
	// 数据库连接数
	DBConnections prometheus.Gauge

	// Redis 操作计数
	RedisOpsTotal prometheus.Counter
	// Redis 操作耗时
	RedisOpDuration prometheus.Histogram

	// 业务指标
	OrdersTotal     prometheus.Counter
	OrdersActive    prometheus.Gauge
	TradesTotal     prometheus.Counter
	PositionsActive prometheus.Gauge
}

// New 创建指标实例
func New(serviceName string) *Metrics {
	return &Metrics{
		// HTTP 指标
		HTTPRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests",
		}),
		HTTPRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}),
		HTTPRequestSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   []float64{100, 1000, 10000, 100000, 1000000},
		}),
		HTTPResponseSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   []float64{100, 1000, 10000, 100000, 1000000},
		}),

		// gRPC 指标
		GRPCRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "grpc_requests_total",
			Help:      "Total gRPC requests",
		}),
		GRPCRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "grpc_request_duration_seconds",
			Help:      "gRPC request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}),

		// 数据库指标
		DBQueriesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "db_queries_total",
			Help:      "Total database queries",
		}),
		DBQueryDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "db_query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}),
		DBConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "db_connections",
			Help:      "Number of database connections",
		}),

		// Redis 指标
		RedisOpsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "redis_ops_total",
			Help:      "Total Redis operations",
		}),
		RedisOpDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "redis_op_duration_seconds",
			Help:      "Redis operation duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}),

		// 业务指标
		OrdersTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "orders_total",
			Help:      "Total orders created",
		}),
		OrdersActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "orders_active",
			Help:      "Number of active orders",
		}),
		TradesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "trades_total",
			Help:      "Total trades executed",
		}),
		PositionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "trading",
			Subsystem: serviceName,
			Name:      "positions_active",
			Help:      "Number of active positions",
		}),
	}
}

// Register 注册所有指标
func (m *Metrics) Register() error {
	metrics := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.GRPCRequestsTotal,
		m.GRPCRequestDuration,
		m.DBQueriesTotal,
		m.DBQueryDuration,
		m.DBConnections,
		m.RedisOpsTotal,
		m.RedisOpDuration,
		m.OrdersTotal,
		m.OrdersActive,
		m.TradesTotal,
		m.PositionsActive,
	}

	for _, metric := range metrics {
		if err := prometheus.DefaultRegisterer.Register(metric); err != nil {
			logger.Error(context.Background(), "Failed to register metric", "error", err)
			return err
		}
	}

	logger.Info(context.Background(), "Metrics registered successfully")
	return nil
}

// StartHTTPServer 启动 Prometheus HTTP 服务器
func StartHTTPServer(port int, path string) error {
	if path == "" {
		path = "/metrics"
	}

	http.Handle(path, promhttp.Handler())

	addr := fmt.Sprintf(":%d", port)
	logger.Info(context.Background(), "Starting Prometheus HTTP server", "addr", addr, "path", path)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error(context.Background(), "Failed to start Prometheus HTTP server", "error", err)
		}
	}()

	return nil
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// 记录 HTTP 请求
	RecordHTTPRequest(method, path string, statusCode int, duration float64, requestSize, responseSize int64)
	// 记录 gRPC 请求
	RecordGRPCRequest(method string, duration float64)
	// 记录数据库查询
	RecordDBQuery(duration float64)
	// 记录 Redis 操作
	RecordRedisOp(duration float64)
	// 记录订单
	RecordOrder()
	// 更新活跃订单数
	UpdateActiveOrders(count int64)
	// 记录交易
	RecordTrade()
	// 更新活跃持仓数
	UpdateActivePositions(count int64)
}

// DefaultMetricsCollector 默认指标收集器实现
type DefaultMetricsCollector struct {
	metrics *Metrics
}

// NewDefaultMetricsCollector 创建默认指标收集器
func NewDefaultMetricsCollector(metrics *Metrics) *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		metrics: metrics,
	}
}

// RecordHTTPRequest 记录 HTTP 请求
func (dmc *DefaultMetricsCollector) RecordHTTPRequest(method, path string, statusCode int, duration float64, requestSize, responseSize int64) {
	dmc.metrics.HTTPRequestsTotal.Inc()
	dmc.metrics.HTTPRequestDuration.Observe(duration)
	dmc.metrics.HTTPRequestSize.Observe(float64(requestSize))
	dmc.metrics.HTTPResponseSize.Observe(float64(responseSize))
}

// RecordGRPCRequest 记录 gRPC 请求
func (dmc *DefaultMetricsCollector) RecordGRPCRequest(method string, duration float64) {
	dmc.metrics.GRPCRequestsTotal.Inc()
	dmc.metrics.GRPCRequestDuration.Observe(duration)
}

// RecordDBQuery 记录数据库查询
func (dmc *DefaultMetricsCollector) RecordDBQuery(duration float64) {
	dmc.metrics.DBQueriesTotal.Inc()
	dmc.metrics.DBQueryDuration.Observe(duration)
}

// RecordRedisOp 记录 Redis 操作
func (dmc *DefaultMetricsCollector) RecordRedisOp(duration float64) {
	dmc.metrics.RedisOpsTotal.Inc()
	dmc.metrics.RedisOpDuration.Observe(duration)
}

// RecordOrder 记录订单
func (dmc *DefaultMetricsCollector) RecordOrder() {
	dmc.metrics.OrdersTotal.Inc()
}

// UpdateActiveOrders 更新活跃订单数
func (dmc *DefaultMetricsCollector) UpdateActiveOrders(count int64) {
	dmc.metrics.OrdersActive.Set(float64(count))
}

// RecordTrade 记录交易
func (dmc *DefaultMetricsCollector) RecordTrade() {
	dmc.metrics.TradesTotal.Inc()
}

// UpdateActivePositions 更新活跃持仓数
func (dmc *DefaultMetricsCollector) UpdateActivePositions(count int64) {
	dmc.metrics.PositionsActive.Set(float64(count))
}
