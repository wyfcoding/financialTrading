package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// ...
	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.TradeMetric{}, &domain.Alert{}, &domain.SystemHealth{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	metricRepo := mysql.NewMetricRepository(db)
	healthRepo := mysql.NewSystemHealthRepository(db)
	alertRepo := mysql.NewAlertRepository(db)

	// 5. Application
	appService := application.NewMonitoringAnalyticsService(metricRepo, healthRepo, alertRepo)

	// TODO: Start Kafka Consumer Group here to feed appService.RecordTrade

	// 6. Interfaces
	// gRPC
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
