package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/risk/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// RiskHandler HTTP 处理器
type RiskHandler struct {
	riskService *application.RiskApplicationService
}

// NewRiskHandler 创建 HTTP 处理器
func NewRiskHandler(riskService *application.RiskApplicationService) *RiskHandler {
	return &RiskHandler{
		riskService: riskService,
	}
}

// RegisterRoutes 注册路由
func (h *RiskHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/risk")
	{
		api.POST("/assess", h.AssessRisk)
		api.GET("/metrics", h.GetRiskMetrics)
		api.GET("/limits", h.CheckRiskLimit)
		api.GET("/alerts", h.GetRiskAlerts)
	}
}

// AssessRisk 评估交易风险
func (h *RiskHandler) AssessRisk(c *gin.Context) {
	var req application.AssessRiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dto, err := h.riskService.AssessRisk(c.Request.Context(), &req)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to assess risk", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// GetRiskMetrics 获取风险指标
func (h *RiskHandler) GetRiskMetrics(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	metrics, err := h.riskService.GetRiskMetrics(c.Request.Context(), userID)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get risk metrics", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// CheckRiskLimit 检查风险限额
func (h *RiskHandler) CheckRiskLimit(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitType := c.Query("limit_type")
	if limitType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit_type is required"})
		return
	}

	limit, err := h.riskService.CheckRiskLimit(c.Request.Context(), userID, limitType)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to check risk limit", "user_id", userID, "limit_type", limitType, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, limit)
}

// GetRiskAlerts 获取风险告警
func (h *RiskHandler) GetRiskAlerts(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	alerts, err := h.riskService.GetRiskAlerts(c.Request.Context(), userID, limit)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get risk alerts", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}
