package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/wyfcoding/financialTrading/internal/primebrokerage/application"
	"github.com/wyfcoding/financialTrading/internal/primebrokerage/domain"
	"github.com/wyfcoding/financialTrading/internal/primebrokerage/infrastructure"
	"github.com/wyfcoding/financialTrading/internal/primebrokerage/interfaces"
	pb "github.com/wyfcoding/financialtrading/go-api/primebrokerage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 初始化日志 (基于 slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("starting prime brokerage service")

	// 2. 初始化数据库 (GORM)
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root1234@tcp(127.0.0.1:3306)/financial_trading?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	// 自动迁移
	db.AutoMigrate(&domain.ClearingSeat{}, &domain.SecurityPool{}, &domain.SecurityLoan{})

	// 3. 依赖注入
	repo := infrastructure.NewPrimeBrokerageRepository(db)

	// 这里模拟初始化一些席位
	initSeats(repo)

	router := &domain.DefaultSeatRouter{
		Seats: make([]*domain.ClearingSeat, 0),
	}
	// 从数据库加载席位 (逻辑简化)
	seats, _ := repo.ListSeats(context.Background(), "NYSE")
	router.Seats = append(router.Seats, seats...)

	poolService := domain.NewSecurityPoolService(repo)
	appService := application.NewPrimeBrokerageApplicationService(repo, router, poolService, logger)
	handler := interfaces.NewPrimeBrokerageHandler(appService)

	// 4. 启动 gRPC 服务
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPrimeBrokerageServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	// 5. 优雅关停
	go func() {
		logger.Info("gRPC server listening on", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	grpcServer.GracefulStop()
	logger.Info("server exited")
}

func initSeats(repo domain.PrimeBrokerageRepository) {
	// 单元测试或初始化逻辑
	ctx := context.Background()
	repo.SaveSeat(ctx, &domain.ClearingSeat{
		ID:           "SEAT-001",
		Name:         "Goldman Sachs NYSE",
		ExchangeCode: "NYSE",
		Capacity:     1000000,
		Latency:      5,
		CostPerTrade: 0.01,
		Status:       "ACTIVE",
	})
}
