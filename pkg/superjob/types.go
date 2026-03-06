package superjob

import "net/http"

// SuperJob is a client for the SuperJob API. It implements interfaces.VacanciesProvider.
type SuperJob struct {
	client       *http.Client
	clientSecret string
}
