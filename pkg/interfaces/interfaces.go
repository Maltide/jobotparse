package interfaces

import (
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

type VacanciesProvider interface {
	Fetch(filters types.Filters, log *zap.SugaredLogger) (types.VacanciesResponse, error)
}
