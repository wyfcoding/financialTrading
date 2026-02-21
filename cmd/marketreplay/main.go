package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketreplay/application"
	"github.com/wyfcoding/financialtrading/internal/marketreplay/domain"
	"github.com/wyfcoding/pkg/messagequeue"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	replayEngine := application.NewReplayEngine(noopEventPublisher{})

	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/marketreplay")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.Success(c, gin.H{"status": "ok"})
		})

		v1.POST("/start", func(c *gin.Context) {
			var req startReplayRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.Topic == "" || req.Symbol == "" || len(req.Ticks) == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", "topic/symbol/ticks are required")
				return
			}
			if req.SpeedFactor <= 0 {
				req.SpeedFactor = 1
			}

			task := &domain.ReplayTask{
				Symbol:      req.Symbol,
				StartTime:   req.StartTime,
				EndTime:     req.EndTime,
				SpeedFactor: req.SpeedFactor,
				Status:      domain.ReplayRunning,
				Topic:       req.Topic,
			}

			go replayEngine.StartReplay(context.Background(), task, req.Ticks)
			response.Success(c, gin.H{
				"started":      true,
				"symbol":       req.Symbol,
				"topic":        req.Topic,
				"ticks_count":  len(req.Ticks),
				"speed_factor": req.SpeedFactor,
			})
		})
	}

	addr := os.Getenv("MARKET_REPLAY_HTTP_ADDR")
	if addr == "" {
		addr = ":9110"
	}
	srv := server.NewGinServer(engine, addr, logger)

	go func() {
		if err := srv.Start(context.Background()); err != nil {
			slog.Error("server exit", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = srv.Stop(context.Background())
	slog.Info("service marketreplay gracefully stopped")
}

type startReplayRequest struct {
	Symbol      string              `json:"symbol"`
	Topic       string              `json:"topic"`
	SpeedFactor float64             `json:"speed_factor"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     time.Time           `json:"end_time"`
	Ticks       []domain.MarketTick `json:"ticks"`
}

type noopEventPublisher struct{}

func (noopEventPublisher) Publish(_ context.Context, _ string, _ string, _ any) error { return nil }

func (noopEventPublisher) PublishInTx(_ context.Context, _ any, _ string, _ string, _ any) error {
	return nil
}

var _ messagequeue.EventPublisher = (*noopEventPublisher)(nil)
