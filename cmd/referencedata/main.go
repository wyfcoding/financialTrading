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

	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"github.com/wyfcoding/financialtrading/internal/referencedata/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/referencedata/interfaces/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/referencedata/config.toml", "path to config file")
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
	if err := db.AutoMigrate(&domain.Instrument{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewReferenceRepository(db)

	// Seed Data (if empty)
	ctx := context.Background()
	instr, _ := repo.GetInstrument(ctx, "BTCUSD")
	if instr.Symbol == "" {
		slog.Info("Seeding BTCUSD")
		repo.Save(ctx, domain.NewInstrument("BTCUSD", "BTC", "USD", 0.01, 0.001, domain.Spot))
	}

	instr2, _ := repo.GetInstrument(ctx, "ETHUSD")
	if instr2.Symbol == "" {
		slog.Info("Seeding ETHUSD")
		repo.Save(ctx, domain.NewInstrument("ETHUSD", "ETH", "USD", 0.01, 0.001, domain.Spot))
	}

	// 5. Application
	appService := application.NewReferenceDataApplicationService(repo)

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
