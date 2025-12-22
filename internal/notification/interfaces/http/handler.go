package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/notification/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与通知相关的 HTTP 请求
type NotificationHandler struct {
	app *application.NotificationService // 通知应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的通知应用服务
func NewNotificationHandler(app *application.NotificationService) *NotificationHandler {
	return &NotificationHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *NotificationHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/notifications")
	{
		api.POST("/send", h.SendNotification)
		api.GET("/history", h.GetNotificationHistory)
	}
}

// SendNotificationRequest 发送通知请求
type SendNotificationRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Subject string `json:"subject" binding:"required"`
	Content string `json:"content" binding:"required"`
	Target  string `json:"target" binding:"required"`
}

// SendNotification 发送通知
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.SendNotification(c.Request.Context(), req.UserID, req.Type, req.Subject, req.Content, req.Target)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to send notification", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notification_id": id})
}

// GetNotificationHistory 获取通知历史
func (h *NotificationHandler) GetNotificationHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	notifications, err := h.app.GetNotificationHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get notification history", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}
