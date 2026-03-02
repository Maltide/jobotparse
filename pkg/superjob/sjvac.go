package superjob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

// NewSuperJobClient creates a SuperJob API client, clientSecret is used as X-Api-App-Id header for SuperJob requests.
func NewSuperJobClient(client *http.Client, clientSecret string) *SuperJob {
	if client == nil {
		client = http.DefaultClient
	}
	return &SuperJob{
		client:       client,
		clientSecret: clientSecret,
	}
}

// Fetch implements interfaces.VacanciesProvider by querying SuperJob vacancies API.
func (s SuperJob) Fetch(filters types.Filters, log *zap.SugaredLogger) (types.VacanciesResponse, error) {
	log.Infof("handlers: vacancies endpoint hit")

	origTown := strings.TrimSpace(filters.Town)
	origFrom, _ := strconv.Atoi(strings.TrimSpace(filters.SalaryFrom))
	origTo, _ := strconv.Atoi(strings.TrimSpace(filters.SalaryTo))

	applyLocalFilters := func(v types.VacanciesResponse) types.VacanciesResponse {
		if len(v.Objects) == 0 {
			return v
		}
		// Если пользователь не задал ни город, ни зарплату — фильтровать нечего.
		if origTown == "" && origFrom <= 0 && origTo <= 0 {
			return v
		}

		filtered := make([]types.Vacancy, 0, len(v.Objects))
		for i := range v.Objects {
			vac := v.Objects[i]

			// Фильтр по городу: API иногда не понимает town как строку, поэтому проверяем на нашей стороне.
			if origTown != "" {
				if vac.Town == nil {
					continue
				}
				title := strings.ToLower(strings.TrimSpace(vac.Town.Title))
				want := strings.ToLower(origTown)
				if title != want && !strings.Contains(title, want) && !strings.Contains(want, title) {
					continue
				}
			}

			// Фильтр по зарплате: проверяем пересечение диапазонов.
			if origFrom > 0 || origTo > 0 {
				min := vac.PaymentFrom
				max := vac.PaymentTo
				if min == 0 && max == 0 {
					continue
				}
				if max == 0 {
					max = min
				}
				if min == 0 {
					min = max
				}
				if origFrom > 0 && max < origFrom {
					continue
				}
				if origTo > 0 && min > origTo {
					continue
				}
			}

			filtered = append(filtered, vac)
		}

		v.Objects = filtered
		return v
	}

	fetchOnce := func(f types.Filters) (types.VacanciesResponse, error) {
		reqString, err := helpers.RequestString(f, log)
		if err != nil {
			return types.VacanciesResponse{}, err
		}

		req, _ := http.NewRequest("GET", reqString, nil)
		// SuperJob uses this header to identify the application.
		req.Header.Set("X-Api-App-Id", s.clientSecret)

		resp, err := s.client.Do(req)
		if err != nil {
			log.Errorf("handlers: error fetching vacancies: %v", err)
			return types.VacanciesResponse{}, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("handlers: error reading vacancies response body: %v", err)
			return types.VacanciesResponse{}, err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Не логируем большие тела целиком: ограничим размер.
			b := strings.TrimSpace(string(body))
			if len(b) > 800 {
				b = b[:800] + "..."
			}
			log.Errorf("handlers: SuperJob API bad status: %d, body: %s", resp.StatusCode, b)
			return types.VacanciesResponse{}, fmt.Errorf("superjob vacancies api status: %d", resp.StatusCode)
		}

		var vacancies types.VacanciesResponse
		if err := json.Unmarshal(body, &vacancies); err != nil {
			log.Errorf("handlers: error parsing vacancies response: %v", err)
			return types.VacanciesResponse{}, err
		}

		return vacancies, nil
	}

	vacancies, err := fetchOnce(filters)
	if err != nil {
		return types.VacanciesResponse{}, err
	}
	if len(vacancies.Objects) > 0 {
		return vacancies, nil
	}

	// Fallback 1: убрать salary-фильтры (API часто даёт 0, хотя на сайте есть результаты)
	if strings.TrimSpace(filters.SalaryFrom) != "" || strings.TrimSpace(filters.SalaryTo) != "" {
		retry := filters
		retry.SalaryFrom = ""
		retry.SalaryTo = ""
		v2, rerr := fetchOnce(retry)
		if rerr == nil {
			v2 = applyLocalFilters(v2)
			if len(v2.Objects) > 0 {
				return v2, nil
			}
		}
	}

	// Fallback 2: убрать town-фильтр (API может не понимать town как строку)
	if origTown != "" {
		retry := filters
		retry.Town = ""
		v3, rerr := fetchOnce(retry)
		if rerr == nil {
			v3 = applyLocalFilters(v3)
			if len(v3.Objects) > 0 {
				return v3, nil
			}
		}
	}

	// Fallback 3: убрать и salary и town сразу
	if origTown != "" && (strings.TrimSpace(filters.SalaryFrom) != "" || strings.TrimSpace(filters.SalaryTo) != "") {
		retry := filters
		retry.Town = ""
		retry.SalaryFrom = ""
		retry.SalaryTo = ""
		v4, rerr := fetchOnce(retry)
		if rerr == nil {
			v4 = applyLocalFilters(v4)
			if len(v4.Objects) > 0 {
				return v4, nil
			}
		}
	}

	// Ничего не нашли: вернём пустой список без ошибки.
	return vacancies, nil
}
