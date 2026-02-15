// Package http 抵押品服务接口
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/collateral/application"
	"github.com/wyfcoding/financialtrading/internal/collateral/domain"
)

type Handler struct {
	service *application.CollateralService
}

func NewHandler(service *application.CollateralService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/collateral")
	{
		g.POST("/deposit", h.Deposit)
		g.POST("/withdraw", h.Withdraw)
		g.POST("/valuation", h.TriggerValuation)
		g.GET("/accounts/:id/value", h.GetCollateralValue)
	}
}

type DepositReq struct {
	AccountID string           `json:"account_id" binding:"required"`
	AssetType domain.AssetType `json:"asset_type" binding:"required"`
	Symbol    string           `json:"symbol" binding:"required"`
	Quantity  string           `json:"quantity" binding:"required"`
	Currency  string           `json:"currency" binding:"required"`
}

func (h *Handler) Deposit(c *gin.Context) {
	var req DepositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qty, _ := decimal.NewFromString(req.Quantity)
	cmd := application.DepositCmd{
		AccountID: req.AccountID,
		AssetType: req.AssetType,
		Symbol:    req.Symbol,
		Quantity:  qty,
		Currency:  req.Currency,
	}

	id, err := h.service.DepositCollateral(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset_id": id})
}

type WithdrawReq struct {
	AssetID  string `json:"asset_id" binding:"required"`
	Quantity string `json:"quantity" binding:"required"`
}

func (h *Handler) Withdraw(c *gin.Context) {
	var req WithdrawReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qty, _ := decimal.NewFromString(req.Quantity)
	cmd := application.WithdrawCmd{
		AssetID:  req.AssetID,
		Quantity: qty,
	}

	if err := h.service.WithdrawCollateral(c.Request.Context(), cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) TriggerValuation(c *gin.Context) {
	accountID := c.Query("account_id")
	if err := h.service.ValuationUpdate(c.Request.Context(), accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) GetCollateralValue(c *gin.Context) {
	accountID := c.Param("id")
	currency := c.DefaultQuery("currency", "USD")

	val, err := h.service.GetAccountCollateralValue(c.Request.Context(), accountID, currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_collateral_value": val})
}
