package http

import (
	"net/http"

	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	app *application.AuthApplicationService
}

func NewHandler(r *gin.Engine, app *application.AuthApplicationService) {
	h := &Handler{app: app}
	g := r.Group("/v1/auth")
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
}

func (h *Handler) Register(c *gin.Context) {
	var req struct{ Email, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.app.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user_id": id})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct{ Email, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, exp, err := h.app.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "type": "Bearer", "expires_at": exp})
}
