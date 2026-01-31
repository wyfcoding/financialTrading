package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	v1 "github.com/wyfcoding/financialtrading/go-api/monitoringanalytics/v1"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	monitor_ck "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/clickhouse"
	monitor_es "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/search"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. Config (Simplified for demo, usually uses pkg/config)
	// viper.SetConfigFile(...) already done or should be managed better.
	// For now, keep current structure but fix missing pieces.

	// 2. Logger
	logger := logging.NewLogger("monitoring", "main", viper.GetString("log.level"))
	slog.SetDefault(logger.Logger)

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

	// Elasticsearch
	esCfg := &config.ElasticsearchConfig{
		Addresses: viper.GetStringSlice("elasticsearch.addresses"),
		Username:  viper.GetString("elasticsearch.username"),
		Password:  viper.GetString("elasticsearch.password"),
	}
	esClient, err := search.NewClient(esCfg, logger)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	}
	auditESRepo := monitor_es.NewAuditESRepository(esClient)

	// ClickHouse
	ckConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{viper.GetString("clickhouse.addr")},
		Auth: clickhouse.Auth{
			Database: viper.GetString("clickhouse.database"),
			Username: viper.GetString("clickhouse.username"),
			Password: viper.GetString("clickhouse.password"),
		},
	})
	if err != nil {
		slog.Error("failed to init clickhouse", "error", err)
	}
	auditRepo := monitor_ck.NewAuditRepository(ckConn)

	// 5. Application
	appService, err := application.NewMonitoringAnalyticsService(metricRepo, healthRepo, alertRepo, auditRepo, auditESRepo, db)
	if err != nil {
		panic(fmt.Sprintf("create app service failed: %v", err))
	}

	// TODO: Start Kafka Consumer Group here to feed appService.RecordTrade

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	monitoringHandler := grpc_server.NewHandler(appService)
	v1.RegisterMonitoringAnalyticsServer(grpcSrv, monitoringHandler)
	reflection.Register(grpcSrv)

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewMonitoringHandler(appService)
	hHandler.RegisterRoutes(r.Group("/api"))

	// 7. Start
	g, ctx := errgroup.WithContext(context.Background())

	grpcPort := viper.GetString("server.grpc_port")
	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
		if err != nil {
			return err
		}
		slog.Info("gRPC server starting", "port", grpcPort)
		return grpcSrv.Serve(lis)
	})

	httpPort := viper.GetString("server.http_port")
	if httpPort == "" {
		httpPort = "8080"
	}
	g.Go(func() error {
		addr := fmt.Sprintf(":%s", httpPort)
		server := &http.Server{Addr: addr, Handler: r}
		slog.Info("HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
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
