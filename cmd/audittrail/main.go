package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/audittrail/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	repo := newMemoryAuditRepository()
	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/audittrail")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.Success(c, gin.H{"status": "ok"})
		})

		v1.POST("/logs", func(c *gin.Context) {
			var req createAuditLogRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", err.Error())
				return
			}
			if req.Service == "" || req.Action == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", "service and action are required")
				return
			}
			occurredAt := req.OccurredAt
			if occurredAt.IsZero() {
				occurredAt = time.Now()
			}
			logItem := &domain.AuditLog{
				ID:           nonEmpty(req.ID, idgen.GenIDString()),
				TraceID:      req.TraceID,
				UserID:       req.UserID,
				Service:      req.Service,
				Action:       req.Action,
				Resource:     req.Resource,
				ResourceID:   req.ResourceID,
				IP:           req.IP,
				UserAgent:    req.UserAgent,
				Request:      req.Request,
				Response:     req.Response,
				Status:       nonEmpty(req.Status, "SUCCESS"),
				ErrorMessage: req.ErrorMessage,
				OccurredAt:   occurredAt,
			}
			if err := repo.Store(logItem); err != nil {
				response.Error(c, err)
				return
			}
			response.Success(c, logItem)
		})

		v1.GET("/logs", func(c *gin.Context) {
			filter := make(map[string]interface{})
			addQueryFilter(c, filter, "trace_id")
			addQueryFilter(c, filter, "user_id")
			addQueryFilter(c, filter, "service")
			addQueryFilter(c, filter, "action")
			addQueryFilter(c, filter, "resource")
			addQueryFilter(c, filter, "resource_id")
			addQueryFilter(c, filter, "status")

			if from := strings.TrimSpace(c.Query("from")); from != "" {
				t, err := parseTime(from)
				if err != nil {
					response.ErrorWithStatus(c, http.StatusBadRequest, "invalid from", err.Error())
					return
				}
				filter["from"] = t
			}
			if to := strings.TrimSpace(c.Query("to")); to != "" {
				t, err := parseTime(to)
				if err != nil {
					response.ErrorWithStatus(c, http.StatusBadRequest, "invalid to", err.Error())
					return
				}
				filter["to"] = t
			}

			items, err := repo.Query(filter)
			if err != nil {
				response.Error(c, err)
				return
			}
			response.Success(c, items)
		})
	}

	addr := os.Getenv("AUDITTRAIL_HTTP_ADDR")
	if addr == "" {
		addr = ":9118"
	}
	srv := server.NewGinServer(engine, addr, logger)

	go func() {
		if err := srv.Start(context.Background()); err != nil {
			slog.Error("server exit", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = srv.Stop(context.Background())
	slog.Info("service audittrail gracefully stopped")
}

type createAuditLogRequest struct {
	ID           string    `json:"id"`
	TraceID      string    `json:"trace_id"`
	UserID       string    `json:"user_id"`
	Service      string    `json:"service"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	Request      string    `json:"request"`
	Response     string    `json:"response"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type memoryAuditRepository struct {
	mu   sync.RWMutex
	logs []*domain.AuditLog
}

func newMemoryAuditRepository() *memoryAuditRepository {
	return &memoryAuditRepository{logs: make([]*domain.AuditLog, 0, 256)}
}

func (r *memoryAuditRepository) Store(logItem *domain.AuditLog) error {
	if logItem == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *logItem
	r.logs = append(r.logs, &cp)
	return nil
}

func (r *memoryAuditRepository) Query(filter map[string]interface{}) ([]*domain.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.AuditLog, 0, len(r.logs))
	for _, item := range r.logs {
		if !matchAuditFilter(item, filter) {
			continue
		}
		cp := *item
		result = append(result, &cp)
	}
	return result, nil
}

func matchAuditFilter(item *domain.AuditLog, filter map[string]interface{}) bool {
	if item == nil {
		return false
	}

	if v, ok := filter["trace_id"].(string); ok && v != "" && item.TraceID != v {
		return false
	}
	if v, ok := filter["user_id"].(string); ok && v != "" && item.UserID != v {
		return false
	}
	if v, ok := filter["service"].(string); ok && v != "" && item.Service != v {
		return false
	}
	if v, ok := filter["action"].(string); ok && v != "" && item.Action != v {
		return false
	}
	if v, ok := filter["resource"].(string); ok && v != "" && item.Resource != v {
		return false
	}
	if v, ok := filter["resource_id"].(string); ok && v != "" && item.ResourceID != v {
		return false
	}
	if v, ok := filter["status"].(string); ok && v != "" && item.Status != v {
		return false
	}
	if from, ok := filter["from"].(time.Time); ok && item.OccurredAt.Before(from) {
		return false
	}
	if to, ok := filter["to"].(time.Time); ok && item.OccurredAt.After(to) {
		return false
	}
	return true
}

func parseTime(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", raw)
}

func addQueryFilter(c *gin.Context, filter map[string]interface{}, key string) {
	if v := strings.TrimSpace(c.Query(key)); v != "" {
		filter[key] = v
	}
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
