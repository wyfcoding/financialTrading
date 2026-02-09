package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/custody/v1"
	"github.com/wyfcoding/financialtrading/internal/custody/application"
	"github.com/wyfcoding/financialtrading/internal/custody/domain"
	"github.com/wyfcoding/financialtrading/internal/custody/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/custody/interfaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 初始化日志
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("starting custody service")

	// 2. 初始化数据库
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
	db.AutoMigrate(&domain.AssetVault{}, &domain.CustodyTransfer{}, &domain.CorpAction{}, &domain.CorpActionExecution{})

	// 3. 依赖注入
	vaultRepo := infrastructure.NewCustodyRepository(db)
	actionRepo := infrastructure.NewCorpActionRepository(db)
	appService := application.NewCustodyApplicationService(vaultRepo, actionRepo, logger)
	handler := interfaces.NewCustodyHandler(appService)

	// 4. 启动 gRPC 服务
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50053"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCustodyServiceServer(grpcServer, handler)
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

	logger.Info("shutting down custody server...")
	grpcServer.GracefulStop()
	logger.Info("server exited")
}
