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
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	match_mem "github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/memory"
	match_mysql "github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/matchingengine/config.toml", "path to config file")
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

	// 3. Database (MySQL)
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// 4. Infrastructure
	orderBookRepo := match_mem.NewInMemoryRepository()
	tradeRepo := match_mysql.NewTradeRepository(db)

	// 5. Domain Engine Configuration
	symbol := viper.GetString("matching.symbol")
	if symbol == "" {
		symbol = "BTC-USDT"
	}

	// 6. Application Service
	service, err := application.NewMatchingEngineService(
		symbol,
		tradeRepo,
		orderBookRepo,
		db,
		nil, // outboxMgr (ignored if not used)
		logger,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to init matching engine service: %v", err))
	}

	// 7. Interfaces (gRPC)
	grpcSrv := grpc.NewServer()
	grpc_server.NewHandler(service)
	reflection.Register(grpcSrv)

	port := viper.GetString("server.grpc_port")
	if port == "" {
		port = "50055"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	// 8. Start
	go func() {
		slog.Info("Starting gRPC server", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	grpcSrv.GracefulStop()
	slog.Info("Server exiting")
}
