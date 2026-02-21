package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/collateral/domain"
	"github.com/wyfcoding/pkg/response"
	"github.com/wyfcoding/pkg/server"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("COLLATERAL_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/financial_collateral?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&collateralAssetPO{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := &gormCollateralRepository{db: db}
	engine := server.NewDefaultGinEngine(gin.Recovery())
	v1 := engine.Group("/api/v1/collateral")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.Success(c, gin.H{"status": "ok"})
		})

		v1.POST("/assets", func(c *gin.Context) {
			var req upsertCollateralAssetRequest
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", bindErr.Error())
				return
			}
			if req.AccountID == "" || req.Symbol == "" || req.Quantity == "" || req.MarketPrice == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request", "account_id/symbol/quantity/market_price are required")
				return
			}
			quantity, err := decimal.NewFromString(req.Quantity)
			if err != nil || quantity.IsNegative() {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid quantity", "quantity must be non-negative decimal")
				return
			}
			marketPrice, err := decimal.NewFromString(req.MarketPrice)
			if err != nil || marketPrice.IsNegative() {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid market_price", "market_price must be non-negative decimal")
				return
			}
			haircut, err := decimal.NewFromString(nonEmpty(req.Haircut, "0"))
			if err != nil || haircut.IsNegative() || haircut.GreaterThan(decimal.NewFromInt(1)) {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid haircut", "haircut must be between 0 and 1")
				return
			}
			asset := &domain.CollateralAsset{
				AccountID:   strings.TrimSpace(req.AccountID),
				Symbol:      strings.ToUpper(strings.TrimSpace(req.Symbol)),
				Quantity:    quantity,
				MarketPrice: marketPrice,
				Haircut:     haircut,
				AssetType:   nonEmpty(strings.ToUpper(strings.TrimSpace(req.AssetType)), "CASH"),
				UpdatedAt:   time.Now().UTC(),
			}
			if err := repo.Save(asset); err != nil {
				response.Error(c, err)
				return
			}
			response.Success(c, asset)
		})

		v1.GET("/assets", func(c *gin.Context) {
			accountID := strings.TrimSpace(c.Query("account_id"))
			if accountID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", "account_id is required")
				return
			}
			items, err := repo.GetAccountAssets(accountID)
			if err != nil {
				response.Error(c, err)
				return
			}
			response.Success(c, items)
		})

		v1.GET("/effective", func(c *gin.Context) {
			accountID := strings.TrimSpace(c.Query("account_id"))
			if accountID == "" {
				response.ErrorWithStatus(c, http.StatusBadRequest, "invalid account_id", "account_id is required")
				return
			}
			items, err := repo.GetAccountAssets(accountID)
			if err != nil {
				response.Error(c, err)
				return
			}
			total := decimal.Zero
			for _, item := range items {
				total = total.Add(item.EffectiveValue())
			}
			response.Success(c, gin.H{
				"account_id":            accountID,
				"total_effective_value": total.String(),
				"assets":                items,
			})
		})
	}

	addr := os.Getenv("COLLATERAL_HTTP_ADDR")
	if addr == "" {
		addr = ":9120"
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
	slog.Info("service collateral gracefully stopped")
}

type upsertCollateralAssetRequest struct {
	AccountID   string `json:"account_id"`
	Symbol      string `json:"symbol"`
	Quantity    string `json:"quantity"`
	MarketPrice string `json:"market_price"`
	Haircut     string `json:"haircut"`
	AssetType   string `json:"asset_type"`
}

type collateralAssetPO struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement"`
	AccountID   string          `gorm:"type:varchar(64);not null;index:idx_collateral_account_symbol,unique"`
	Symbol      string          `gorm:"type:varchar(64);not null;index:idx_collateral_account_symbol,unique"`
	Quantity    decimal.Decimal `gorm:"type:decimal(32,16);not null"`
	MarketPrice decimal.Decimal `gorm:"type:decimal(32,16);not null"`
	Haircut     decimal.Decimal `gorm:"type:decimal(10,8);not null"`
	AssetType   string          `gorm:"type:varchar(32);not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (collateralAssetPO) TableName() string { return "collateral_assets" }

type gormCollateralRepository struct {
	db *gorm.DB
}

func (r *gormCollateralRepository) Save(asset *domain.CollateralAsset) error {
	if asset == nil {
		return nil
	}
	po := collateralAssetPO{
		ID:          asset.ID,
		AccountID:   asset.AccountID,
		Symbol:      asset.Symbol,
		Quantity:    asset.Quantity,
		MarketPrice: asset.MarketPrice,
		Haircut:     asset.Haircut,
		AssetType:   asset.AssetType,
		UpdatedAt:   asset.UpdatedAt,
	}
	if po.UpdatedAt.IsZero() {
		po.UpdatedAt = time.Now().UTC()
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}, {Name: "symbol"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "market_price", "haircut", "asset_type", "updated_at"}),
	}).Create(&po).Error; err != nil {
		return err
	}
	asset.ID = po.ID
	asset.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *gormCollateralRepository) GetAccountAssets(accountID string) ([]*domain.CollateralAsset, error) {
	var rows []collateralAssetPO
	if err := r.db.Where("account_id = ?", accountID).Order("symbol asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.CollateralAsset, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domain.CollateralAsset{
			ID:          row.ID,
			AccountID:   row.AccountID,
			Symbol:      row.Symbol,
			Quantity:    row.Quantity,
			MarketPrice: row.MarketPrice,
			Haircut:     row.Haircut,
			AssetType:   row.AssetType,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return result, nil
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
