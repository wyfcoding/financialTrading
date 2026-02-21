package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/marginlending/v1"
	"github.com/wyfcoding/financialtrading/internal/margin/application"
	"github.com/wyfcoding/financialtrading/internal/margin/domain"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/margin/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/margin/interfaces/grpc"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("MARGIN_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_margin?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&domain.MarginAccount{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := persistence_mysql.NewMarginRepo(db)
	app := application.NewMarginAppService(repo, logger)
	svc := grpc_server.NewServer(app)

	addr := os.Getenv("MARGIN_GRPC_ADDR")
	if addr == "" {
		addr = ":9098"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMarginLendingServiceServer(s, svc)

	go func() {
		logger.Info("server started", "addr", addr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")
	s.GracefulStop()
}
