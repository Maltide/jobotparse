package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Maltide/jobotparse/pkg/handlers"
	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetServer(log *zap.SugaredLogger, database *gorm.DB, apis []interfaces.VacanciesProvider) error {
	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Сервер запущен")
	})

	router.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		err := handlers.Authorize(w, r, log)
		if err != nil {
			log.Errorf("from server.go: error in authorize handler: %v", err)
		}
	})

	router.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		err := middleware.GetAccessToken(w, r, log)
		if err != nil {
			log.Errorf("from server.go: error in get access token middleware: %v", err)
		}
	})

	router.HandleFunc("/vacancies", func(w http.ResponseWriter, r *http.Request) {
		// If it's a plain GET with no query parameters — serve the HTML form (no results)
		if r.Method == http.MethodGet && len(r.URL.Query()) == 0 {
			// render template with empty data (Searched=false)
			tmpl, terr := template.ParseFiles("vacancies.html")
			if terr != nil {
				log.Errorf("server: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			data := struct {
				Links    []string
				Searched bool
			}{Links: nil, Searched: false}
			tmpl.Execute(w, data)
			return
		}

		// Run middleware to ensure tokens are valid / refreshed
		if err := middleware.BeforeRequest(log); err != nil {
			log.Infof("server: BeforeRequest failed, redirecting to auth")
			http.Redirect(w, r, "/auth", http.StatusFound)
			return
		}

		filters, err := helpers.VacancyFilters(r, log)
		if err != nil {
			log.Errorf("from server.go: error parsing vacancy filters: %v", err)
			http.Error(w, "Invalid filters", http.StatusBadRequest)
			return
		}

		vacs, err := handlers.AllVacancies(apis, filters, log)
		if err != nil {
			http.Error(w, "error fetching vacancies", http.StatusInternalServerError)
			log.Errorf("from server.go: error fetching vacancies from SuperJob: %v", err)
			return
		}

		err = helpers.DBWrite(database, vacs, log)
		if err != nil {
			http.Error(w, "error writing vacancies to DB", http.StatusInternalServerError)
			log.Errorf("from server.go: error writing vacancies to DB: %v", err)
			return
		}

		// collect links and render template with results
		links := []string{}
		for i := range vacs.Objects {
			v := &vacs.Objects[i]
			if v.Link != "" {
				links = append(links, v.Link)
			}
		}

		tmpl, terr := template.ParseFiles("vacancies.html")
		if terr != nil {
			log.Errorf("handlers: template parse error: %v", terr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := struct {
			Links    []string
			Searched bool
		}{Links: links, Searched: true}

		if err := tmpl.Execute(w, data); err != nil {
			log.Errorf("handlers: template execute error: %v", err)
			return
		}
	})

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Errorf("fail to create server: %v", err)
		return err
	}

	log.Info("server started on port 8080")

	return nil
}
