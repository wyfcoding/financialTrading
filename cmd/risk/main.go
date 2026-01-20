package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	risk_pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/risk/interfaces/grpc"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/risk/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := logging.NewLogger("risk", "main", viper.GetString("log.level"))
	slog.SetDefault(logger.Logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.RiskLimit{}, &domain.RiskAssessment{}, &domain.RiskMetrics{}, &domain.RiskAlert{}, &domain.CircuitBreaker{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	limitRepo := mysql.NewRiskLimitRepository(db)
	assessmentRepo := mysql.NewRiskAssessmentRepository(db)
	metricsRepo := mysql.NewRiskMetricsRepository(db)
	alertRepo := mysql.NewRiskAlertRepository(db)
	breakerRepo := mysql.NewCircuitBreakerRepository(db)

	ruleEngine := risk.NewBaseEvaluator(logger.Logger)
	localCache, _ := cache.NewBigCache(time.Minute, 100, logger)

	// 5. Application
	appService := application.NewRiskService(assessmentRepo, metricsRepo, limitRepo, alertRepo, breakerRepo, ruleEngine, localCache)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	riskHandler := grpc_server.NewHandler(appService)
	risk_pb.RegisterRiskServiceServer(grpcSrv, riskHandler)
	reflection.Register(grpcSrv)
	port := viper.GetString("server.grpc_port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	// 7. Start
	go func() {
		slog.Info("Starting gRPC server", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	grpcSrv.GracefulStop()
	slog.Info("Server exiting")
}
