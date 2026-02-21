package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/riskgateway/application"
	"github.com/wyfcoding/financialtrading/internal/riskgateway/domain"
	"github.com/wyfcoding/pkg/money"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	repo := domain.NewMemoryRiskRepository()
	svc := application.NewRiskCheckService(repo)

	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/riskgateway")
	{
		v1.POST("/limits", func(c *gin.Context) {
			var req setRiskLimitRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.AccountID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", "account_id is required")
				return
			}
			maxOrder := money.New(req.MaxOrderSize)
			maxDailyLoss := money.New(req.MaxDailyLoss)
			repo.SaveAccountRisk(&domain.AccountRisk{
				AccountID:    req.AccountID,
				MaxOrderSize: maxOrder,
				MaxDailyLoss: maxDailyLoss,
			})
			response.Success(c, gin.H{"success": true})
		})

		v1.POST("/pretrade", func(c *gin.Context) {
			var req preTradeCheckRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.AccountID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", "account_id is required")
				return
			}
			amount, parseErr := money.NewFromString(req.OrderAmount)
			if parseErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid order_amount", parseErr.Error())
				return
			}
			passed, checkErr := svc.PreTradeCheck(c.Request.Context(), req.AccountID, amount)
			if checkErr != nil {
				response.Error(c, checkErr)
				return
			}
			response.Success(c, gin.H{"passed": passed})
		})
	}

	addr := os.Getenv("RISKGATEWAY_HTTP_ADDR")
	if addr == "" {
		addr = ":9111"
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
	slog.Info("service riskgateway gracefully stopped")
}

type setRiskLimitRequest struct {
	AccountID    string  `json:"account_id"`
	MaxOrderSize float64 `json:"max_order_size"`
	MaxDailyLoss float64 `json:"max_daily_loss"`
}

type preTradeCheckRequest struct {
	AccountID   string `json:"account_id"`
	OrderAmount string `json:"order_amount"`
}
