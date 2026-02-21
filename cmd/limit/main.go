package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/limit/application"
	"github.com/wyfcoding/financialtrading/internal/limit/infrastructure"
	limitiface "github.com/wyfcoding/financialtrading/internal/limit/interfaces"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("LIMIT_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_limit?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&infrastructure.LimitPO{},
		&infrastructure.LimitBreachPO{},
		&infrastructure.AccountLimitConfigPO{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	limitRepo := infrastructure.NewGormLimitRepository(db)
	breachRepo := infrastructure.NewGormLimitBreachRepository(db)
	configRepo := infrastructure.NewGormAccountLimitConfigRepository(db)
	appSvc := application.NewLimitApplicationService(limitRepo, breachRepo, configRepo, logger)
	handler := limitiface.NewLimitHandler(appSvc)

	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/limits")
	{
		v1.POST("/check", func(c *gin.Context) {
			var req limitiface.CheckLimitRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			resp, callErr := handler.CheckLimit(c.Request.Context(), &req)
			if callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, resp)
		})

		v1.POST("/value", func(c *gin.Context) {
			var req limitiface.UpdateLimitValueRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.LimitID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit_id", "limit_id is required")
				return
			}
			if callErr := handler.UpdateLimitValue(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.POST("/init", func(c *gin.Context) {
			var req limitiface.InitializeAccountLimitsRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.AccountID == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", "account_id is required")
				return
			}
			if callErr := handler.InitializeAccountLimits(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.GET("/:limit_id", func(c *gin.Context) {
			limitID := c.Param("limit_id")
			if limitID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit_id", "limit_id is required")
				return
			}
			resp, callErr := handler.GetLimit(c.Request.Context(), &limitiface.GetLimitRequest{LimitID: limitID})
			if callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, resp)
		})

		v1.GET("", func(c *gin.Context) {
			accountID, parseErr := parseUint64Query(c, "account_id")
			if parseErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", parseErr.Error())
				return
			}
			resp, callErr := handler.ListAccountLimits(c.Request.Context(), &limitiface.ListAccountLimitsRequest{AccountID: accountID})
			if callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, resp)
		})

		v1.GET("/config", func(c *gin.Context) {
			accountID, parseErr := parseUint64Query(c, "account_id")
			if parseErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", parseErr.Error())
				return
			}
			resp, callErr := handler.GetLimitConfig(c.Request.Context(), &limitiface.GetLimitConfigRequest{AccountID: accountID})
			if callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, resp)
		})

		v1.GET("/breaches", func(c *gin.Context) {
			accountID, parseErr := parseUint64Query(c, "account_id")
			if parseErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", parseErr.Error())
				return
			}
			page := parsePositiveInt(c.Query("page"), 1)
			pageSize := parsePositiveInt(c.Query("page_size"), 20)
			if pageSize > 200 {
				pageSize = 200
			}
			resolved := c.Query("resolved") == "true"

			resp, callErr := handler.ListBreaches(c.Request.Context(), &limitiface.ListBreachesRequest{
				AccountID: accountID,
				Resolved:  resolved,
				Page:      page,
				PageSize:  pageSize,
			})
			if callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, resp)
		})

		v1.POST("/breaches/resolve", func(c *gin.Context) {
			var req limitiface.ResolveBreachRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.BreachID == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid breach_id", "breach_id is required")
				return
			}
			if callErr := handler.ResolveBreach(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.POST("/freeze", func(c *gin.Context) {
			var req limitiface.FreezeLimitRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.LimitID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit_id", "limit_id is required")
				return
			}
			if callErr := handler.FreezeLimit(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.POST("/unfreeze", func(c *gin.Context) {
			var req limitiface.UnfreezeLimitRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.LimitID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit_id", "limit_id is required")
				return
			}
			if callErr := handler.UnfreezeLimit(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.POST("/instrument", func(c *gin.Context) {
			var req limitiface.CreateInstrumentLimitRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.AccountID == 0 || req.Symbol == "" || req.LimitType == "" || req.LimitValue == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", "account_id/symbol/limit_type/limit_value are required")
				return
			}
			if callErr := handler.CreateInstrumentLimit(c.Request.Context(), &req); callErr != nil {
				response.Error(c, callErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})
	}

	addr := os.Getenv("LIMIT_HTTP_ADDR")
	if addr == "" {
		addr = ":9109"
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
	slog.Info("service limit gracefully stopped")
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func parseUint64Query(c *gin.Context, key string) (uint64, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		return 0, fmt.Errorf("%s must be an unsigned integer", key)
	}
	return v, nil
}
