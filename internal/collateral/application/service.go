// Package application 抵押品管理应用服务
package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/collateral/domain"
	"github.com/wyfcoding/pkg/idgen"
	"gorm.io/gorm"
)

// MarketPriceService 市场价格服务接口 (External Dependency)
type MarketPriceService interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}

type CollateralService struct {
	collateralRepo domain.CollateralRepository
	haircutRepo    domain.HaircutRepository
	allocRepo      domain.AllocationRepository
	priceSvc       MarketPriceService
	logger         *slog.Logger
}

func NewCollateralService(
	collateralRepo domain.CollateralRepository,
	haircutRepo domain.HaircutRepository,
	allocRepo domain.AllocationRepository,
	priceSvc MarketPriceService,
	logger *slog.Logger,
) *CollateralService {
	return &CollateralService{
		collateralRepo: collateralRepo,
		haircutRepo:    haircutRepo,
		allocRepo:      allocRepo,
		priceSvc:       priceSvc,
		logger:         logger.With("module", "collateral_service"),
	}
}

// DepositCollateral 存入抵押品
func (s *CollateralService) DepositCollateral(ctx context.Context, cmd DepositCmd) (string, error) {
	// 获取或创建资产记录
	asset, err := s.collateralRepo.GetByAccountAndSymbol(ctx, cmd.AccountID, cmd.Symbol)
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	if asset == nil {
		asset = domain.NewCollateralAsset(cmd.AccountID, cmd.AssetType, cmd.Symbol, decimal.Zero, cmd.Currency)
		asset.AssetID = fmt.Sprintf("COL%s", idgen.GenIDString())
	}

	// 增加数量
	asset.Deposit(cmd.Quantity)

	// 立即重估值
	if err := s.updateValuationInternal(ctx, asset); err != nil {
		s.logger.WarnContext(ctx, "failed to update valuation on deposit", "error", err)
		// 不阻断存入，但可能有估值延迟
	}

	if err := s.collateralRepo.Save(ctx, asset); err != nil {
		return "", err
	}

	return asset.AssetID, nil
}

// WithdrawCollateral 提取抵押品
func (s *CollateralService) WithdrawCollateral(ctx context.Context, cmd WithdrawCmd) error {
	asset, err := s.collateralRepo.GetByAssetID(ctx, cmd.AssetID)
	if err != nil {
		return err
	}

	// 检查是否被锁定/分配
	allocations, _ := s.allocRepo.ListByAssetID(ctx, asset.AssetID)
	lockedAmount := decimal.Zero
	for _, a := range allocations {
		lockedAmount = lockedAmount.Add(a.Amount)
	}

	// 计算提取后的剩余估值是否足够覆盖义务（此处简化逻辑，仅检查数量）
	// 实际应调用 Risk Service 检查 Withdrawal 后的Margin Ratio

	// 执行提取
	if err := asset.Withdraw(cmd.Quantity); err != nil {
		return err
	}

	// 重估值
	_ = s.updateValuationInternal(ctx, asset)

	return s.collateralRepo.Save(ctx, asset)
}

// ValuationUpdate 每日/实时盯市估值
func (s *CollateralService) ValuationUpdate(ctx context.Context, accountID string) error {
	assets, err := s.collateralRepo.ListByAccount(ctx, accountID)
	if err != nil {
		return err
	}

	for _, asset := range assets {
		if err := s.updateValuationInternal(ctx, asset); err != nil {
			s.logger.ErrorContext(ctx, "valuation failed", "asset_id", asset.AssetID, "error", err)
			continue
		}
		if err := s.collateralRepo.Save(ctx, asset); err != nil {
			continue
		}
	}
	return nil
}

func (s *CollateralService) updateValuationInternal(ctx context.Context, asset *domain.CollateralAsset) error {
	// 1. 获取最新市价
	price, err := s.priceSvc.GetPrice(ctx, asset.Symbol)
	if err != nil {
		// 假如是现金，价格始终为1
		if asset.AssetType == domain.AssetTypeCash {
			price = decimal.NewFromInt(1)
		} else {
			return err
		}
	}

	// 2. 获取 Haircut 规则
	haircutObj, err := s.haircutRepo.GetSchedule(ctx, asset.AssetType, asset.Symbol)
	haircutRate := decimal.Zero
	if err == nil {
		haircutRate = haircutObj.BaseHaircut
		// TODO: 应用 VolatilityAdj
	} else {
		// 默认 Haircut 或报错
		haircutRate = decimal.NewFromFloat(0.1) // 默认 10%
	}

	// 3. 更新
	asset.UpdateValuation(price, haircutRate)
	return nil
}

// GetAccountCollateralValue 获取账户总抵押价值
func (s *CollateralService) GetAccountCollateralValue(ctx context.Context, accountID, currency string) (decimal.Decimal, error) {
	return s.collateralRepo.GetTotalCollateralValue(ctx, accountID, currency)
}

type DepositCmd struct {
	AccountID string
	AssetType domain.AssetType
	Symbol    string
	Quantity  decimal.Decimal
	Currency  string
}

type WithdrawCmd struct {
	AssetID  string
	Quantity decimal.Decimal
}
