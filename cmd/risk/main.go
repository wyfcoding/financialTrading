package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	pb "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/mysql"
	grpchandler "github.com/wyfcoding/financialtrading/internal/risk/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialtrading/internal/risk/interfaces/http"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/cache"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/databases"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/limiter"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/middleware"
)

// BootstrapName 服务标识。
const BootstrapName = "risk"

// Config 扩展配置。
type Config struct {
	configpkg.Config `mapstructure:",squash"`
}

// AppContext 应用上下文。
type AppContext struct {
	Config     *Config
	AppService *application.RiskApplicationService
	Clients    *ServiceClients
	Metrics    *metrics.Metrics
	Limiter    limiter.Limiter
}

// ServiceClients 下游微服务。
type ServiceClients struct {
	// 目前无强依赖
}

func main() {
	if err := app.NewBuilder(BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		WithGin(registerGin).
		WithGinMiddleware(
			middleware.MetricsMiddleware(),
			middleware.CORS(),
		).
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

func registerGRPC(s *grpc.Server, srv any) {
	ctx := srv.(*AppContext)
	pb.RegisterRiskServiceServer(s, grpchandler.NewGRPCHandler(ctx.AppService))
}

func registerGin(e *gin.Engine, srv any) {
	ctx := srv.(*AppContext)

	if ctx.Config.Server.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 1. 系统路由组 (跳过限流)
	sys := e.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) {
			response.SuccessWithRawData(c, gin.H{
				"status":    "UP",
				"service":   BootstrapName,
				"timestamp": time.Now().Unix(),
			})
		})
		sys.GET("/ready", func(c *gin.Context) {
			response.SuccessWithRawData(c, gin.H{"status": "READY"})
		})
	}

	if ctx.Config.Metrics.Enabled {
		e.GET(ctx.Config.Metrics.Path, gin.WrapH(ctx.Metrics.Handler()))
	}

	// 2. 业务限流
	e.Use(middleware.RateLimitWithLimiter(ctx.Limiter))

	// 3. 业务路由
	httpHandler := httphandler.NewRiskHandler(ctx.AppService)
	httpHandler.RegisterRoutes(e)

	slog.Info("HTTP service configured", "service", BootstrapName)
}

func initService(cfg any, m *metrics.Metrics) (any, func(), error) {
	c := cfg.(*Config)
	bootLog := slog.With("module", "bootstrap")
	logger := logging.Default()

	// 1. 基础设施
	db, err := databases.NewDB(c.Data.Database, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("database init error: %w", err)
	}

	redisCache, err := cache.NewRedisCache(c.Data.Redis)
	if err != nil {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		return nil, nil, fmt.Errorf("redis init error: %w", err)
	}

	// 2. 限流治理
	rateLimiter := limiter.NewRedisLimiter(redisCache.GetClient(), c.RateLimit.Rate, time.Second)

	// 3. 仓储层初始化
	bootLog.Info("initializing risk repositories...")
	assessmentRepo := mysql.NewRiskAssessmentRepository(db)
	metricsRepo := mysql.NewRiskMetricsRepository(db)
	limitRepo := mysql.NewRiskLimitRepository(db)
	alertRepo := mysql.NewRiskAlertRepository(db)

	// 4. 应用服务装配
	appService := application.NewRiskApplicationService(assessmentRepo, metricsRepo, limitRepo, alertRepo)

	// 5. 下游客户端
	clients := &ServiceClients{}
	clientCleanup, err := grpcclient.InitClients(c.Services, clients)
	if err != nil {
		redisCache.Close()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		return nil, nil, fmt.Errorf("clients init error: %w", err)
	}

	cleanup := func() {
		bootLog.Info("performing graceful shutdown...")
		clientCleanup()
		redisCache.Close()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}

	return &AppContext{
		Config:     c,
		AppService: appService,
		Clients:    clients,
		Metrics:    m,
		Limiter:    rateLimiter,
	}, cleanup, nil
}
