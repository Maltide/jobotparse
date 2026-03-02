package helpers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

// VacancyFilters parses filters from an HTTP request's query parameters.
// This is suitable for HTTP handlers; for local stdin use a separate helper.
func VacancyFilters(r *http.Request, log *zap.SugaredLogger) (types.Filters, error) {
	var filters types.Filters
	if r == nil {
		log.Infof("helpers: VacancyFilters: nil request, returning empty filters")
		return filters, fmt.Errorf("helpers: VacancyFilters: nil request")
	}

	q := r.URL.Query()

	filters.Profession = strings.TrimSpace(q.Get("profession"))
	if filters.Profession == "" {
		return filters, fmt.Errorf("helpers: VacancyFilters: profession parameter is required")
	}

	filters.Town = strings.TrimSpace(q.Get("town"))

	filters.SalaryFrom = strings.TrimSpace(q.Get("salary_from"))

	filters.SalaryTo = strings.TrimSpace(q.Get("salary_to"))

	filters.Skills = strings.TrimSpace(q.Get("skills"))

	return filters, nil
}

// RequestString builds a SuperJob vacancies API URL using the provided filters.
func RequestString(filters types.Filters, log *zap.SugaredLogger) (string, error) {
	base := "https://api.superjob.ru/2.0/vacancies/?"

	u, err := url.Parse(base)
	if err != nil {
		log.Errorf("helpers: RequestString: error parsing base URL: %v", err)
		return "", err
	}

	q := u.Query()

	// Просим больше результатов за один запрос: это помогает при fallback-фильтрации на нашей стороне.
	// Если API проигнорирует параметр — ничего страшного.
	q.Set("count", "100")

	for _, f := range []struct {
		key   string
		value string
	}{
		{"keyword", filters.Profession},
		{"town", filters.Town},
		{"payment_from", filters.SalaryFrom},
		{"payment_to", filters.SalaryTo},
		{"skills", filters.Skills},
	} {
		if f.value != "" {
			q.Set(f.key, f.value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
