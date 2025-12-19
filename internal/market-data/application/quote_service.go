package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/market-data/domain"
	"github.com/wyfcoding/pkg/logging"
)

// GetLatestQuoteRequest 获取最新行情请求 DTO
// 用于接收获取最新行情的请求参数
type GetLatestQuoteRequest struct {
	Symbol string // 交易对符号，例如 "BTC/USD"
}

// QuoteDTO 行情数据 DTO
// 用于向外层返回行情数据
type QuoteDTO struct {
	Symbol    string // 交易对符号
	BidPrice  string // 买一价
	AskPrice  string // 卖一价
	BidSize   string // 买一量
	AskSize   string // 卖一量
	LastPrice string // 最新成交价
	LastSize  string // 最新成交量
	Timestamp int64  // 时间戳（毫秒）
	Source    string // 数据来源
}

// QuoteApplicationService 行情应用服务
// 处理行情相关的用例逻辑
type QuoteApplicationService struct {
	quoteRepo domain.QuoteRepository
}

// NewQuoteApplicationService 创建行情应用服务
func NewQuoteApplicationService(quoteRepo domain.QuoteRepository) *QuoteApplicationService {
	return &QuoteApplicationService{
		quoteRepo: quoteRepo,
	}
}

// GetLatestQuote 获取最新行情
// 用例流程：
// 1. 验证交易对符号
// 2. 从仓储获取最新行情
// 3. 转换为 DTO 返回
func (qas *QuoteApplicationService) GetLatestQuote(ctx context.Context, req *GetLatestQuoteRequest) (*QuoteDTO, error) {
	// 记录操作开始
	defer logging.Info(ctx, "GetLatestQuote completed", "symbol", req.Symbol)

	// 验证输入
	if req.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// 从仓储获取最新行情
	quote, err := qas.quoteRepo.GetLatest(ctx, req.Symbol)
	if err != nil {
		logging.Error(ctx, "Failed to get latest quote",
			"symbol", req.Symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get latest quote: %w", err)
	}

	if quote == nil {
		logging.Warn(ctx, "Quote not found",
			"symbol", req.Symbol,
		)
		return nil, fmt.Errorf("quote not found for symbol: %s", req.Symbol)
	}

	// 转换为 DTO
	return &QuoteDTO{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp,
		Source:    quote.Source,
	}, nil
}

// SaveQuote 保存行情数据
// 用例流程：
// 1. 验证行情数据
// 2. 创建领域对象
// 3. 保存到仓储
func (qas *QuoteApplicationService) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	// 验证输入
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	// 创建领域对象
	quote := domain.NewQuote(symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize, timestamp, source)

	// 保存到仓储
	if err := qas.quoteRepo.Save(ctx, quote); err != nil {
		logging.Error(ctx, "Failed to save quote",
			"symbol", symbol,
			"error", err,
		)
		return fmt.Errorf("failed to save quote: %w", err)
	}

	logging.Debug(ctx, "Quote saved successfully",
		"symbol", symbol,
		"timestamp", timestamp,
	)

	return nil
}

// GetHistoricalQuotes 获取历史行情
// 用例流程：
// 1. 验证时间范围
// 2. 从仓储获取历史行情
// 3. 转换为 DTO 列表返回
func (qas *QuoteApplicationService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	// 验证输入
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if startTime >= endTime {
		return nil, fmt.Errorf("startTime must be less than endTime")
	}

	// 从仓储获取历史行情
	quotes, err := qas.quoteRepo.GetHistory(ctx, symbol, startTime, endTime)
	if err != nil {
		logging.Error(ctx, "Failed to get historical quotes",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get historical quotes: %w", err)
	}

	// 转换为 DTO 列表
	dtos := make([]*QuoteDTO, 0, len(quotes))
	for _, quote := range quotes {
		dtos = append(dtos, &QuoteDTO{
			Symbol:    quote.Symbol,
			BidPrice:  quote.BidPrice.String(),
			AskPrice:  quote.AskPrice.String(),
			BidSize:   quote.BidSize.String(),
			AskSize:   quote.AskSize.String(),
			LastPrice: quote.LastPrice.String(),
			LastSize:  quote.LastSize.String(),
			Timestamp: quote.Timestamp,
			Source:    quote.Source,
		})
	}

	return dtos, nil
}
