package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/messagequeue"
)

// PricingCommandService 处理定价相关的命令操作
// 使用 Outbox 发布领域事件

type PricingCommandService struct {
	repo      domain.PricingRepository
	publisher messagequeue.EventPublisher
}

// NewPricingCommandService 创建新的 PricingCommandService 实例
func NewPricingCommandService(repo domain.PricingRepository, publisher messagequeue.EventPublisher) *PricingCommandService {
	return &PricingCommandService{
		repo:      repo,
		publisher: publisher,
	}
}

// PriceOption 期权定价
func (c *PricingCommandService) PriceOption(ctx context.Context, cmd PriceOptionCommand) (*domain.PricingResult, error) {
	if cmd.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if cmd.PricingModel == "" {
		cmd.PricingModel = "BlackScholes"
	}

	var result *domain.PricingResult
	var greeks domain.Greeks

	err := c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		// 根据定价模型选择不同的定价方法
		var price float64
		timeToExpiry := float64(cmd.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
		if timeToExpiry < 0 {
			timeToExpiry = 0
		}
		switch cmd.PricingModel {
		case "BlackScholes":
			input := domain.BlackScholesInput{
				S: cmd.UnderlyingPrice,
				K: cmd.StrikePrice,
				T: timeToExpiry,
				R: cmd.RiskFreeRate,
				V: cmd.Volatility,
			}
			bs := domain.CalculateBlackScholes(domain.OptionType(cmd.OptionType), input)
			price = bs.Price.InexactFloat64()
			greeks = domain.Greeks{
				Delta: bs.Delta,
				Gamma: bs.Gamma,
				Theta: bs.Theta,
				Vega:  bs.Vega,
				Rho:   bs.Rho,
			}
		case "LongstaffSchwartz":
			pricer := domain.NewLSMPricer()
			isPut := strings.EqualFold(cmd.OptionType, string(domain.OptionTypePut))
			lsmPrice, calcErr := pricer.Price(domain.AmericanOptionParams{
				S0:    cmd.UnderlyingPrice,
				K:     cmd.StrikePrice,
				T:     timeToExpiry,
				R:     cmd.RiskFreeRate,
				Sigma: cmd.Volatility,
				IsPut: isPut,
				Paths: 2000,
				Steps: 50,
			})
			if calcErr != nil {
				return calcErr
			}
			price = lsmPrice
			greeks = domain.Greeks{}
		default:
			input := domain.BlackScholesInput{
				S: cmd.UnderlyingPrice,
				K: cmd.StrikePrice,
				T: timeToExpiry,
				R: cmd.RiskFreeRate,
				V: cmd.Volatility,
			}
			bs := domain.CalculateBlackScholes(domain.OptionType(cmd.OptionType), input)
			price = bs.Price.InexactFloat64()
			greeks = domain.Greeks{
				Delta: bs.Delta,
				Gamma: bs.Gamma,
				Theta: bs.Theta,
				Vega:  bs.Vega,
				Rho:   bs.Rho,
			}
		}

		// 创建定价结果
		result = &domain.PricingResult{
			Symbol:          cmd.Symbol,
			OptionPrice:     decimal.NewFromFloat(price),
			UnderlyingPrice: decimal.NewFromFloat(cmd.UnderlyingPrice),
			Delta:           greeks.Delta,
			Gamma:           greeks.Gamma,
			Theta:           greeks.Theta,
			Vega:            greeks.Vega,
			Rho:             greeks.Rho,
			CalculatedAt:    time.Now().Unix(),
			PricingModel:    cmd.PricingModel,
		}

		// 保存定价结果
		if err := c.repo.SavePricingResult(txCtx, result); err != nil {
			return err
		}

		if c.publisher == nil {
			return nil
		}

		optionEvent := domain.OptionPricedEvent{
			Symbol:          cmd.Symbol,
			OptionType:      domain.OptionType(cmd.OptionType),
			StrikePrice:     cmd.StrikePrice,
			ExpiryDate:      cmd.ExpiryDate,
			OptionPrice:     price,
			UnderlyingPrice: cmd.UnderlyingPrice,
			Volatility:      cmd.Volatility,
			RiskFreeRate:    cmd.RiskFreeRate,
			DividendYield:   cmd.DividendYield,
			PricingModel:    cmd.PricingModel,
			CalculatedAt:    result.CalculatedAt,
			OccurredOn:      time.Now(),
		}
		if err := c.publisher.PublishInTx(txCtx, tx, domain.OptionPricedEventType, cmd.Symbol, optionEvent); err != nil {
			return err
		}

		greeksEvent := domain.GreeksCalculatedEvent{
			Symbol:          cmd.Symbol,
			OptionType:      domain.OptionType(cmd.OptionType),
			StrikePrice:     cmd.StrikePrice,
			ExpiryDate:      cmd.ExpiryDate,
			UnderlyingPrice: cmd.UnderlyingPrice,
			Delta:           greeks.Delta.InexactFloat64(),
			Gamma:           greeks.Gamma.InexactFloat64(),
			Theta:           greeks.Theta.InexactFloat64(),
			Vega:            greeks.Vega.InexactFloat64(),
			Rho:             greeks.Rho.InexactFloat64(),
			CalculatedAt:    result.CalculatedAt,
			OccurredOn:      time.Now(),
		}
		return c.publisher.PublishInTx(txCtx, tx, domain.GreeksCalculatedEventType, cmd.Symbol, greeksEvent)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateVolatility 更新波动率
func (c *PricingCommandService) UpdateVolatility(ctx context.Context, cmd UpdateVolatilityCommand) error {
	return nil
}

// ChangePricingModel 变更定价模型
func (c *PricingCommandService) ChangePricingModel(ctx context.Context, cmd ChangePricingModelCommand) error {
	return nil
}

// BatchPriceOptions 批量定价
func (c *PricingCommandService) BatchPriceOptions(ctx context.Context, cmd BatchPriceOptionsCommand) (*BatchPricingResult, error) {
	results := make([]*domain.PricingResult, 0, len(cmd.Contracts))
	successCount := 0
	failureCount := 0
	totalTime := 0.0

	for _, contract := range cmd.Contracts {
		startTime := time.Now()
		result, err := c.PriceOption(ctx, contract)
		totalTime += time.Since(startTime).Seconds()

		if err != nil {
			failureCount++
			continue
		}

		results = append(results, result)
		successCount++
	}

	avg := 0.0
	if len(cmd.Contracts) > 0 {
		avg = totalTime / float64(len(cmd.Contracts))
	}

	if c.publisher != nil {
		_ = c.publisher.Publish(ctx, domain.BatchPricingCompletedEventType, cmd.BatchID, domain.BatchPricingCompletedEvent{
			BatchID:        cmd.BatchID,
			Symbols:        extractSymbols(cmd.Contracts),
			TotalContracts: len(cmd.Contracts),
			SuccessCount:   successCount,
			FailureCount:   failureCount,
			AverageTime:    avg,
			CompletedAt:    time.Now().Unix(),
			OccurredOn:     time.Now(),
		})
	}

	return &BatchPricingResult{
		BatchID:      cmd.BatchID,
		Results:      results,
		SuccessCount: successCount,
		FailureCount: failureCount,
		AverageTime:  avg,
	}, nil
}

// 辅助函数：提取合约符号
func extractSymbols(contracts []PriceOptionCommand) []string {
	symbols := make([]string, 0, len(contracts))
	seen := make(map[string]bool)

	for _, contract := range contracts {
		if !seen[contract.Symbol] {
			symbols = append(symbols, contract.Symbol)
			seen[contract.Symbol] = true
		}
	}

	return symbols
}
