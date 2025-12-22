package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/market-simulation/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与市场模拟相关的 HTTP 请求
type MarketSimulationHandler struct {
	app *application.MarketSimulationService // 市场模拟应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的市场模拟应用服务
func NewMarketSimulationHandler(app *application.MarketSimulationService) *MarketSimulationHandler {
	return &MarketSimulationHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *MarketSimulationHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/market-simulation")
	{
		api.POST("/start", h.StartSimulation)
		api.POST("/stop", h.StopSimulation)
		api.GET("/status", h.GetSimulationStatus)
	}
}

// StartSimulationRequest 启动模拟请求
type StartSimulationRequest struct {
	Name       string `json:"name" binding:"required"`
	Symbol     string `json:"symbol" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Parameters string `json:"parameters"`
}

// StartSimulation 启动模拟
func (h *MarketSimulationHandler) StartSimulation(c *gin.Context) {
	var req StartSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.StartSimulation(c.Request.Context(), req.Name, req.Symbol, req.Type, req.Parameters)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to start simulation", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"simulation_id": id})
}

// StopSimulationRequest 停止模拟请求
type StopSimulationRequest struct {
	SimulationID string `json:"simulation_id" binding:"required"`
}

// StopSimulation 停止模拟
func (h *MarketSimulationHandler) StopSimulation(c *gin.Context) {
	var req StopSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	success, err := h.app.StopSimulation(c.Request.Context(), req.SimulationID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to stop simulation", "simulation_id", req.SimulationID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": success})
}

// GetSimulationStatus 获取模拟状态
func (h *MarketSimulationHandler) GetSimulationStatus(c *gin.Context) {
	simulationID := c.Query("simulation_id")
	if simulationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "simulation_id is required"})
		return
	}

	scenario, err := h.app.GetSimulationStatus(c.Request.Context(), simulationID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get simulation status", "simulation_id", simulationID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if scenario == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scenario not found"})
		return
	}

	c.JSON(http.StatusOK, scenario)
}
