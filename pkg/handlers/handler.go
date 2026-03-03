package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/ollama"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

func AdaptResume(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	return fmt.Errorf("handlers: AdaptResume is not wired; use AdaptResumeWithDeps")
}

func vacancyToText(v types.Vacancy) string {
	var b strings.Builder
	if v.Profession != "" {
		b.WriteString("Позиция: ")
		b.WriteString(v.Profession)
		b.WriteString("\n")
	}
	if v.FirmName != "" {
		b.WriteString("Компания: ")
		b.WriteString(v.FirmName)
		b.WriteString("\n")
	}
	if v.TownName != "" {
		b.WriteString("Город: ")
		b.WriteString(v.TownName)
		b.WriteString("\n")
	}
	if v.TypeOfWorkTitle != "" {
		b.WriteString("Тип занятости: ")
		b.WriteString(v.TypeOfWorkTitle)
		b.WriteString("\n")
	}
	if v.ExperienceTitle != "" {
		b.WriteString("Опыт: ")
		b.WriteString(v.ExperienceTitle)
		b.WriteString("\n")
	}
	if v.PaymentFrom != 0 || v.PaymentTo != 0 {
		b.WriteString("Зарплата: ")
		if v.PaymentFrom != 0 {
			b.WriteString(fmt.Sprintf("от %d ", v.PaymentFrom))
		}
		if v.PaymentTo != 0 {
			b.WriteString(fmt.Sprintf("до %d ", v.PaymentTo))
		}
		if v.Currency != "" {
			b.WriteString(v.Currency)
		}
		b.WriteString("\n")
	}
	if v.Work != "" {
		b.WriteString("Обязанности: \n")
		b.WriteString(strings.TrimSpace(v.Work))
		b.WriteString("\n")
	}
	if v.Compensation != "" {
		b.WriteString("Условия работы: \n")
		b.WriteString(strings.TrimSpace(v.Compensation))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// AdaptResumeWithDeps handles POST /adapt (multipart/form-data):
// - vacancy_url: link to SuperJob vacancy
// - resume_pdf: PDF file with the resume
func AdaptResumeWithDeps(w http.ResponseWriter, r *http.Request, db *gorm.DB, ollamasession *ollama.ChatSession, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	// 1) Parse multipart form to access both fields and file.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Errorf("handlers: error parsing multipart form: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return err
	}

	// 2) Read the vacancy link (expected field name in HTML: vacancy_url).
	vacancyURL := strings.TrimSpace(r.FormValue("vacancy_url"))
	if vacancyURL == "" {
		log.Errorf("handlers: vacancy_url is missing")
		http.Error(w, "vacancy_url is required", http.StatusBadRequest)
		return nil
	}

	// 3) Read the resume PDF file (expected field name in HTML: resume_pdf).
	resume, _, err := r.FormFile("resume_pdf")
	if err != nil {
		log.Errorf("handlers: error receiving resume PDF: %v", err)
		http.Error(w, "resume_pdf is required", http.StatusBadRequest)
		return nil
	}
	defer resume.Close()

	// 4) Convert PDF -> plain text.
	resumeText, err := helpers.PDFToText(resume, log)
	if err != nil {
		log.Errorf("handlers: PDFToText failed: %v", err)
		http.Error(w, "Failed to read PDF text (is pdftotext installed?)", http.StatusBadRequest)
		return err
	}

	// 5) Fetch vacancy data from DB; if not found, fetch from SuperJob API and store.
	vac, err := helpers.CheckVacInDB(r, db, log)
	if err != nil {
		log.Errorf("handlers: CheckVacInDB failed: %v", err)
		http.Error(w, "Failed to fetch vacancy", http.StatusBadRequest)
		return err
	}

	// 6) Build the prompt for the model.
	// The explicit "\n\n" separators are just for readability and to clearly separate sections.
	userContent := strings.TrimSpace(strings.Join([]string{
		"Адаптируй моё резюме под вакансию.",
		"Вакансия:\n" + vacancyToText(vac),
		"Резюме (текст из PDF):\n" + strings.TrimSpace(resumeText),
	}, "\n\n"))

	// 7) Add the user's message to the in-memory chat history.
	ollamasession.Messages = append(ollamasession.Messages, ollama.Message{Role: "user", Content: userContent})

	// 8) Call Ollama with the full history (system + all user/assistant turns).
	assistantMsg, err := ollama.OllamaRequest(ollamasession, log)
	if err != nil {
		log.Errorf("handlers: OllamaRequest failed: %v", err)
		http.Error(w, "Model request failed", http.StatusInternalServerError)
		return err
	}
	// 9) Store assistant reply in history so the next iteration has context.
	ollamasession.Messages = append(ollamasession.Messages, assistantMsg)

	// 10) Return only the assistant's text to the frontend chat UI.
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"assistant": assistantMsg.Content})
}

// AdaptIterate handles POST /adapt/iterate (application/json): {"instruction":"..."}
func AdaptIterate(w http.ResponseWriter, r *http.Request, ollamaSession *ollama.ChatSession, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	// 1) Parse user's instruction from JSON body.
	var in struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return err
	}
	instruction := strings.TrimSpace(in.Instruction)
	if instruction == "" {
		http.Error(w, "instruction is required", http.StatusBadRequest)
		return nil
	}

	// 2) Append as a user turn into the same in-memory chat history.
	ollamaSession.Messages = append(ollamaSession.Messages, ollama.Message{Role: "user", Content: instruction})
	// 3) Ask the model again with the full history.
	assistantMsg, err := ollama.OllamaRequest(ollamaSession, log)
	if err != nil {
		log.Errorf("handlers: OllamaRequest failed: %v", err)
		http.Error(w, "Model request failed", http.StatusInternalServerError)
		return err
	}
	// 4) Save assistant reply for the next turn.
	ollamaSession.Messages = append(ollamaSession.Messages, assistantMsg)

	// 5) Return reply as a simple JSON object for the frontend.
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"assistant": assistantMsg.Content})
}
