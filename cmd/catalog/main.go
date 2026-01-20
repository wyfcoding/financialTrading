package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/wyfcoding/financialtrading/internal/catalog/application"
	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
	"github.com/wyfcoding/financialtrading/internal/catalog/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/catalog/interfaces/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/catalog/config.toml", "path to config file")
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
	db.AutoMigrate(&domain.Product{})

	repo := mysql.NewProductRepository(db)
	appService := application.NewCatalogApplicationService(repo)

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
