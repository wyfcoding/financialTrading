package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"github.com/wyfcoding/financialtrading/internal/auth/infrastructure/persistence/mysql"
	auth_redis "github.com/wyfcoding/financialtrading/internal/auth/infrastructure/persistence/redis"
	grpc_server "github.com/wyfcoding/financialtrading/internal/auth/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/auth/interfaces/http"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

// Publish 实现 domain.EventPublisher 接口
func (m *mockEventPublisher) Publish(ctx context.Context, topic string, key string, event any) error {
	return nil
}

// PublishInTx 实现 domain.EventPublisher 接口
func (m *mockEventPublisher) PublishInTx(ctx context.Context, tx interface{}, topic string, key string, event any) error {
	return nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/auth/config.toml", "path to config file")
	flag.Parse()

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.APIKey{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewUserRepository(db)
	apiKeyRepo := mysql.NewAPIKeyRepository(db)

	// Redis Cache
	redisCfg := &config.RedisConfig{
		Addrs:    []string{viper.GetString("redis.addr")},
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
		PoolSize: viper.GetInt("redis.pool_size"),
	}
	cbCfg := config.CircuitBreakerConfig{
		Interval:    5 * time.Second,
		Timeout:     10 * time.Second,
		MaxRequests: 100,
		Enabled:     true,
	}

	metricsImpl := metrics.NewMetrics("auth-service")
	loggerWrapper := logging.NewLogger("auth", "main", viper.GetString("log.level"))

	redisCache, err := cache.NewRedisCache(redisCfg, cbCfg, loggerWrapper, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}

	sessionRepo := auth_redis.NewSessionRedisRepository(redisCache.GetClient())
	apiKeyRedisRepo := auth_redis.NewAPIKeyRedisRepository(redisCache.GetClient())

	keySvc := application.NewAPIKeyService(apiKeyRepo)

	// 创建事件发布者（使用空实现）
	eventPublisher := &mockEventPublisher{}

	appService := application.NewAuthService(repo, apiKeyRepo, apiKeyRedisRepo, sessionRepo, keySvc, eventPublisher)

	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	grpcPort := viper.GetString("server.grpc_port")
	lis, _ := net.Listen("tcp", ":"+grpcPort)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	http_server.NewHandler(r, appService)
	httpPort := viper.GetString("server.http_port")
	httpSrv := &http.Server{Addr: ":" + httpPort, Handler: r}

	go func() { slog.Info("gRPC server", "port", grpcPort); grpcSrv.Serve(lis) }()
	go func() { slog.Info("HTTP server", "port", httpPort); httpSrv.ListenAndServe() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down...")
	grpcSrv.GracefulStop()
}
