package interfaces

import (
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

type VacanciesProvider interface {
	// Fetch returns vacancies for provided filters.
	Fetch(filters types.Filters, log *zap.SugaredLogger) (types.VacanciesResponse, error)
}
