package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/admin/application"
)

type Handler struct {
	app *application.AdminService
}

func NewHandler(r *gin.Engine, app *application.AdminService) {
	h := &Handler{app: app}

	g := r.Group("/v1/admin")
	{
		g.POST("/login", h.Login)
		// Other routes would be protected by middleware
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.app.Login(c.Request.Context(), application.LoginCommand{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}
