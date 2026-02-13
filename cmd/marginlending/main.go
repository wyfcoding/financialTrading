package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/marginlending/v1"
	"github.com/wyfcoding/financialtrading/internal/marginlending/application"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/marginlending/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/marginlending/interfaces/grpc"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Config (Simplified)
	dsn := os.Getenv("To be replaced by config loader")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_margin?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// 3. Database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(&persistence_mysql.MarginAccountModel{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 4. Layers
	repo := persistence_mysql.NewMarginRepo(db)
	app := application.NewMarginAppService(repo, logger)
	svc := grpc_server.NewServer(app)

	// 5. Server
	lis, err := net.Listen("tcp", ":9099")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMarginLendingServiceServer(s, svc)

	go func() {
		logger.Info("server started", "addr", ":9099")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")
	s.GracefulStop()
}
