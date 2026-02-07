package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// InterestAccrualJob 负责定期为所有杠杆账户计息。
type InterestAccrualJob struct {
	cmdService *AccountCommandService
	query      *AccountQueryService
	logger     *slog.Logger
	interval   time.Duration
	dailyRate  decimal.Decimal // 日利率，例如 0.0005 (0.05%)
}

func NewInterestAccrualJob(
	cmdService *AccountCommandService,
	query *AccountQueryService,
	logger *slog.Logger,
) *InterestAccrualJob {
	return &InterestAccrualJob{
		cmdService: cmdService,
		query:      query,
		logger:     logger,
		interval:   1 * time.Hour,                // 每小时计息一次
		dailyRate:  decimal.NewFromFloat(0.0005), // 万分之五
	}
}

func (j *InterestAccrualJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.logger.Info("Interest Accrual Job started", "interval", j.interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.run(ctx)
		}
	}
}

func (j *InterestAccrualJob) run(ctx context.Context) {
	// 每小时利率
	hourlyRate := j.dailyRate.Div(decimal.NewFromInt(24))

	pageToken := 0
	for {
		dtos, nextToken, err := j.query.ListAccounts(ctx, "MARGIN", 100, pageToken)
		if err != nil {
			j.logger.Error("failed to list margin accounts", "error", err)
			break
		}

		for _, acc := range dtos {
			// 只有有借款的才计息
			borrowed, _ := decimal.NewFromString(acc.BorrowedAmount)
			if borrowed.IsPositive() {
				err := j.cmdService.AccrueInterest(ctx, AccrueInterestCommand{
					AccountID: acc.AccountID,
					Rate:      hourlyRate,
				})
				if err != nil {
					j.logger.Error("failed to accrue interest", "account_id", acc.AccountID, "error", err)
				}
			}
		}

		if nextToken == 0 {
			break
		}
		pageToken = nextToken
	}
}
