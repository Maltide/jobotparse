package helpers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Maltide/JoBot/pkg/config"
	"github.com/Maltide/JoBot/pkg/consts"
	"github.com/Maltide/JoBot/pkg/types"
	"go.uber.org/zap"
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
		return false, nil // токены пустые — невалиден
	}

	if time.Now().Unix() >= int64(tokens.Ttl) {
		return false, nil // истёк
	}

	return true, nil // токен валиден
}

func VacancyFilters(log *zap.SugaredLogger) (types.Filters, error) {
	var filters types.Filters
	fields := []struct {
		object string
		input  *string
	}{
		{"profession:", &filters.Profession},
		{"town:", &filters.Town},
		{"salary from:", &filters.SalaryFrom},
		{"salary to:", &filters.SalaryTo},
		{"skills:", &filters.Skills},
	}

	r := bufio.NewReader(os.Stdin)

	for _, f := range fields {
		fmt.Println(f.object)
		text, err := r.ReadString('\n')
		if err != nil {
			log.Errorf("helpers: VacancyFilters: error reading vacancy filters: %v", err)
			return types.Filters{}, err
		}
		*f.input = strings.TrimSpace(text)
	}
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
