package superjob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

func FetchAPI(vacancyURL string, log *zap.SugaredLogger) (types.Vacancy, error) {
	vacancyURL = strings.TrimSpace(vacancyURL)
	if vacancyURL == "" {
		log.Errorf("helpers: FetchAPI: vacancy_url is empty")
		return types.Vacancy{}, fmt.Errorf("helpers: FetchAPI: vacancy_url is empty")
	}

	parsedURL, err := url.Parse(vacancyURL)
	if err != nil {
		log.Errorf("helpers: FetchAPI: error parsing vacancy_url: %v", err)
		return types.Vacancy{}, err
	}

	hostname := parsedURL.Host // "www.superjob.ru"
	log.Infof("helpers: FetchAPI: parsed hostname: %s", hostname)

	tmpl := parsedURL.Path // "/vacancies/razrab-go-12345.html"
	log.Infof("helpers: FetchAPI: parsed path: %s", tmpl)

	parts := strings.Split(tmpl, "-")
	log.Infof("helpers: FetchAPI: split path into parts: %v", parts)

	lastPart := parts[len(parts)-1]
	log.Infof("helpers: FetchAPI: last part of path (expected to contain ID): %s", lastPart)

	vacID := strings.Split(lastPart, ".html")[0] // "12345"
	log.Infof("helpers: FetchAPI: extracted vacancy ID: %s", vacID)

	code := "https://api.superjob.ru/2.0/vacancies/" + vacID + "/"

	req, _ := http.NewRequest("GET", code, nil)
	req.Header.Set("X-Api-App-Id", os.Getenv("CLIENT_SECRET"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("helpers: FetchAPI: fail to get-request to superjob: %v", err)
		return types.Vacancy{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("helpers: FetchAPI: fail to fetch vacancy, status code: %d", resp.StatusCode)
		return types.Vacancy{}, fmt.Errorf("fail to fetch vacancy, status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("helpers: FetchAPI: fail to read response body: %v", err)
		return types.Vacancy{}, err
	}

	vacancy := types.Vacancy{}
	err = json.Unmarshal(body, &vacancy)
	if err != nil {
		log.Errorf("helpers: FetchAPI: error unmarshalling vacancy data: %v", err)
		return types.Vacancy{}, err
	}

	if vacancy.Town != nil {
		vacancy.TownName = vacancy.Town.Title
	} else {
		vacancy.TownName = ""
	}
	if vacancy.TypeOfWork != nil {
		vacancy.TypeOfWorkTitle = vacancy.TypeOfWork.Title
	} else {
		vacancy.TypeOfWorkTitle = ""
	}
	if vacancy.Experience != nil {
		vacancy.ExperienceTitle = vacancy.Experience.Title
	} else {
		vacancy.ExperienceTitle = ""
	}

	log.Infof("helpers: FetchAPI: fetched vacancy ID: %d, Profession: %s", vacancy.ExternalID, vacancy.Profession)
	return vacancy, nil
}
