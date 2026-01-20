package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	executionv1 "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
)

type ExecutionHandler struct {
	app   *application.ExecutionApplicationService
	query *application.ExecutionQueryService
}

func NewExecutionHandler(app *application.ExecutionApplicationService, query *application.ExecutionQueryService) *ExecutionHandler {
	return &ExecutionHandler{app: app, query: query}
}

func (h *ExecutionHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1/execution")
	{
		v1.POST("/order", h.ExecuteOrder)
		v1.POST("/algo", h.SubmitAlgoOrder)
	}
}

func (h *ExecutionHandler) ExecuteOrder(c *gin.Context) {
	var req executionv1.ExecuteOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	price, _ := decimal.NewFromString(req.Price)
	qty, _ := decimal.NewFromString(req.Quantity)

	cmd := application.ExecuteOrderCommand{
		OrderID:  req.OrderId,
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    price,
		Quantity: qty,
	}

	dto, err := h.app.ExecuteOrder(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

func (h *ExecutionHandler) SubmitAlgoOrder(c *gin.Context) {
	var req executionv1.SubmitAlgoOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	totalQty, _ := decimal.NewFromString(req.TotalQuantity)

	cmd := application.SubmitAlgoCommand{
		UserID:    req.UserId,
		Symbol:    req.Symbol,
		Side:      req.Side,
		TotalQty:  totalQty,
		AlgoType:  req.AlgoType,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Params:    req.ParticipationRate,
	}

	algoID, err := h.app.SubmitAlgoOrder(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"algo_id": algoID, "status": "ACCEPTED"})
}
