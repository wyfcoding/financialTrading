package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	portfoliov1 "github.com/wyfcoding/financialtrading/go-api/portfolio/v1"
	"github.com/wyfcoding/financialtrading/internal/portfolio/application"
	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
	"github.com/wyfcoding/financialtrading/internal/portfolio/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/portfolio/interfaces"
	"github.com/wyfcoding/pkg/config"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := &config.ServerConfig{}
	cfg.Name = "portfolio-service"
	cfg.GRPC.Addr = ":9012"

	// DB 连接
	dsn := "root:root@tcp(127.0.0.1:3306)/financial_portfolio?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移
	_ = db.AutoMigrate(&domain.PortfolioSnapshot{}, &domain.UserPerformance{})

	// 依赖注入
	repo := infrastructure.NewPortfolioRepository(db)
	app := application.NewPortfolioService(repo)
	handler := interfaces.NewPortfolioHandler(app, repo)

	// gRPC Server
	lis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	portfoliov1.RegisterPortfolioServiceServer(s, handler)

	fmt.Printf("server listening at %v\n", lis.Addr())

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
