// 变更说明：新增 ledger HTTP handler，提供 RESTful API 接口。
// 主要用于管理后台和运维工具的查询操作，核心写操作通过 gRPC 进行。
package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyfcoding/financialtrading/internal/ledger/application"
	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
	"github.com/wyfcoding/pkg/response"
)

// Handler 账本 HTTP 处理器。
type Handler struct {
	querySvc *application.LedgerQueryService
	cmdSvc   *application.LedgerCommandService
}

// NewHandler 创建 HTTP 处理器实例。
func NewHandler(querySvc *application.LedgerQueryService, cmdSvc *application.LedgerCommandService) *Handler {
	return &Handler{
		querySvc: querySvc,
		cmdSvc:   cmdSvc,
	}
}

// RegisterRoutes 注册路由。
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	ledger := r.Group("/ledger")
	{
		ledger.GET("/accounts/:accountId/balance", h.GetBalance)
		ledger.GET("/accounts/summary/:ownerId", h.GetAccountSummary)
		ledger.GET("/accounts/:accountId/statement", h.GetStatement)
		ledger.GET("/accounts/:accountId/holds", h.GetActiveHolds)
		ledger.GET("/journals/:journalId", h.GetJournal)
		ledger.GET("/trial-balance", h.GetTrialBalance)
		ledger.GET("/accounts/:accountId/audit-trail", h.GetAuditTrail)
		ledger.POST("/entries/search", h.SearchEntries)
		ledger.POST("/journals/search", h.SearchJournals)
	}
}

// GetBalance 查询账户余额。
func (h *Handler) GetBalance(c *gin.Context) {
	accountID := c.Param("accountId")
	currency := c.DefaultQuery("currency", "USD")

	result, err := h.querySvc.GetBalance(c.Request.Context(), accountID, currency)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetAccountSummary 获取用户所有账户汇总。
func (h *Handler) GetAccountSummary(c *gin.Context) {
	ownerID := c.Param("ownerId")

	result, err := h.querySvc.GetAccountSummary(c.Request.Context(), ownerID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// statementQuery 流水查询参数。
type statementQuery struct {
	Currency string `form:"currency" binding:"required"`
	Start    string `form:"start" binding:"required"`
	End      string `form:"end" binding:"required"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

// GetStatement 查询账户流水。
func (h *Handler) GetStatement(c *gin.Context) {
	accountID := c.Param("accountId")

	var q statementQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, err := time.Parse("2006-01-02", q.Start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format, use YYYY-MM-DD"})
		return
	}
	end, err := time.Parse("2006-01-02", q.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format, use YYYY-MM-DD"})
		return
	}

	result, err := h.querySvc.GetStatement(c.Request.Context(), accountID, q.Currency, start, end, q.Page, q.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetActiveHolds 查询活跃冻结记录。
func (h *Handler) GetActiveHolds(c *gin.Context) {
	accountID := c.Param("accountId")

	result, err := h.querySvc.GetActiveHolds(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetJournal 查询凭证详情。
func (h *Handler) GetJournal(c *gin.Context) {
	journalID := c.Param("journalId")

	result, err := h.querySvc.GetJournal(c.Request.Context(), journalID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetTrialBalance 生成试算平衡表。
func (h *Handler) GetTrialBalance(c *gin.Context) {
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	result, err := h.querySvc.GetTrialBalance(c.Request.Context(), date)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetAuditTrail 获取审计追踪。
func (h *Handler) GetAuditTrail(c *gin.Context) {
	accountID := c.Param("accountId")
	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	start, _ := time.Parse("2006-01-02", startStr)
	end, _ := time.Parse("2006-01-02", endStr)

	result, err := h.querySvc.GetAuditTrail(c.Request.Context(), accountID, start, end)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// SearchEntries 搜索分录。
func (h *Handler) SearchEntries(c *gin.Context) {
	var query domain.EntrySearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.querySvc.SearchEntries(c.Request.Context(), &query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// SearchJournals 搜索凭证。
func (h *Handler) SearchJournals(c *gin.Context) {
	var query domain.JournalSearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.querySvc.SearchJournals(c.Request.Context(), &query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
