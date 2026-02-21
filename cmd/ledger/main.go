package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/ledger/v1"
	"github.com/wyfcoding/financialtrading/internal/ledger/application"
	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
	grpc_handler "github.com/wyfcoding/financialtrading/internal/ledger/interfaces/grpc"
	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	repo := domain.NewMemoryLedgerRepository()
	cmdSvc := application.NewLedgerCommandService(repo, logger)
	querySvc := application.NewLedgerQueryService(repo, nil, nil, nil)
	handler := grpc_handler.NewServer(cmdSvc, querySvc)

	addr := os.Getenv("LEDGER_GRPC_ADDR")
	if addr == "" {
		addr = ":9117"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	srv := grpc.NewServer()
	pb.RegisterLedgerServiceServer(srv, handler)
	go func() {
		logger.Info("ledger service started", "grpc_addr", addr)
		if serveErr := srv.Serve(lis); serveErr != nil {
			logger.Error("ledger service exit", "error", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down ledger service...")
	srv.GracefulStop()
	slog.Info("service ledger gracefully stopped")
}
