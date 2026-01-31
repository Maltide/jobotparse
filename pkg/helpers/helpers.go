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

// Authstr returns the SuperJob OAuth authorization URL.
func Authstr() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	return "https://www.superjob.ru/authorize/?client_id=" + cfg.ClientID + "&redirect_uri=" + cfg.BaseURL + "%2Fcallback&state=custom", nil
}

// ReadTokens reads tokens from tokens.json and decodes them into types.Client.
func ReadTokens(log *zap.SugaredLogger) (types.Client, error) {
	var tokens types.Client

	body, err := os.ReadFile(consts.TokensFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Error("helpers: tokens file does not exist")
			return tokens, err // файла нет — токен невалиден
		}
		log.Errorf("helpers: error reading tokens file: %v", err)
		return tokens, err
	}

	err = json.Unmarshal(body, &tokens)
	if err != nil {
		log.Errorf("helpers: IsValidToken: error unmarshalling tokens: %v", err)
		return tokens, err
	}
	return tokens, nil
}

// IsValidToken checks that access and refresh tokens are present and not expired.
func IsValidToken(tokens *types.Client, log *zap.SugaredLogger) (bool, error) {
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		log.Infof("helpers: tokens are empty")
		return false, fmt.Errorf("tokens are empty") // токены пустые — невалиден
	}

	if time.Now().Unix() >= int64(tokens.Ttl) {
		log.Infof("helpers: token has expired")
		return false, fmt.Errorf("token has expired") // истёк
	}

	return true, nil // токен валиден
}

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

// DBWrite upserts vacancies into Postgres.
// Ignores duplicates if external_id already exists.
func DBWrite(db *gorm.DB, vacancies types.VacanciesResponse, log *zap.SugaredLogger) error {
	if len(vacancies.Objects) == 0 {
		log.Errorf("helpers: DBWrite: no vacancies to write to DB")
		return fmt.Errorf("no vacancies to write to DB")
	}

	for i := range vacancies.Objects {
		v := &vacancies.Objects[i]
		if v.Town != nil {
			v.TownName = v.Town.Title
		} else {
			v.TownName = ""
		}

		fmt.Printf("vacancy link: %v\n", v.Link)

		// OnConflict with DoNothing checks that external_id is unique.
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
