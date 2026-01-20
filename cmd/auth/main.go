package main

import (
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
	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"github.com/wyfcoding/financialtrading/internal/auth/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/auth/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/auth/interfaces/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/auth/config.toml", "path to config file")
	flag.Parse()

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.APIKey{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	repo := mysql.NewUserRepository(db)
	apiKeyRepo := mysql.NewAPIKeyRepository(db)
	keySvc := application.NewAPIKeyService(apiKeyRepo)
	appService := application.NewAuthApplicationService(repo, apiKeyRepo, keySvc)

	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	grpcPort := viper.GetString("server.grpc_port")
	lis, _ := net.Listen("tcp", ":"+grpcPort)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	http_server.NewHandler(r, appService)
	httpPort := viper.GetString("server.http_port")
	httpSrv := &http.Server{Addr: ":" + httpPort, Handler: r}

	go func() { slog.Info("gRPC server", "port", grpcPort); grpcSrv.Serve(lis) }()
	go func() { slog.Info("HTTP server", "port", httpPort); httpSrv.ListenAndServe() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down...")
	grpcSrv.GracefulStop()
}
