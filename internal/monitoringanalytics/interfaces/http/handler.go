package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与监控分析相关的 HTTP 请求
type MonitoringAnalyticsHandler struct {
	app *application.MonitoringAnalyticsService // 监控分析应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的监控分析应用服务
func NewMonitoringAnalyticsHandler(app *application.MonitoringAnalyticsService) *MonitoringAnalyticsHandler {
	return &MonitoringAnalyticsHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *MonitoringAnalyticsHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/monitoring")
	{
		api.POST("/metrics", h.RecordMetric)
		api.GET("/metrics", h.GetMetrics)
		api.GET("/health/:service_name", h.GetSystemHealth)
	}
}

// RecordMetricRequest 记录指标请求
type RecordMetricRequest struct {
	Name      string            `json:"name" binding:"required"`
	Value     float64           `json:"value" binding:"required"`
	Tags      map[string]string `json:"tags"`
	Timestamp time.Time         `json:"timestamp"`
}

// RecordMetric 记录指标
func (h *MonitoringAnalyticsHandler) RecordMetric(c *gin.Context) {
	var req RecordMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	err := h.app.RecordMetric(c.Request.Context(), req.Name, req.Value, req.Tags, req.Timestamp)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to record metric", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetMetrics 获取指标
func (h *MonitoringAnalyticsHandler) GetMetrics(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
			return
		}
	} else {
		startTime = time.Now().Add(-1 * time.Hour)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
			return
		}
	} else {
		endTime = time.Now()
	}

	metrics, err := h.app.GetMetrics(c.Request.Context(), name, startTime, endTime)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get metrics", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetSystemHealth 获取系统健康状态
func (h *MonitoringAnalyticsHandler) GetSystemHealth(c *gin.Context) {
	serviceName := c.Param("service_name")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_name is required"})
		return
	}

	healths, err := h.app.GetSystemHealth(c.Request.Context(), serviceName)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get system health", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, healths)
}
