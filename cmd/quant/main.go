package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	quant_pb "github.com/wyfcoding/financialtrading/go-api/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/quant/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/quant/config.toml", "path to config file")
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
	if err := db.AutoMigrate(&domain.Signal{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	signalRepo := mysql.NewSignalRepository(db)
	strategyRepo := mysql.NewStrategyRepository(db)
	backtestRepo := mysql.NewBacktestResultRepository(db)
	// marketDataClient := ...

	// 5. Application
	appService := application.NewQuantService(strategyRepo, backtestRepo, signalRepo, nil)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	quantHandler := grpc_server.NewHandler(appService)
	quant_pb.RegisterQuantServiceServer(grpcSrv, quantHandler)
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
