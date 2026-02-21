package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AccountStatement represents paged account entries within a time window.
type AccountStatement struct {
	AccountID   string          `json:"account_id"`
	Currency    string          `json:"currency"`
	StartDate   time.Time       `json:"start_date"`
	EndDate     time.Time       `json:"end_date"`
	Entries     []LedgerEntry   `json:"entries"`
	TotalDebit  decimal.Decimal `json:"total_debit"`
	TotalCredit decimal.Decimal `json:"total_credit"`
	Page        int             `json:"page"`
	PageSize    int             `json:"page_size"`
	Total       int64           `json:"total"`
}

type TrialBalanceItem struct {
	AccountID   string          `json:"account_id"`
	AccountNo   string          `json:"account_no"`
	AccountName string          `json:"account_name"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
}

type TrialBalance struct {
	AsOf        time.Time          `json:"as_of"`
	Items       []TrialBalanceItem `json:"items"`
	TotalDebit  decimal.Decimal    `json:"total_debit"`
	TotalCredit decimal.Decimal    `json:"total_credit"`
	Balanced    bool               `json:"balanced"`
}
