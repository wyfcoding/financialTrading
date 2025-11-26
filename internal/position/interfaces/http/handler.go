package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/position/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// PositionHandler HTTP 处理器
type PositionHandler struct {
	positionService *application.PositionApplicationService
}

// NewPositionHandler 创建 HTTP 处理器
func NewPositionHandler(positionService *application.PositionApplicationService) *PositionHandler {
	return &PositionHandler{
		positionService: positionService,
	}
}

// RegisterRoutes 注册路由
func (h *PositionHandler) RegisterRoutes(router *gin.Engine) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
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

	dtos, total, err := h.positionService.GetPositions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get positions", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "position_id is required"})
		return
	}

	dto, err := h.positionService.GetPosition(c.Request.Context(), positionID)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get position", "position_id", positionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// ClosePositionRequest 平仓请求
type ClosePositionRequest struct {
	ClosePrice string `json:"close_price" binding:"required"`
}

// ClosePosition 平仓
func (h *PositionHandler) ClosePosition(c *gin.Context) {
	positionID := c.Param("id")
	if positionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position_id is required"})
		return
	}

	var req ClosePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	closePrice, err := decimal.NewFromString(req.ClosePrice)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid close price"})
		return
	}

	if err := h.positionService.ClosePosition(c.Request.Context(), positionID, closePrice); err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to close position", "position_id", positionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated position
	dto, err := h.positionService.GetPosition(c.Request.Context(), positionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "closed", "message": "Position closed but failed to fetch updated details"})
		return
	}

	c.JSON(http.StatusOK, dto)
}
