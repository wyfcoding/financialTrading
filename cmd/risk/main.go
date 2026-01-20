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

	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/risk/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/risk/config.toml", "path to config file")
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
	if err := db.AutoMigrate(&domain.RiskLimit{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewRiskLimitRepository(db)

	// Seed Default Risk Limit for Test User
	ctx := context.Background()
	testUserID := "user-123" // Example User
	_, err = repo.GetByUserID(ctx, testUserID)
	if err != nil { // Not found or error
		slog.Info("Seeding limits for test user", "user_id", testUserID)
		repo.Save(ctx, &domain.RiskLimit{
			UserID:       testUserID,
			LimitType:    "ORDER_SIZE",
			LimitValue:   decimal.NewFromFloat(1000.0),
			CurrentValue: decimal.Zero,
			IsExceeded:   false,
		})
	}

	// 5. Application
	appService := application.NewRiskApplicationService(repo)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	port := viper.GetString("server.grpc_port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	// 7. Start
	go func() {
		slog.Info("Starting gRPC server", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	grpcSrv.GracefulStop()
	slog.Info("Server exiting")
}
