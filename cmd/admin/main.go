package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/admin/application"
	"github.com/wyfcoding/financialtrading/internal/admin/domain"
	"github.com/wyfcoding/financialtrading/internal/admin/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/admin/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/admin/interfaces/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/admin/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.Admin{}, &domain.Role{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	adminRepo := mysql.NewAdminRepository(db)
	roleRepo := mysql.NewRoleRepository(db)

	// Seed SuperAdmin Role & User
	ctx := context.Background()
	superRole, err := roleRepo.GetByName(ctx, "SuperAdmin")
	if err != nil { // Not found
		slog.Info("Seeding SuperAdmin Role")
		superRole = domain.NewRole("SuperAdmin", `["*"]`)
		roleRepo.Save(ctx, superRole)
	}

	_, err = adminRepo.GetByUsername(ctx, "admin")
	if err != nil {
		slog.Info("Seeding admin user")
		// Password "admin" (hashed in reality, plain for mock)
		adminRepo.Save(ctx, domain.NewAdmin("admin", "admin", superRole.ID))
	}

	// 5. Application

	// 创建事件发布者
	eventPublisher := &dummyEventPublisher{}

	appService := application.NewAdminService(adminRepo, roleRepo, eventPublisher)

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	grpcPort := viper.GetString("server.grpc_port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		panic(err)
	}

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	http_server.NewHandler(r, appService)
	httpPort := viper.GetString("server.http_port")
	httpSrv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}

	// 7. Start
	go func() {
		slog.Info("Starting gRPC server", "port", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	go func() {
		slog.Info("Starting HTTP server", "port", httpPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	grpcSrv.GracefulStop()
	httpSrv.Shutdown(ctx)
	slog.Info("Server exiting")
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
