package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/cart/application"
	"github.com/wyfcoding/financialtrading/internal/cart/domain"
	"github.com/wyfcoding/financialtrading/internal/cart/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/cart/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/cart/config.toml", "path to config file")
	flag.Parse()

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}
	db.AutoMigrate(&domain.Cart{}, &domain.CartItem{})

	repo := mysql.NewCartRepository(db)

	// 创建事件发布者
	eventPublisher := &dummyEventPublisher{}

	appService := application.NewCartApplicationService(repo, eventPublisher)

	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	port := viper.GetString("server.grpc_port")
	lis, _ := net.Listen("tcp", ":"+port)

	go func() { slog.Info("gRPC server", "port", port); grpcSrv.Serve(lis) }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down...")
	grpcSrv.GracefulStop()
}

// dummyEventPublisher 简单的事件发布者实现
type dummyEventPublisher struct{}

// Publish 发布一个普通事件
func (p *dummyEventPublisher) Publish(ctx context.Context, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event", "topic", topic, "key", key, "event", event)
	return nil
}

// PublishInTx 在事务中发布事件
func (p *dummyEventPublisher) PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event in transaction", "topic", topic, "key", key, "event", event)
	return nil
}
