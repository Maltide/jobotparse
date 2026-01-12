package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Maltide/JoBot/pkg/config"
	"github.com/Maltide/JoBot/pkg/helpers"
	"github.com/Maltide/JoBot/pkg/types"
	"go.uber.org/zap"
)

func Authorize(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	var tokensinfo types.Client

	ok, err := helpers.IsValidToken(&tokensinfo, log)
	if err != nil {
		return err
	}
	if !ok {
		log.Infof("handlers: tokens are present, no need to authorize")
		return nil
	}

	url, err := helpers.Authstr()
	if err != nil {
		log.Errorf("handlers: error getting auth URL: %v", err)
		return err
	}

	http.Redirect(w, r, url, http.StatusFound)

	return nil
}

func GetVacancies(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	log.Infof("handlers: vacancies endpoint hit")

	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("handlers: error getting config: %v", err)
		return err
	}

	var tokens types.Client

	ok, err := helpers.IsValidToken(&tokens, log)
	if err != nil {
		return err
	}
	if !ok {
		log.Infof("handlers: invalid token, redirecting to authorize")
		http.Redirect(w, r, "/authorize", http.StatusFound)
		return nil
	}

	filters, err := helpers.VacancyFilters(log)
	if err != nil {
		return err
	}

	reqString, err := helpers.RequestString(filters, log)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("GET", reqString, nil)
	req.Header.Set("X-Api-App-Id", cfg.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("handlers: error fetching vacancies: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("handlers: error reading vacancies response body: %v", err)
		return err
	}

	// log.Infof("handlers: vacancies response body: %s", string(body))

	var vacancies types.VacanciesResponse

	err = json.Unmarshal(body, &vacancies)
	if err != nil {
		log.Errorf("handlers: error parsing vacancies response: %v", err)
		return err
	}

	for _, vacancy := range vacancies.Objects {
		log.Infof("handlers: vacancy ID: %d, Profession: %s", vacancy.ID, vacancy.Profession)
		fmt.Printf("vacancy link: %v\n", vacancy.Link)
	}
	return nil
}
