package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/marketsurveillance/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/application"
	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/marketsurveillance/infrastructure/persistence/mysql"
	grpc_handler "github.com/wyfcoding/financialtrading/internal/marketsurveillance/interfaces/grpc"
	"github.com/wyfcoding/pkg/messagequeue"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("MARKET_SURVEILLANCE_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_marketsurveillance?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := persistence_mysql.AutoMigrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	alertRepo := persistence_mysql.NewAlertRepository(db)
	ruleRepo := persistence_mysql.NewRuleRepository(db)
	userScoreRepo := persistence_mysql.NewUserScoreRepository(db)
	eventRepo := persistence_mysql.NewOrderEventRepository(db)
	engine := domain.NewDetectionEngine(logger)
	publisher := noopEventPublisher{}

	cmdSvc := application.NewCommandService(alertRepo, ruleRepo, userScoreRepo, eventRepo, engine, publisher, logger)
	querySvc := application.NewQueryService(alertRepo, ruleRepo, userScoreRepo, eventRepo, engine, logger)
	handler := grpc_handler.NewMarketSurveillanceHandler(cmdSvc, querySvc, logger)

	addr := os.Getenv("MARKET_SURVEILLANCE_GRPC_ADDR")
	if addr == "" {
		addr = ":9107"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	srv := grpc.NewServer()
	pb.RegisterMarketSurveillanceServiceServer(srv, handler)

	go func() {
		logger.Info("market surveillance service started", "grpc_addr", addr)
		if serveErr := srv.Serve(lis); serveErr != nil {
			logger.Error("market surveillance service exit", "error", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down market surveillance service...")
	srv.GracefulStop()

	logger.Info("service marketsurveillance gracefully stopped")
}

type noopEventPublisher struct{}

func (noopEventPublisher) Publish(_ context.Context, _ string, _ string, _ any) error { return nil }

func (noopEventPublisher) PublishInTx(_ context.Context, _ any, _ string, _ string, _ any) error {
	return nil
}

var _ messagequeue.EventPublisher = (*noopEventPublisher)(nil)
