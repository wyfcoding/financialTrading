package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/fixgateway/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/fix/application"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/fix/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/fix/interfaces/grpc"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("FIXGATEWAY_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_fixgateway?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&persistence_mysql.FixSessionModel{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := persistence_mysql.NewGormFixRepository(db)
	app := application.NewFixApplicationService(repo, nil, logger)
	svc := grpc_server.NewServer(app)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.HeartbeatMonitor(ctx)

	lis, err := net.Listen("tcp", ":9098")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterFixGatewayServiceServer(s, svc)

	go func() {
		logger.Info("server started", "addr", ":9098")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	cancel()
	s.GracefulStop()

	if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
		_ = sqlDB.Close()
	}
}
