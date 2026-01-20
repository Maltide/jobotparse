package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/consts"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Authstr() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	return "https://www.superjob.ru/authorize/?client_id=" + cfg.ClientID + "&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&state=custom", nil
}

func IsValidToken(tokens *types.Client, log *zap.SugaredLogger) (bool, error) {
	body, err := os.ReadFile(consts.TokensFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // файла нет — токен невалиден
		}
		log.Errorf("helpers: error reading tokens file: %v", err)
		return false, err
	}

	err = json.Unmarshal(body, tokens)
	if err != nil {
		log.Errorf("helpers: IsValidToken: error unmarshalling tokens: %v", err)
		return false, err
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		log.Infof("helpers: tokens are empty")
		return false, nil // токены пустые — невалиден
	}

	if time.Now().Unix() >= int64(tokens.Ttl) {
		log.Infof("helpers: token has expired")
		return false, nil // истёк
	}

	return true, nil // токен валиден
}

// VacancyFilters parses filters from an HTTP request's query parameters.
// This is suitable for HTTP handlers; for local stdin use a separate helper.
func VacancyFilters(r *http.Request, log *zap.SugaredLogger) (types.Filters, error) {
	var filters types.Filters
	if r == nil {
		return filters, nil
	}
	q := r.URL.Query()
	filters.Profession = strings.TrimSpace(q.Get("profession"))
	filters.Town = strings.TrimSpace(q.Get("town"))
	// accept both snake_case and space-separated names from query
	if filters.SalaryFrom == "" {
		filters.SalaryFrom = strings.TrimSpace(q.Get("salary_from"))
	}
	if filters.SalaryTo == "" {
		filters.SalaryTo = strings.TrimSpace(q.Get("salary_to"))
	}
	filters.Skills = strings.TrimSpace(q.Get("skills"))
	return filters, nil
}

func RequestString(filters types.Filters, log *zap.SugaredLogger) (string, error) {
	base := "https://api.superjob.ru/2.0/vacancies/?"

	u, err := url.Parse(base)
	if err != nil {
		log.Errorf("helpers: RequestString: error parsing base URL: %v", err)
		return "", err
	}

	q := u.Query()

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

func DBWrite(db *gorm.DB, vacancies types.VacanciesResponse, log *zap.SugaredLogger) error {
	for i := range vacancies.Objects {
		v := &vacancies.Objects[i]
		if v.Town != nil && v.Town.Title != "" {
			v.TownName = v.Town.Title
		}

		fmt.Printf("vacancy link: %v\n", v.Link)

		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "external_id"}},
			DoNothing: true,
		}).Create(v).Error
		if err != nil {
			log.Errorf("handlers: error saving vacancy to database: %v", err)
			return err
		}

		log.Infof("handlers: vacancy ID: %d, Profession: %s", v.ID, v.Profession)
	}
	return nil
}
