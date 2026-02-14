package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialtrading/go-api/settlement/v1"
	"github.com/wyfcoding/financialtrading/internal/settlement/application"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/settlement/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/settlement/interfaces/grpc"
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
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_settlement?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// 3. Database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(&domain.SettlementInstruction{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 4. Layers
	repo := persistence_mysql.NewSettlementRepo(db)
	nettingRepo := persistence_mysql.NewNettingRepo(db)
	batchRepo := persistence_mysql.NewBatchRepo(db)
	fxRateRepo := persistence_mysql.NewFXRateRepo(db)
	domainSvc := domain.NewSettlementDomainService(nil, nil, nil)
	app := application.NewSettlementAppService(repo, nettingRepo, batchRepo, fxRateRepo, domainSvc, logger)
	svc := grpc_server.NewServer(app)

	// 5. Server
	lis, err := net.Listen("tcp", ":9094")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSettlementServiceServer(s, svc)

	go func() {
		logger.Info("server started", "addr", ":9094")
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
