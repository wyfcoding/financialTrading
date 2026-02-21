package application

import (
	"context"
	"github.com/wyfcoding/financialtrading/internal/riskgateway/domain"
	"github.com/wyfcoding/pkg/money"
	"github.com/wyfcoding/pkg/xerrors"
)

// 生成摘要：风控网关应用服务。
// 极致低延迟实现。

type RiskCheckService struct {
	repo domain.RiskRepository
}

func NewRiskCheckService(repo domain.RiskRepository) *RiskCheckService {
	return &RiskCheckService{repo: repo}
}

// PreTradeCheck 下单前风控检查。
func (s *RiskCheckService) PreTradeCheck(ctx context.Context, accountID string, orderAmount money.Money) (bool, error) {
	risk, ok := s.repo.GetAccountRisk(accountID)
	if !ok {
		return false, xerrors.NotFound("account risk config not found")
	}

	if !risk.CheckOrder(orderAmount) {
		return false, xerrors.PermissionDenied("risk limit exceeded")
	}

	return true, nil
}
