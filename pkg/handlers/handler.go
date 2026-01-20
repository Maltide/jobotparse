package handlers

import (
	"net/http"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

func Authorize(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	var tokensinfo types.Client

	ok, err := helpers.IsValidToken(&tokensinfo, log)
	if err != nil {
		return err
	}
	if ok {
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

func AllVacancies(apis []interfaces.VacanciesProvider, filters types.Filters, log *zap.SugaredLogger) (types.VacanciesResponse, error) {
	var allVacs types.VacanciesResponse

	for _, api := range apis {
		vacs, err := api.Fetch(filters, log)
		if err != nil {
			log.Errorf("handlers: error fetching vacancies from API: %v", err)
			continue
		}
		allVacs.Objects = append(allVacs.Objects, vacs.Objects...)
	}

	log.Infof("handlers: total vacancies fetched from all APIs: %d", len(allVacs.Objects))

	return allVacs, nil
}
