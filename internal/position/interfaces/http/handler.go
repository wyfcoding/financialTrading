package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与持仓管理相关的 HTTP 请求
type PositionHandler struct {
	positionService *application.PositionService // 持仓应用服务
}

// 创建 HTTP 处理器
// positionService: 注入的持仓应用服务
func NewPositionHandler(positionService *application.PositionService) *PositionHandler {
	return &PositionHandler{
		positionService: positionService,
	}
}

// 注册路由
func (h *PositionHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api/v1/positions")
	{
		api.GET("", h.GetPositions)
		api.GET("/:id", h.GetPosition)
		api.POST("/:id/close", h.ClosePosition)
	}
}

// GetPositions 获取持仓列表
func (h *PositionHandler) GetPositions(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid offset", "")
		return
	}

	dtos, total, err := h.positionService.GetPositions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get positions", "user_id", userID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  dtos,
		"total": total,
	})
}

// GetPosition 获取持仓详情
func (h *PositionHandler) GetPosition(c *gin.Context) {
	positionID := c.Param("id")
	if positionID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "position_id is required", "")
		return
	}

	dto, err := h.positionService.GetPosition(c.Request.Context(), positionID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get position", "position_id", positionID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}

// ClosePositionRequest 平仓请求
type ClosePositionRequest struct {
	ClosePrice string `json:"close_price" binding:"required"`
}

// ClosePosition 平仓
func (h *PositionHandler) ClosePosition(c *gin.Context) {
	positionID := c.Param("id")
	if positionID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "position_id is required", "")
		return
	}

	var req ClosePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	closePrice, err := decimal.NewFromString(req.ClosePrice)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid close price", "")
		return
	}

	if err := h.positionService.ClosePosition(c.Request.Context(), positionID, closePrice); err != nil {
		logging.Error(c.Request.Context(), "Failed to close position", "position_id", positionID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	// 返回更新后的持仓信息
	dto, err := h.positionService.GetPosition(c.Request.Context(), positionID)
	if err != nil {
		response.Success(c, gin.H{"status": "closed", "message": "Position closed but failed to fetch updated details"})
		return
	}

	response.Success(c, dto)
}
