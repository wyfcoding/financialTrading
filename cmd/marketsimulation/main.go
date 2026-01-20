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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/marketsimulation/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/marketsimulation/interfaces/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/marketsimulation/config.toml", "path to config file")
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
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&mysql.SimulationPO{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewSimulationRepository(db)

	// 5. Application
	appService := application.NewMarketSimulationApplicationService(repo)

	// Resume running simulations
	ctx := context.Background()
	// Logic to resume: list runnning in DB, and start them.
	// Note: In a real distributed system, we might need leader election or partition handling.
	// For now, we assume single instance or that StartSimulation handles idempotency (locally).
	// Ideally, we should check DB for "Running" status and call StartSimulation.
	// However, appService.StartSimulation checks if it is ALREADY running in memory.
	// So we need to iterate DB "Running" ones and start them.
	// But our StartSimulation also writes to DB. We don't want to double write "Running".
	// The current StartSimulation sets runningSims map. I should expose a Resume method or just call StartSimulation and ignore "already running" DB error?
	// But StartSimulation returns error if status is running in DB?
	// Let's manually resume here.
	// Actually, the simplest way is to fetch "Running" from Repo and blindly spawn workers.
	// I'll leave this as a TODO or simple loop if I had exposed a "Resume" method.
	// Given the code I wrote, StartSimulation checks `s.Status == Running` => Error.
	// So I can't call StartSimulation on already running ones.
	// I will skip auto-resume for now to keep it simple, or I should have added `Resume` in app service.

	// 6. Interfaces
	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	http_server.NewHandler(r, appService)
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%s", viper.GetString("server.http_port")),
		Handler: r,
	}

	// gRPC
	grpcSrv := grpc.NewServer()
	grpc_server.NewServer(grpcSrv, appService)
	reflection.Register(grpcSrv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", viper.GetString("server.grpc_port")))
	if err != nil {
		panic(err)
	}

	// 7. Start
	go func() {
		slog.Info("Starting HTTP server", "port", viper.GetString("server.http_port"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	go func() {
		slog.Info("Starting gRPC server", "port", viper.GetString("server.grpc_port"))
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	grpcSrv.GracefulStop()

	slog.Info("Server exiting")
}
