package superjob

import "net/http"

type SuperJob struct {
	client       *http.Client
	clientSecret string
}
