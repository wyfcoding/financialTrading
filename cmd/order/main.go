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
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/order/interfaces/grpc"
	"github.com/wyfcoding/pkg/security/risk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/order/config.toml", "path to config file")
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
	if err := db.AutoMigrate(&domain.Order{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewOrderRepository(db)

	// Risk Evaluator (Local)
	riskEngine, err := risk.NewDynamicRiskEngine(logger)
	if err != nil {
		panic(fmt.Sprintf("failed to init risk engine: %v", err))
	}

	// 5. Application
	// Inject dependencies
	orderManager := application.NewOrderManager(repo, riskEngine, logger)
	orderQuery := application.NewOrderQuery(repo)

	// Configure DTM/Remote Services if needed (from config)
	if dtmServer := viper.GetString("dtm.server"); dtmServer != "" {
		orderManager.SetDTMServer(dtmServer)
	}
	// TODO: SetRiskClient, SetAccountClient, SetPositionClient using gRPC connection pools

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	handler := grpc_server.NewHandler(orderManager, orderQuery)
	orderv1.RegisterOrderServiceServer(grpcSrv, handler)
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
