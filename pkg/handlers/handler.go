package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/ollama"
	"github.com/Maltide/jobotparse/pkg/superjob"
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
		log.Errorf("handlers: method not allowed: %s", r.Method)
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

// AdaptResumeWithDeps handles POST /adapt and starts the chat session by
// building a prompt from vacancy + resume fields and sending it to Ollama.
func AdaptResumeWithDeps(w http.ResponseWriter, r *http.Request, ollamasession *ollama.ChatSession, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		log.Errorf("handlers: adaptresumewithdeps: not POST method: %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("handlers: error reading body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return err
	}

	var in types.Resume

	if err := json.Unmarshal(body, &in); err != nil {
		log.Errorf("handlers: error unmarshalling JSON body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return err
	}

	vacancyURL := strings.TrimSpace(in.VacancyURL)
	if vacancyURL == "" {
		log.Errorf("handlers: vacancy_url is missing in payload")
		http.Error(w, "vacancy_url is required", http.StatusBadRequest)
		return nil
	}

	vac, err := superjob.FetchAPI(vacancyURL, log)
	if err != nil {
		log.Errorf("handlers: FetchAPI failed: %v", err)
		http.Error(w, "Failed to fetch vacancy", http.StatusBadRequest)
		return err
	}

	resumeText := helpers.BuildResumeText(in)

	aireq, err := os.ReadFile("./static/aireq.txt")
	if err != nil {
		log.Errorf("handlers: error reading aireq.txt: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}

	parts := []string{string(aireq)}

	parts = append(parts, "Вакансия:\n"+helpers.VacancyToText(vac))

	if resumeText != "" {
		parts = append(parts, "Резюме:\n"+resumeText)
	}

	userContent := strings.TrimSpace(strings.Join(parts, "\n\n"))

	ollamasession.Messages = append(ollamasession.Messages, ollama.Message{Role: "system", Content: userContent})

	assistantMsg, err := ollama.OllamaRequest(ollamasession, log)
	if err != nil {
		log.Errorf("handlers: OllamaRequest failed: %v", err)
		http.Error(w, "Model request failed", http.StatusInternalServerError)
		return err
	}
	ollamasession.Messages = append(ollamasession.Messages, assistantMsg)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"assistant": assistantMsg.Content})
}

// AdaptIterate handles POST /adapt/iterate (application/json): {"instruction":"..."}
func AdaptIterate(w http.ResponseWriter, r *http.Request, ollamaSession *ollama.ChatSession, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		log.Errorf("handlers: adaptiterate: not POST method: %s", r.Method)
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

func Final(w http.ResponseWriter, r *http.Request, ollamaSession *ollama.ChatSession, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		log.Errorf("handlers: final: not POST method: %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	if len(ollamaSession.Messages) == 0 {
		http.Error(w, "no messages in session", http.StatusBadRequest)
		return nil
	}

	// Fallback: return last assistant content
	content := ollamaSession.Messages[len(ollamaSession.Messages)-1].Content
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"assistant": content})
}
