package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/pkg/logging"
)

// RiskHandler 负责处理与风险管理相关的 HTTP 请求
type RiskHandler struct {
	cmd   *application.RiskCommandService
	query *application.RiskQueryService
}

// NewRiskHandler 创建 HTTP 处理器
func NewRiskHandler(cmd *application.RiskCommandService, query *application.RiskQueryService) *RiskHandler {
	return &RiskHandler{cmd: cmd, query: query}
}

// RegisterRoutes 注册路由
func (h *RiskHandler) RegisterRoutes(router *gin.RouterGroup) {
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
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	qty, err := strconv.ParseFloat(req.Quantity, 64)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid quantity", "")
		return
	}
	price, err := strconv.ParseFloat(req.Price, 64)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid price", "")
		return
	}
	cmd := application.AssessRiskCommand{
		UserID:   req.UserID,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: qty,
		Price:    price,
	}

	dto, err := h.cmd.AssessRisk(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to assess risk", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}

// GetRiskMetrics 获取风险指标
func (h *RiskHandler) GetRiskMetrics(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	metrics, err := h.query.GetRiskMetrics(c.Request.Context(), userID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get risk metrics", "user_id", userID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, metrics)
}

// CheckRiskLimit 检查风险限额
func (h *RiskHandler) CheckRiskLimit(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	limitType := c.Query("limit_type")
	if limitType == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "limit_type is required", "")
		return
	}

	limit, err := h.query.CheckRiskLimit(c.Request.Context(), userID, limitType)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to check risk limit", "user_id", userID, "limit_type", limitType, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, limit)
}

// GetRiskAlerts 获取风险告警
func (h *RiskHandler) GetRiskAlerts(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	alerts, err := h.query.GetRiskAlerts(c.Request.Context(), userID, limit)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get risk alerts", "user_id", userID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, alerts)
}
