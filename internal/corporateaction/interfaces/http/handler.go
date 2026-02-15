// Package http 公司行动 HTTP 接口
package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/application"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/domain"
)

type Handler struct {
	service *application.CorporateActionService
}

func NewHandler(service *application.CorporateActionService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/corporate-actions")
	{
		g.POST("", h.AnnounceAction)
		g.POST("/:id/calculate", h.CalculateEntitlements)
		g.POST("/:id/process", h.ProcessPayments)
	}
}

type AnnounceActionReq struct {
	Symbol           string            `json:"symbol" binding:"required"`
	Type             domain.ActionType `json:"type" binding:"required"`
	ExDate           string            `json:"ex_date" binding:"required"`
	RecordDate       string            `json:"record_date" binding:"required"`
	PaymentDate      string            `json:"payment_date" binding:"required"`
	RatioNumerator   string            `json:"ratio_numerator" binding:"required"`
	RatioDenominator string            `json:"ratio_denominator" binding:"required"`
	Currency         string            `json:"currency"`
	Description      string            `json:"description"`
}

func (h *Handler) AnnounceAction(c *gin.Context) {
	var req AnnounceActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exDate, _ := time.Parse("2006-01-02", req.ExDate)
	recordDate, _ := time.Parse("2006-01-02", req.RecordDate)
	paymentDate, _ := time.Parse("2006-01-02", req.PaymentDate)
	num, _ := decimal.NewFromString(req.RatioNumerator)
	den, _ := decimal.NewFromString(req.RatioDenominator)

	cmd := application.CreateActionCmd{
		Symbol:           req.Symbol,
		Type:             req.Type,
		ExDate:           exDate,
		RecordDate:       recordDate,
		PaymentDate:      paymentDate,
		RatioNumerator:   num,
		RatioDenominator: den,
		Currency:         req.Currency,
		Description:      req.Description,
	}

	id, err := h.service.AnnounceAction(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"event_id": id})
}

func (h *Handler) CalculateEntitlements(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.CalculateEntitlements(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) ProcessPayments(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.ProcessPayments(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
