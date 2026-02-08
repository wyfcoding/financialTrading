package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	derivativesv1 "github.com/wyfcoding/financialtrading/go-api/derivatives/v1"
	"github.com/wyfcoding/financialtrading/internal/derivatives/application"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
	"github.com/wyfcoding/financialtrading/internal/derivatives/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/derivatives/interfaces"
	"github.com/wyfcoding/pkg/config"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := &config.ServerConfig{}
	cfg.Name = "derivatives-service"
	cfg.GRPC.Addr = ":9011"

	// DB 连接
	dsn := "root:root@tcp(127.0.0.1:3306)/financial_derivatives?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移
	_ = db.AutoMigrate(&domain.Contract{})

	// 依赖注入
	repo := infrastructure.NewContractRepository(db)
	app := application.NewDerivativesService(repo)
	handler := interfaces.NewDerivativesHandler(app, repo)

	// gRPC Server
	lis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	derivativesv1.RegisterDerivativesServiceServer(s, handler)

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
