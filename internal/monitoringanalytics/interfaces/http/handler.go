package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/response"
)

// MonitoringHandler HTTP 处理器
type MonitoringHandler struct {
	app *application.MonitoringAnalyticsService
}

// NewMonitoringHandler 创建 HTTP 处理器实例
func NewMonitoringHandler(app *application.MonitoringAnalyticsService) *MonitoringHandler {
	return &MonitoringHandler{app: app}
}

// RegisterRoutes 注册路由
func (h *MonitoringHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api/v1/monitoring")
	{
		api.GET("/metrics/trade", h.GetTradeMetrics)
		api.GET("/alerts", h.GetAlerts)
		api.GET("/health/:service", h.GetSystemHealth)
	}
}

// GetTradeMetrics 获取交易指标
func (h *MonitoringHandler) GetTradeMetrics(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "symbol is required", "")
		return
	}

	// Default time range: last 24h
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if s := c.Query("start_time"); s != "" {
		if t, err := strconv.ParseInt(s, 10, 64); err == nil {
			startTime = time.Unix(t, 0)
		}
	}
	if e := c.Query("end_time"); e != "" {
		if t, err := strconv.ParseInt(e, 10, 64); err == nil {
			endTime = time.Unix(t, 0)
		}
	}

	dtos, err := h.app.GetTradeMetrics(c.Request.Context(), symbol, startTime, endTime)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get trade metrics", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dtos)
}

// GetAlerts 获取告警列表
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	dtos, err := h.app.GetAlerts(c.Request.Context(), limit)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get alerts", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dtos)
}

// GetSystemHealth 获取系统健康状态
func (h *MonitoringHandler) GetSystemHealth(c *gin.Context) {
	serviceName := c.Param("service")
	if serviceName == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "service name is required", "")
		return
	}

	dtos, err := h.app.GetSystemHealth(c.Request.Context(), serviceName)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get system health", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dtos)
}
