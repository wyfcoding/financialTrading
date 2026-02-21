package application

import (
	"log/slog"

	marginapp "github.com/wyfcoding/financialtrading/internal/margin/application"
	margindomain "github.com/wyfcoding/financialtrading/internal/margin/domain"
)

// NewService exposes margin lending capabilities through the marginlending bounded-context path.
func NewService(repo margindomain.MarginRepository, logger *slog.Logger) *marginapp.MarginAppService {
	return marginapp.NewMarginAppService(repo, logger)
}
