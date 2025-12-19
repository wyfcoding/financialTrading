package main

import (
	"log/slog"

	"time"

	"github.com/wyfcoding/pkg/grpcclient"

	"github.com/gin-gonic/gin"
	pb "github.com/wyfcoding/financialTrading/go-api/risk"
	"github.com/wyfcoding/financialTrading/internal/risk/application"
	"github.com/wyfcoding/financialTrading/internal/risk/infrastructure/repository"
	grpchandler "github.com/wyfcoding/financialTrading/internal/risk/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/risk/interfaces/http"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/cache"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/databases"
	"github.com/wyfcoding/pkg/limiter"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/middleware"
	"google.golang.org/grpc"
)

type AppContext struct {
	AppService *application.RiskApplicationService
	Limiter    limiter.Limiter
	Config     *configpkg.Config
	Clients    *ServiceClients
}

type ServiceClients struct {
	Account  *grpc.ClientConn
	Position *grpc.ClientConn
	Order    *grpc.ClientConn
}

const BootstrapName = "risk"

func main() {
	app.NewBuilder(BootstrapName).
		WithConfig(&configpkg.Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		WithGin(registerGin).
		WithGinMiddleware(middleware.CORS()).
		Build().
		Run()
}

func registerGRPC(s *grpc.Server, srv interface{}) {
	ctx := srv.(*AppContext)
	handler := grpchandler.NewGRPCHandler(ctx.AppService)
	pb.RegisterRiskServiceServer(s, handler)
	slog.Default().Info("gRPC server registered", "service", BootstrapName)
}

func registerGin(e *gin.Engine, srv interface{}) {
	ctx := srv.(*AppContext)
	e.Use(middleware.RateLimitWithLimiter(ctx.Limiter))
	httpHandler := httphandler.NewRiskHandler(ctx.AppService)
	httpHandler.RegisterRoutes(e)
	e.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   BootstrapName,
			"timestamp": time.Now().Unix(),
		})
	})
	slog.Default().Info("HTTP routes registered", "service", BootstrapName)
}

func initService(cfg interface{}, m *metrics.Metrics) (interface{}, func(), error) {
	c := cfg.(*configpkg.Config)
	slog.Info("initializing service dependencies...")
	db, err := databases.NewDB(c.Data.Database, logging.Default())
	if err != nil {
		return nil, nil, err
	}
	redisCache, err := cache.NewRedisCache(c.Data.Redis)
	if err != nil {
		return nil, nil, err
	}
	rateLimiter := limiter.NewRedisLimiter(redisCache.GetClient(), c.RateLimit.Rate, time.Second)
	assessmentRepo := repository.NewRiskAssessmentRepository(db)
	metricsRepo := repository.NewRiskMetricsRepository(db)
	limitRepo := repository.NewRiskLimitRepository(db)
	alertRepo := repository.NewRiskAlertRepository(db)
	appService := application.NewRiskApplicationService(assessmentRepo, metricsRepo, limitRepo, alertRepo)

	// Downstream Clients
	clients := &ServiceClients{}
	clientCleanup, err := grpcclient.InitServiceClients(c.Services, clients)
	if err != nil {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		redisCache.Close()
		return nil, nil, err
	}
	cleanup := func() {
		slog.Info("cleaning up resources...")
		clientCleanup()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		redisCache.Close()
	}
	return &AppContext{
		AppService: appService,
		Limiter:    rateLimiter,
		Config:     c,
		Clients:    clients,
	}, cleanup, nil
}
