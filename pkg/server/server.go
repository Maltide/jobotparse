package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Maltide/jobotparse/pkg/handlers"
	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/middleware"
	"github.com/Maltide/jobotparse/pkg/ollama"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GetServer wires all HTTP routes and starts listening on :8080.
//
// It provides:
// - /auth: local login + redirect to SuperJob OAuth
// - /callback: exchange code for tokens
// - /vacancies: UI + fetch vacancies
func GetServer(log *zap.SugaredLogger, database *gorm.DB, apis []interfaces.VacanciesProvider, ollamaSession *ollama.ChatSession) error {
	router := http.NewServeMux()

	type vacanciesPageData struct {
		Links    []string
		Searched bool
		Error    string
	}

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
		// /callback is the OAuth redirect endpoint from SuperJob.
		err := middleware.GetAccessToken(w, r, log)
		if err != nil {
			log.Errorf("from server.go: error in get access token middleware: %v", err)
		}
	})

	router.HandleFunc("/vacancies", func(w http.ResponseWriter, r *http.Request) {
		// If it's a plain GET with no query parameters — serve the HTML form (no results)
		if r.Method == http.MethodGet && len(r.URL.Query()) == 0 {
			// render template with empty data (Searched=false)
			tmpl, terr := template.ParseFiles("static/vacancies.html")
			if terr != nil {
				log.Errorf("server: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			data := vacanciesPageData{Links: nil, Searched: false}
			tmpl.Execute(w, data)
			return
		}

		// Ensure tokens are valid (refresh if needed) before hitting external APIs.
		if err := middleware.BeforeRequest(log); err != nil {
			log.Infof("server: BeforeRequest failed, redirecting to auth")
			http.Redirect(w, r, "/auth", http.StatusFound)
			return
		}

		filters, err := helpers.VacancyFilters(r, log)
		if err != nil {
			log.Errorf("from server.go: error parsing vacancy filters: %v", err)
			tmpl, terr := template.ParseFiles("static/vacancies.html")
			if terr != nil {
				log.Errorf("server: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// Render the same page with an inline error message so the user doesn't land on a blank error page.
			data := vacanciesPageData{Links: nil, Searched: false, Error: "Неверные фильтры: профессия обязательна."}
			if err := tmpl.Execute(w, data); err != nil {
				log.Errorf("server: template execute error: %v", err)
			}
			return
		}

		vacs, err := handlers.AllVacancies(apis, filters, log)
		if err != nil {
			http.Error(w, "error fetching vacancies", http.StatusInternalServerError)
			log.Errorf("from server.go: error fetching vacancies from SuperJob: %v", err)
			return
		}

		// 0 вакансий — это нормальный результат поиска, а не ошибка БД.
		// Просто покажем страницу с пустым результатом.
		if len(vacs.Objects) == 0 {
			tmpl, terr := template.ParseFiles("static/vacancies.html")
			if terr != nil {
				log.Errorf("handlers: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			data := vacanciesPageData{Links: nil, Searched: true}
			if err := tmpl.Execute(w, data); err != nil {
				log.Errorf("handlers: template execute error: %v", err)
			}
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

		tmpl, terr := template.ParseFiles("static/vacancies.html")
		if terr != nil {
			log.Errorf("handlers: template parse error: %v", terr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := vacanciesPageData{Links: links, Searched: true}

		if err := tmpl.Execute(w, data); err != nil {
			log.Errorf("handlers: template execute error: %v", err)
			return
		}
	})

	router.HandleFunc("/adapt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl, terr := template.ParseFiles("static/adapt.html")
			if terr != nil {
				log.Errorf("server: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, nil)
			return
		}

		if r.Method == http.MethodPost {
			if err := handlers.AdaptResumeWithDeps(w, r, ollamaSession, log); err != nil {
				log.Errorf("server: error in AdaptResumeWithDeps: %v", err)
				return
			}
			return
		}
		log.Errorf("server: method not allowed for /adapt: %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	router.HandleFunc("/adapt/iterate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl, terr := template.ParseFiles("static/adapt.html")
			if terr != nil {
				log.Errorf("server: template parse error: %v", terr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, nil)
			return
		}

		if err := handlers.AdaptIterate(w, r, ollamaSession, log); err != nil {
			log.Errorf("server: error in AdaptIterate: %v", err)
			return
		}
	})

	log.Info("server starting on port 8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Errorf("fail to create server: %v", err)
		return err
	}

	return nil
}
