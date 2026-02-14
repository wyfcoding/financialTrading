package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/sor/v1"
	"github.com/wyfcoding/financialtrading/internal/sor/application"
	"github.com/wyfcoding/financialtrading/internal/sor/domain"
	grpc_server "github.com/wyfcoding/financialtrading/internal/sor/interfaces/grpc"
	"google.golang.org/grpc"
)

func main() {
	// 1. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Config (Simplified)
	// SOR might not need DB for now, purely calculation based on market data feeds

	// 3. Domain Engine
	engine := domain.NewDefaultSOREngine()

	// 4. Layers
	app := application.NewSORApplicationService(engine, nil, nil, nil, logger)
	svc := grpc_server.NewServer(app)

	// 5. Server
	lis, err := net.Listen("tcp", ":9093")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSOREngineServiceServer(s, svc)

	go func() {
		logger.Info("server started", "addr", ":9093")
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
