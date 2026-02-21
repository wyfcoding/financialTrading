package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/notification/application"
	"github.com/wyfcoding/financialtrading/internal/notification/infrastructure"
	notificationiface "github.com/wyfcoding/financialtrading/internal/notification/interfaces"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("NOTIFICATION_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_notification?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&infrastructure.NotificationPO{},
		&infrastructure.NotificationTemplatePO{},
		&infrastructure.UserNotificationPreferencePO{},
		&infrastructure.NotificationBatchPO{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	notificationRepo := infrastructure.NewGormNotificationRepository(db)
	templateRepo := infrastructure.NewGormNotificationTemplateRepository(db)
	preferenceRepo := infrastructure.NewGormUserNotificationPreferenceRepository(db)
	batchRepo := infrastructure.NewGormNotificationBatchRepository(db)

	appService := application.NewNotificationApplicationService(
		notificationRepo,
		templateRepo,
		preferenceRepo,
		batchRepo,
		infrastructure.NoopEmailSender{},
		infrastructure.NoopSMSSender{},
		infrastructure.NoopPushSender{},
		infrastructure.NoopWebSocketSender{},
		infrastructure.NoopWebhookSender{},
		logger,
	)
	handler := notificationiface.NewNotificationHandler(appService)

	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/notifications")
	{
		v1.POST("/send", func(c *gin.Context) {
			var req notificationiface.SendNotificationRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}

			resp, sendErr := handler.SendNotification(c.Request.Context(), &req)
			if sendErr != nil {
				response.Error(c, sendErr)
				return
			}
			response.Success(c, resp)
		})

		v1.GET("", func(c *gin.Context) {
			userID, parseErr := strconv.ParseUint(c.Query("user_id"), 10, 64)
			if parseErr != nil || userID == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid user_id", "user_id must be an unsigned integer")
				return
			}

			page := parsePositiveInt(c.Query("page"), 1)
			pageSize := parsePositiveInt(c.Query("page_size"), 20)
			if pageSize > 200 {
				pageSize = 200
			}

			resp, listErr := handler.ListNotifications(c.Request.Context(), &notificationiface.ListNotificationsRequest{
				UserID:   userID,
				Status:   c.Query("status"),
				Page:     page,
				PageSize: pageSize,
			})
			if listErr != nil {
				response.Error(c, listErr)
				return
			}
			response.Success(c, resp)
		})

		v1.POST("/read", func(c *gin.Context) {
			var req notificationiface.MarkAsReadRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.NotificationID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid notification_id", "notification_id is required")
				return
			}
			if markErr := handler.MarkAsRead(c.Request.Context(), &req); markErr != nil {
				response.Error(c, markErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.POST("/preferences", func(c *gin.Context) {
			var req notificationiface.SetPreferenceRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.UserID == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid user_id", "user_id is required")
				return
			}
			if req.Type == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid type", "type is required")
				return
			}
			if setErr := handler.SetPreference(c.Request.Context(), &req); setErr != nil {
				response.Error(c, setErr)
				return
			}
			response.Success(c, gin.H{"ok": true})
		})

		v1.GET("/preferences", func(c *gin.Context) {
			userID, parseErr := strconv.ParseUint(c.Query("user_id"), 10, 64)
			if parseErr != nil || userID == 0 {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid user_id", "user_id must be an unsigned integer")
				return
			}
			resp, getErr := handler.GetPreferences(c.Request.Context(), &notificationiface.GetPreferencesRequest{UserID: userID})
			if getErr != nil {
				response.Error(c, getErr)
				return
			}
			response.Success(c, resp)
		})
	}

	addr := os.Getenv("NOTIFICATION_HTTP_ADDR")
	if addr == "" {
		addr = ":9106"
	}
	httpServer := server.NewGinServer(engine, addr, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Info("notification service started", "addr", addr)
		if runErr := httpServer.Start(ctx); runErr != nil {
			logger.Error("notification service exit", "error", runErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down notification service...")
	if stopErr := httpServer.Stop(context.Background()); stopErr != nil {
		logger.Error("failed to stop notification service", "error", stopErr)
	}
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
