// 包  HTTP 处理器（Handler）的实现。
// 这一层是接口层（Interfaces Layer）的一部分，使用 Gin 框架，
// 负责适配外部的 HTTP 请求，并将其转换为对应用层的调用。
package http

import (
	"github.com/wyfcoding/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"github.com/wyfcoding/pkg/logging"
)

// ClearingHandler 是清算服务的 HTTP 处理器。
// 它封装了所有与清算相关的 HTTP 接口逻辑。
type ClearingHandler struct {
	clearingService *application.ClearingApplicationService // 依赖注入的应用服务实例
}

// NewClearingHandler 是 ClearingHandler 的构造函数。
func NewClearingHandler(clearingService *application.ClearingApplicationService) *ClearingHandler {
	return &ClearingHandler{
		clearingService: clearingService,
	}
}

// RegisterRoutes 在 Gin 路由引擎上注册所有与清算相关的 HTTP 路由。
func (h *ClearingHandler) RegisterRoutes(router *gin.Engine) {
	// 创建一个路由组，为所有相关路由添加统一的前缀 `/api/v1/clearing`。
	api := router.Group("/api/v1/clearing")
	{
		api.POST("/settle", h.SettleTrade)
		api.POST("/eod", h.ExecuteEODClearing)
		api.GET("/status/:id", h.GetClearingStatus) // 使用 /status/:id 增加语义化
	}
}

// SettleTrade 处理清算单笔交易的 HTTP POST 请求。
// @Summary 清算单笔交易
// @Description 接收交易详情并执行清算
// @Tags Clearing
// @Accept json
// @Produce json
// @Param trade body application.SettleTradeRequest true "清算请求"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/clearing/settle [post]
func (h *ClearingHandler) SettleTrade(c *gin.Context) {
	var req application.SettleTradeRequest
	// 1. 将请求的 JSON body 绑定到 `req` 结构体上。
	//    `ShouldBindJSON` 会自动进行数据类型和格式的初步校验。
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "Invalid request body: " + err.Error(), "")
		return
	}

	// 2. 调用应用服务执行核心业务逻辑,接收返回的 settlementID。
	//    从 Gin 的 context 中传递 `Request.Context()`，以支持 Trace 和超时控制。
	settlementID, err := h.clearingService.SettleTrade(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to settle trade", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, "Failed to settle trade: " + err.Error(), "")
		return
	}

	// 3. 返回成功的响应,包含 settlementID。
	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"message":       "Trade settlement initiated",
		"settlement_id": settlementID,
	})
}

// ExecuteEODClearingRequest 是执行日终清算请求的专用结构体。
// 在接口层定义独立的请求结构体可以更灵活地进行绑定和验证。
type ExecuteEODClearingRequest struct {
	ClearingDate string `json:"clearing_date" binding:"required,datetime=2006-01-02"` // `binding` tag 提供了校验规则
}

// ExecuteEODClearing 处理执行日终清算的 HTTP POST 请求。
// @Summary 执行日终清算
// @Description 根据指定日期启动日终清算流程
// @Tags Clearing
// @Accept json
// @Produce json
// @Param clearing_date body ExecuteEODClearingRequest true "清算日期"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/clearing/eod [post]
func (h *ClearingHandler) ExecuteEODClearing(c *gin.Context) {
	var req ExecuteEODClearingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "Invalid request body: " + err.Error(), "")
		return
	}

	clearingID, err := h.clearingService.ExecuteEODClearing(c.Request.Context(), req.ClearingDate)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to execute EOD clearing", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, "Failed to execute EOD clearing: " + err.Error(), "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "processing",
		"message":     "EOD clearing process started",
		"clearing_id": clearingID,
	})
}

// GetClearingStatus 处理获取清算任务状态的 HTTP GET 请求。
// @Summary 获取清算任务状态
// @Description 根据清算ID获取日终清算等任务的当前状态和进度
// @Tags Clearing
// @Produce json
// @Param id path string true "清算任务ID"
// @Success 200 {object} domain.EODClearing
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/clearing/status/{id} [get]
func (h *ClearingHandler) GetClearingStatus(c *gin.Context) {
	// 1. 从 URL 路径参数中获取 `id`。
	clearingID := c.Param("id")
	if clearingID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "clearing_id is required in path", "")
		return
	}

	// 2. 调用应用服务查询状态。
	clearing, err := h.clearingService.GetClearingStatus(c.Request.Context(), clearingID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get clearing status", "clearing_id", clearingID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, "Failed to get clearing status: " + err.Error(), "")
		return
	}

	// 3. 如果未找到记录，返回 404 Not Found。
	if clearing == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "Clearing task not found", "")
		return
	}

	// 4. 返回查询到的清算任务详情。
	response.Success(c, clearing)
}
