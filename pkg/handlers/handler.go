package handlers

import (
	"net/http"
	"os"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

// Authorize serves the /auth endpoint.
// GET returns an HTML login form; POST validates admin and redirects the client to SuperJob OAuth authorization.
func Authorize(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./static/auth.html")
		return nil
	}

	if r.Method == http.MethodPost {
		// Validate admin fields from env before redirecting to SuperJob OAuth.
		if err := r.ParseForm(); err != nil {
			log.Errorf("handlers: error parsing form: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return err
		}
		user := r.PostFormValue("username")
		pass := r.PostFormValue("password")

		if user == "" || pass == "" {
			log.Errorf("handlers: username or password is empty")
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return nil
		}
		if user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			log.Errorf("handlers: invalid username or password")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return nil
		}
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	// Not BeforeRequest func because it uses RefreshFunc like if tokens already expired 100%, but we need to check tokens are still valid
	tokensinfo, rerr := helpers.ReadTokens(log)
	if rerr == nil {
		ok, ierr := helpers.IsValidToken(&tokensinfo, log)
		if ierr == nil && ok {
			// If a valid token already exists, avoid forcing OAuth.
			log.Infof("handlers: tokens are present, no need to authorize")
			return nil
		}
	}

	url, err := helpers.Authstr()
	if err != nil {
		log.Errorf("handlers: error getting auth URL: %v", err)
		return err
	}

	http.Redirect(w, r, url, http.StatusFound)

	return nil
}

// AllVacancies fetches vacancies from all configured providers and merges results.
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
