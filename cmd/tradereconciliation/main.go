package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	tradereconciliationv1 "github.com/wyfcoding/financialtrading/go-api/tradereconciliation/v1"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/application"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/domain"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/interfaces"
	"github.com/wyfcoding/pkg/config"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := &config.ServerConfig{}
	cfg.Name = "tradereconciliation-service"
	cfg.GRPC.Addr = ":9013"

	// DB 连接
	dsn := "root:root@tcp(127.0.0.1:3306)/financial_reconciliation?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移
	_ = db.AutoMigrate(&domain.ReconciliationTask{}, &domain.Discrepancy{})

	// 依赖注入
	repo := infrastructure.NewReconciliationRepository(db)
	app := application.NewReconciliationService(repo)
	handler := interfaces.NewReconciliationHandler(app, repo)

	// gRPC Server
	lis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	tradereconciliationv1.RegisterTradeReconciliationServiceServer(s, handler)

	fmt.Printf("%s listening at %v\n", cfg.Name, lis.Addr())

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.GracefulStop()
}
