package superjob

import (
	"encoding/json"
	"io"
	"net/http"

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

	reqString, err := helpers.RequestString(filters, log)
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

	// log.Infof("handlers: vacancies response body: %s", string(body))

	var vacancies types.VacanciesResponse

	err = json.Unmarshal(body, &vacancies)
	if err != nil {
		log.Errorf("handlers: error parsing vacancies response: %v", err)
		return types.VacanciesResponse{}, err
	}

	return vacancies, nil
}
