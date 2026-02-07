// 生成摘要：充值和提现命令服务，实现充值/提现业务逻辑。
// 迁移自 ecommerce/payment，现在属于 financialTrading/account 服务。
// 集成 Account 聚合进行账户余额操作。
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// DepositWithdrawalCommandService 充值提现命令服务
type DepositWithdrawalCommandService struct {
	accountRepo         domain.AccountRepository
	depositOrderRepo    domain.DepositOrderRepository
	withdrawalOrderRepo domain.WithdrawalOrderRepository
	publisher           domain.EventPublisher
	idGenerator         idgen.Generator
	logger              *slog.Logger
}

// NewDepositWithdrawalCommandService 创建充值提现命令服务
func NewDepositWithdrawalCommandService(
	accountRepo domain.AccountRepository,
	depositOrderRepo domain.DepositOrderRepository,
	withdrawalOrderRepo domain.WithdrawalOrderRepository,
	publisher domain.EventPublisher,
	idGenerator idgen.Generator,
	logger *slog.Logger,
) *DepositWithdrawalCommandService {
	return &DepositWithdrawalCommandService{
		accountRepo:         accountRepo,
		depositOrderRepo:    depositOrderRepo,
		withdrawalOrderRepo: withdrawalOrderRepo,
		publisher:           publisher,
		idGenerator:         idGenerator,
		logger:              logger,
	}
}

// CreateDepositRequest 创建充值请求
type CreateDepositRequest struct {
	UserID      string
	AccountID   string
	Amount      decimal.Decimal
	Currency    string
	GatewayType domain.GatewayType
}

// CreateDepositResponse 创建充值响应
type CreateDepositResponse struct {
	DepositNo  string `json:"deposit_no"`
	PaymentURL string `json:"payment_url"`
}

// CreateDeposit 创建充值订单
func (s *DepositWithdrawalCommandService) CreateDeposit(ctx context.Context, req *CreateDepositRequest) (*CreateDepositResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	// 验证账户存在
	account, err := s.accountRepo.Get(ctx, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, errors.New("account not found")
	}

	depositOrder := domain.NewDepositOrder(req.UserID, req.AccountID, req.Amount, req.Currency, req.GatewayType, s.idGenerator)

	// TODO: 调用外部网关获取支付链接
	depositOrder.PaymentURL = fmt.Sprintf("https://pay.example.com/deposit/%s", depositOrder.DepositNo)

	if err := s.depositOrderRepo.Save(ctx, depositOrder); err != nil {
		return nil, fmt.Errorf("failed to save deposit order: %w", err)
	}

	s.logger.Info("deposit order created", "deposit_no", depositOrder.DepositNo, "user_id", req.UserID, "amount", req.Amount)

	return &CreateDepositResponse{
		DepositNo:  depositOrder.DepositNo,
		PaymentURL: depositOrder.PaymentURL,
	}, nil
}

// ConfirmDepositRequest 确认充值请求 (网关回调)
type ConfirmDepositRequest struct {
	DepositNo     string
	TransactionID string
	ThirdPartyNo  string
}

// ConfirmDeposit 确认充值并入账
func (s *DepositWithdrawalCommandService) ConfirmDeposit(ctx context.Context, req *ConfirmDepositRequest) error {
	depositOrder, err := s.depositOrderRepo.FindByDepositNo(ctx, req.DepositNo)
	if err != nil {
		return fmt.Errorf("failed to find deposit order: %w", err)
	}
	if depositOrder == nil {
		return errors.New("deposit order not found")
	}

	// 幂等检查
	if depositOrder.Status == domain.DepositStatusCompleted {
		return nil
	}

	return s.depositOrderRepo.WithTx(ctx, func(txCtx context.Context) error {
		// 1. 确认充值订单
		if err := depositOrder.Confirm(txCtx, req.TransactionID, req.ThirdPartyNo); err != nil {
			return fmt.Errorf("failed to confirm deposit: %w", err)
		}

		// 2. 获取账户并入账
		account, err := s.accountRepo.Get(txCtx, depositOrder.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}
		if account == nil {
			return errors.New("account not found")
		}

		account.Deposit(depositOrder.Amount)
		if err := s.accountRepo.Save(txCtx, account); err != nil {
			return fmt.Errorf("failed to save account: %w", err)
		}

		// 3. 完成充值订单
		if err := depositOrder.Complete(txCtx); err != nil {
			return fmt.Errorf("failed to complete deposit: %w", err)
		}
		if err := s.depositOrderRepo.Update(txCtx, depositOrder); err != nil {
			return fmt.Errorf("failed to update deposit order: %w", err)
		}

		s.logger.Info("deposit confirmed and completed", "deposit_no", req.DepositNo, "account_id", depositOrder.AccountID, "amount", depositOrder.Amount)
		return nil
	})
}

// CreateWithdrawalRequest 创建提现请求
type CreateWithdrawalRequest struct {
	UserID        string
	AccountID     string
	Amount        decimal.Decimal
	Fee           decimal.Decimal
	Currency      string
	BankAccountNo string
	BankName      string
	BankHolder    string
}

// CreateWithdrawalResponse 创建提现响应
type CreateWithdrawalResponse struct {
	WithdrawalNo string `json:"withdrawal_no"`
	Status       string `json:"status"`
}

// CreateWithdrawal 创建提现订单
func (s *DepositWithdrawalCommandService) CreateWithdrawal(ctx context.Context, req *CreateWithdrawalRequest) (*CreateWithdrawalResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}
	if req.Amount.LessThanOrEqual(req.Fee) {
		return nil, errors.New("amount must be greater than fee")
	}

	// 1. 获取账户并检查余额
	account, err := s.accountRepo.Get(ctx, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, errors.New("account not found")
	}

	// 2. 冻结资金
	if !account.Freeze(req.Amount, "withdrawal") {
		return nil, errors.New("insufficient available balance")
	}

	withdrawalOrder := domain.NewWithdrawalOrder(req.UserID, req.AccountID, req.Amount, req.Fee, req.Currency, req.BankAccountNo, req.BankName, req.BankHolder, s.idGenerator)

	var response *CreateWithdrawalResponse
	err = s.withdrawalOrderRepo.WithTx(ctx, func(txCtx context.Context) error {
		// 保存账户冻结状态
		if err := s.accountRepo.Save(txCtx, account); err != nil {
			return fmt.Errorf("failed to save account: %w", err)
		}

		// 保存提现订单
		if err := s.withdrawalOrderRepo.Save(txCtx, withdrawalOrder); err != nil {
			return fmt.Errorf("failed to save withdrawal order: %w", err)
		}

		// 自动进入审核状态
		if err := withdrawalOrder.StartAudit(txCtx); err != nil {
			return fmt.Errorf("failed to start audit: %w", err)
		}
		if err := s.withdrawalOrderRepo.Update(txCtx, withdrawalOrder); err != nil {
			return fmt.Errorf("failed to update withdrawal order: %w", err)
		}

		s.logger.Info("withdrawal order created", "withdrawal_no", withdrawalOrder.WithdrawalNo, "user_id", req.UserID, "amount", req.Amount)
		response = &CreateWithdrawalResponse{
			WithdrawalNo: withdrawalOrder.WithdrawalNo,
			Status:       string(withdrawalOrder.Status),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// AuditWithdrawalRequest 审核提现请求
type AuditWithdrawalRequest struct {
	WithdrawalNo string
	Approved     bool
	Auditor      string
	Remark       string
}

// AuditWithdrawal 审核提现
func (s *DepositWithdrawalCommandService) AuditWithdrawal(ctx context.Context, req *AuditWithdrawalRequest) error {
	withdrawalOrder, err := s.withdrawalOrderRepo.FindByWithdrawalNo(ctx, req.WithdrawalNo)
	if err != nil {
		return fmt.Errorf("failed to find withdrawal order: %w", err)
	}
	if withdrawalOrder == nil {
		return errors.New("withdrawal order not found")
	}

	return s.withdrawalOrderRepo.WithTx(ctx, func(txCtx context.Context) error {
		if req.Approved {
			if err := withdrawalOrder.Approve(txCtx, req.Auditor, req.Remark); err != nil {
				return fmt.Errorf("failed to approve withdrawal: %w", err)
			}
		} else {
			// 拒绝时解冻资金
			account, err := s.accountRepo.Get(txCtx, withdrawalOrder.AccountID)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}
			if account != nil {
				account.Unfreeze(withdrawalOrder.Amount)
				if err := s.accountRepo.Save(txCtx, account); err != nil {
					return fmt.Errorf("failed to save account: %w", err)
				}
			}

			if err := withdrawalOrder.Reject(txCtx, req.Auditor, req.Remark); err != nil {
				return fmt.Errorf("failed to reject withdrawal: %w", err)
			}
		}

		if err := s.withdrawalOrderRepo.Update(txCtx, withdrawalOrder); err != nil {
			return fmt.Errorf("failed to update withdrawal order: %w", err)
		}

		s.logger.Info("withdrawal audited", "withdrawal_no", req.WithdrawalNo, "approved", req.Approved, "auditor", req.Auditor)
		return nil
	})
}

// ProcessWithdrawal 处理提现 (执行打款)
func (s *DepositWithdrawalCommandService) ProcessWithdrawal(ctx context.Context, withdrawalNo string) error {
	withdrawalOrder, err := s.withdrawalOrderRepo.FindByWithdrawalNo(ctx, withdrawalNo)
	if err != nil {
		return fmt.Errorf("failed to find withdrawal order: %w", err)
	}
	if withdrawalOrder == nil {
		return errors.New("withdrawal order not found")
	}

	if withdrawalOrder.Status != domain.WithdrawalStatusApproved {
		return errors.New("withdrawal is not approved")
	}

	return s.withdrawalOrderRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := withdrawalOrder.StartProcessing(txCtx); err != nil {
			return fmt.Errorf("failed to start processing: %w", err)
		}
		if err := s.withdrawalOrderRepo.Update(txCtx, withdrawalOrder); err != nil {
			return fmt.Errorf("failed to update withdrawal order: %w", err)
		}

		// TODO: 调用银行/网关打款
		gatewayRef := fmt.Sprintf("BANK-%d", time.Now().UnixNano())

		// 从冻结余额扣除
		account, err := s.accountRepo.Get(txCtx, withdrawalOrder.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}
		if account != nil {
			account.DeductFrozen(withdrawalOrder.Amount)
			if err := s.accountRepo.Save(txCtx, account); err != nil {
				return fmt.Errorf("failed to save account: %w", err)
			}
		}

		if err := withdrawalOrder.Complete(txCtx, gatewayRef); err != nil {
			return fmt.Errorf("failed to complete withdrawal: %w", err)
		}
		if err := s.withdrawalOrderRepo.Update(txCtx, withdrawalOrder); err != nil {
			return fmt.Errorf("failed to update withdrawal order: %w", err)
		}

		s.logger.Info("withdrawal processed and completed", "withdrawal_no", withdrawalNo, "gateway_ref", gatewayRef)
		return nil
	})
}
