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
	pricing_pb "github.com/wyfcoding/financialtrading/go-api/pricing/v1"
	"github.com/wyfcoding/financialtrading/internal/pricing/application"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/financialtrading/internal/pricing/infrastructure/persistence"
	grpc_server "github.com/wyfcoding/financialtrading/internal/pricing/interfaces/grpc"
	"github.com/wyfcoding/pkg/logging"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/pricing/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := logging.NewLogger("pricing", "main", viper.GetString("log.level"))
	slog.SetDefault(logger.Logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.Price{}, &persistence.PricingResultModel{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := persistence.NewPricingRepository(db)

	// 5. Application
	appService, err := application.NewPricingService(repo, db)
	if err != nil {
		panic(fmt.Sprintf("failed to init pricing service: %v", err))
	}

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	pricingHandler := grpc_server.NewHandler(appService)
	pricing_pb.RegisterPricingServiceServer(grpcSrv, pricingHandler)
	reflection.Register(grpcSrv)

	// 7. Start
	g, ctx := errgroup.WithContext(context.Background())

	grpcPort := viper.GetString("server.grpc_port")
	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
		if err != nil {
			return err
		}
		slog.Info("Starting gRPC server", "port", grpcPort)
		return grpcSrv.Serve(lis)
	})

	// 8. Graceful Shutdown
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-quit:
			slog.Info("shutting down servers...")
		case <-ctx.Done():
			slog.Info("context cancelled, shutting down...")
		}
		grpcSrv.GracefulStop()
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}
