// Package handler 市场监察服务 HTTP Handler（健康检查 + 管理 API）
// 生成摘要：
//  1. 提供 HTTP REST 端点用于运维管理和健康检查
//  2. 告警列表、规则管理、用户评分查询
//  3. 主要面向内部运维人员，非交易链路
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/application"
	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
)

// RegisterRoutes 注册 HTTP 路由
func RegisterRoutes(
	router *gin.Engine,
	cmdSvc *application.CommandService,
	querySvc *application.QueryService,
) {
	h := &httpHandler{cmdSvc: cmdSvc, querySvc: querySvc}

	v1 := router.Group("/api/v1/surveillance")
	{
		// 告警管理
		v1.GET("/alerts", h.listAlerts)
		v1.GET("/alerts/:id", h.getAlert)
		v1.POST("/alerts/:id/review", h.reviewAlert)

		// 规则管理
		v1.GET("/rules", h.listRules)
		v1.GET("/rules/:id", h.getRule)

		// 用户评分
		v1.GET("/users/:user_id/score", h.getUserScore)

		// 系统状态
		v1.GET("/stats", h.getStats)
	}
}

type httpHandler struct {
	cmdSvc   *application.CommandService
	querySvc *application.QueryService
}

// listAlerts 列表查询告警
func (h *httpHandler) listAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := application.ListAlertsQuery{
		UserID:   c.Query("user_id"),
		Symbol:   c.Query("symbol"),
		Page:     page,
		PageSize: pageSize,
	}

	if statusStr := c.Query("status"); statusStr != "" {
		s, _ := strconv.Atoi(statusStr)
		st := domain.AlertStatus(s)
		query.Status = &st
	}
	if sevStr := c.Query("severity"); sevStr != "" {
		s, _ := strconv.Atoi(sevStr)
		sev := domain.AlertSeverity(s)
		query.Severity = &sev
	}

	alerts, total, err := h.querySvc.ListAlerts(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "total": total})
}

// getAlert 查询告警详情
func (h *httpHandler) getAlert(c *gin.Context) {
	alert, err := h.querySvc.GetAlert(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alert": alert})
}

// reviewAlertReq HTTP 审核请求体
type reviewAlertReq struct {
	ReviewerID string `json:"reviewer_id" binding:"required"`
	Confirmed  bool   `json:"confirmed"`
	Comment    string `json:"comment"`
}

// reviewAlert 审核告警
func (h *httpHandler) reviewAlert(c *gin.Context) {
	var req reviewAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd := application.ReviewAlertCmd{
		AlertID:    c.Param("id"),
		ReviewerID: req.ReviewerID,
		Confirmed:  req.Confirmed,
		Comment:    req.Comment,
	}
	if err := h.cmdSvc.ReviewAlert(c.Request.Context(), cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// listRules 列出规则
func (h *httpHandler) listRules(c *gin.Context) {
	query := application.ListRulesQuery{
		EnabledOnly: c.Query("enabled_only") == "true",
	}
	rules, err := h.querySvc.ListRules(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// getRule 获取规则详情
func (h *httpHandler) getRule(c *gin.Context) {
	rule, err := h.querySvc.GetRule(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": rule})
}

// getUserScore 获取用户评分
func (h *httpHandler) getUserScore(c *gin.Context) {
	score, err := h.querySvc.GetUserScore(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "score not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"score": score})
}

// getStats 系统统计
func (h *httpHandler) getStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   "marketsurveillance",
		"status":    "running",
		"timestamp": time.Now().Unix(),
	})
}
