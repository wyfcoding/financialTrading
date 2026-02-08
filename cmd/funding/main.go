package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	fundingv1 "github.com/wyfcoding/financialtrading/go-api/funding/v1"
	"github.com/wyfcoding/financialtrading/internal/funding/application"
	"github.com/wyfcoding/financialtrading/internal/funding/domain"
	"github.com/wyfcoding/financialtrading/internal/funding/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/funding/interfaces"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	cfg := &config.Config{}
	if err := config.Load("configs/funding/config.toml", cfg); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	logger := logging.NewLogger(cfg.Server.Name, "main", cfg.Log.Level)

	// 3. 初始化指标
	m := metrics.NewMetrics(cfg.Server.Name)

	// 4. 初始化数据库
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, m)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// 5. 自动迁移 (使用 Unwrap 获取原始 *gorm.DB)
	if err := db.DB.AutoMigrate(&domain.MarginLoan{}, &domain.FundingRate{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	// 6. 依赖注入
	repo := infrastructure.NewGormFundingRepository(db.DB)
	app := application.NewFundingService(repo)
	handler := interfaces.NewFundingHandler(app)

	// 7. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPC.Port))
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	fundingv1.RegisterFundingServiceServer(s, handler)
	reflection.Register(s)

	fmt.Printf("%s listening at %v\n", cfg.Server.Name, lis.Addr())

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()

	// 7. 优雅关停
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	s.GracefulStop()
	logger.Info("server stopped")
}
