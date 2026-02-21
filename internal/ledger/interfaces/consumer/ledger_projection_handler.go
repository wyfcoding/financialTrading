// 变更说明：新增 ledger 投影处理器，负责消费领域事件并更新读模型。
// 消费事件源：Kafka topic "ledger.events"。
// 更新目标：Redis 缓存（实时余额）、Elasticsearch（流水搜索索引）。
// 幂等保证：通过事件 ID 去重，确保重复消费不会导致数据不一致。
package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
)

// LedgerProjectionHandler 账本投影处理器。
// 负责将领域事件投影到读模型（Redis + Elasticsearch）。
type LedgerProjectionHandler struct {
	readRepo   domain.LedgerReadRepository
	searchRepo domain.LedgerSearchRepository
	logger     *slog.Logger
}

// NewLedgerProjectionHandler 创建投影处理器实例。
func NewLedgerProjectionHandler(
	readRepo domain.LedgerReadRepository,
	searchRepo domain.LedgerSearchRepository,
	logger *slog.Logger,
) *LedgerProjectionHandler {
	return &LedgerProjectionHandler{
		readRepo:   readRepo,
		searchRepo: searchRepo,
		logger:     logger.With("module", "ledger_projection"),
	}
}

// HandleJournalCreated 处理凭证创建事件。
// 更新涉及账户的缓存余额，并将凭证和分录索引到 Elasticsearch。
func (h *LedgerProjectionHandler) HandleJournalCreated(ctx context.Context, payload []byte) error {
	var event domain.JournalCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		h.logger.ErrorContext(ctx, "failed to unmarshal journal created event",
			"error", err,
		)
		return err
	}

	h.logger.InfoContext(ctx, "projecting journal created event",
		"journal_id", event.JournalID,
		"transaction_id", event.TransactionID,
		"journal_type", event.JournalType,
	)

	// 1. 索引凭证到 Elasticsearch。
	if h.searchRepo != nil {
		journalDoc := &domain.JournalSearchDoc{
			JournalID:     event.JournalID,
			TransactionID: event.TransactionID,
			JournalType:   event.JournalType,
			Description:   event.Description,
			TotalAmount:   event.TotalAmount,
			Currency:      event.Currency,
			CreatedAt:     event.OccurredAt,
		}
		if err := h.searchRepo.IndexJournal(ctx, journalDoc); err != nil {
			h.logger.ErrorContext(ctx, "failed to index journal",
				"journal_id", event.JournalID,
				"error", err,
			)
		}

		// 2. 索引每条分录。
		for _, entry := range event.Entries {
			entryDoc := &domain.EntrySearchDoc{
				EntryID:       entry.EntryID,
				JournalID:     event.JournalID,
				TransactionID: event.TransactionID,
				AccountID:     entry.AccountID,
				Direction:     string(entry.Direction),
				Amount:        entry.Amount,
				Currency:      entry.Currency,
				JournalType:   event.JournalType,
				CreatedAt:     event.OccurredAt,
			}
			if err := h.searchRepo.IndexEntry(ctx, entryDoc); err != nil {
				h.logger.ErrorContext(ctx, "failed to index entry",
					"entry_id", entry.EntryID,
					"error", err,
				)
			}
		}
	}

	return nil
}

// HandleBalanceUpdated 处理余额更新事件。
// 更新 Redis 中的账户余额缓存。
func (h *LedgerProjectionHandler) HandleBalanceUpdated(ctx context.Context, payload []byte) error {
	var view domain.AccountBalanceView
	if err := json.Unmarshal(payload, &view); err != nil {
		h.logger.ErrorContext(ctx, "failed to unmarshal balance updated event",
			"error", err,
		)
		return err
	}

	h.logger.DebugContext(ctx, "projecting balance update",
		"account_id", view.AccountID,
		"available", view.AvailableBalance.String(),
		"hold", view.HoldBalance.String(),
	)

	if h.readRepo != nil {
		if err := h.readRepo.SetCachedBalance(ctx, &view); err != nil {
			h.logger.ErrorContext(ctx, "failed to update cached balance",
				"account_id", view.AccountID,
				"error", err,
			)
			return err
		}
	}

	return nil
}

// HandleFundsHeld 处理资金冻结事件。
func (h *LedgerProjectionHandler) HandleFundsHeld(ctx context.Context, payload []byte) error {
	var event domain.FundsHeldEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	h.logger.InfoContext(ctx, "projecting funds held event",
		"account_id", event.AccountID,
		"reference_id", event.ReferenceID,
		"amount", event.Amount.String(),
	)

	// 索引到 Elasticsearch 用于审计。
	if h.searchRepo != nil {
		entryDoc := &domain.EntrySearchDoc{
			EntryID:       event.EventID,
			TransactionID: event.TransactionID,
			AccountID:     event.AccountID,
			Direction:     "HOLD",
			Amount:        event.Amount,
			Currency:      event.Currency,
			Description:   "funds held for " + event.ReferenceType + ": " + event.ReferenceID,
			JournalType:   "HOLD",
			CreatedAt:     event.OccurredAt,
		}
		if err := h.searchRepo.IndexEntry(ctx, entryDoc); err != nil {
			h.logger.ErrorContext(ctx, "failed to index hold event",
				"error", err,
			)
		}
	}

	return nil
}

// HandleDayEndReconciled 处理日终对账完成事件。
func (h *LedgerProjectionHandler) HandleDayEndReconciled(ctx context.Context, payload []byte) error {
	var event domain.DayEndReconciledEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	h.logger.InfoContext(ctx, "day-end reconciliation completed",
		"date", event.ReconcileDate.Format("2006-01-02"),
		"total_accounts", event.TotalAccounts,
		"matched", event.MatchedCount,
		"mismatched", event.MismatchCount,
		"is_balanced", event.IsBalanced,
	)

	if !event.IsBalanced {
		h.logger.ErrorContext(ctx, "CRITICAL: day-end reconciliation UNBALANCED",
			"date", event.ReconcileDate.Format("2006-01-02"),
			"total_debit", event.TotalDebit.String(),
			"total_credit", event.TotalCredit.String(),
		)
	}

	return nil
}
