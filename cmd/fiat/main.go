package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/fiat/v1"
	"github.com/wyfcoding/financialtrading/internal/fiat/application"
	persistence "github.com/wyfcoding/financialtrading/internal/fiat/infrastructure/persistence"
	grpc_handler "github.com/wyfcoding/financialtrading/internal/fiat/interfaces/grpc"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("FIAT_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_fiat?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&persistence.ExchangeRateModel{}, &persistence.RateLockModel{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := persistence.NewFiatRepository(db)
	appSvc := application.NewFiatApplicationService(repo)
	handler := grpc_handler.NewFiatHandler(appSvc)

	addr := os.Getenv("FIAT_GRPC_ADDR")
	if addr == "" {
		addr = ":9108"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	srv := grpc.NewServer()
	pb.RegisterFiatServiceServer(srv, handler)

	go func() {
		logger.Info("fiat service started", "grpc_addr", addr)
		if serveErr := srv.Serve(lis); serveErr != nil {
			logger.Error("fiat service exit", "error", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down fiat service...")
	srv.GracefulStop()

	slog.Info("service fiat gracefully stopped")
}
