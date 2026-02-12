package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/consts"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResumeKeywordGroups struct {
	Confirmed []string
	Learning  []string
}

// ExtractResumeKeywordGroups returns two conservative lists of keywords found in the resume text:
// - Confirmed: mentioned without nearby "learning" markers
// - Learning: mentioned near phrases like "готов изучить", "в процессе"
// The goal is to keep the LLM grounded and avoid overstating skills.
func ExtractResumeKeywordGroups(resumeText string) ResumeKeywordGroups {
	text := strings.ToLower(resumeText)
	if strings.TrimSpace(text) == "" {
		return ResumeKeywordGroups{}
	}

	type kw struct {
		label    string
		patterns []string
	}

	known := []kw{
		{label: "Go", patterns: []string{" golang ", " go ", "go/", "go-", "go ", "(go", "go)"}},
		{label: "net/http", patterns: []string{"net/http"}},
		{label: "context", patterns: []string{" context", "context "}},
		{label: "JSON", patterns: []string{" json"}},
		{label: "REST API", patterns: []string{"rest", " rest "}},
		{label: "OpenAPI/Swagger", patterns: []string{"openapi", "swagger"}},
		{label: "PostgreSQL", patterns: []string{"postgres", "postgresql"}},
		{label: "SQL", patterns: []string{" sql"}},
		{label: "GORM", patterns: []string{"gorm"}},
		{label: "Docker", patterns: []string{"docker"}},
		{label: "Docker Compose", patterns: []string{"docker compose", "docker-compose", "compose"}},
		{label: "Linux", patterns: []string{" linux"}},
		{label: "nginx", patterns: []string{"nginx"}},
		{label: "Git", patterns: []string{" git"}},
		{label: "Makefile", patterns: []string{"makefile"}},
		{label: "CI/CD", patterns: []string{"ci/cd", " ci ", " cd ", "continuous integration", "continuous delivery"}},
		{label: "gRPC", patterns: []string{"grpc"}},
		{label: "Kafka", patterns: []string{"kafka"}},
		{label: "RabbitMQ", patterns: []string{"rabbitmq"}},
		{label: "Redis", patterns: []string{"redis"}},
		{label: "Testing", patterns: []string{"testing", "unit test", "unit-test", "unit testing", "тест"}},
	}

	learningMarkers := []string{
		"готов", "готовность", "готова", "готов изуч", "готов( ",
		"изуч", "в процессе", "планир", "осваива", "learn", "learning",
	}
	// PDF-to-text often introduces line breaks; keep a wider window.
	window := 220

	confirmedSeen := make(map[string]bool, len(known))
	learningSeen := make(map[string]bool, len(known))
	confirmed := make([]string, 0, 12)
	learning := make([]string, 0, 12)

	containsLearningMarker := func(s string) bool {
		for _, m := range learningMarkers {
			if strings.Contains(s, m) {
				return true
			}
		}
		return false
	}

	for _, k := range known {
		foundAny := false
		foundLearningOnly := false

		for _, p := range k.patterns {
			idx := strings.Index(text, p)
			for idx != -1 {
				foundAny = true
				start := idx - window
				if start < 0 {
					start = 0
				}
				end := idx + len(p) + window
				if end > len(text) {
					end = len(text)
				}
				ctx := text[start:end]
				if containsLearningMarker(ctx) {
					foundLearningOnly = true
				} else {
					// at least one occurrence not marked as learning => treat as confirmed
					foundLearningOnly = false
					break
				}
				next := idx + len(p)
				if next >= len(text) {
					break
				}
				j := strings.Index(text[next:], p)
				if j == -1 {
					break
				}
				idx = next + j
			}
			if foundAny && !foundLearningOnly {
				break
			}
		}

		if !foundAny {
			continue
		}
		if foundLearningOnly {
			if !learningSeen[k.label] {
				learningSeen[k.label] = true
				learning = append(learning, k.label)
			}
			continue
		}
		if !confirmedSeen[k.label] {
			confirmedSeen[k.label] = true
			confirmed = append(confirmed, k.label)
		}
	}

	return ResumeKeywordGroups{Confirmed: confirmed, Learning: learning}
}

// ExtractResumeKeywords is kept for backward compatibility; prefer ExtractResumeKeywordGroups.
func ExtractResumeKeywords(resumeText string) []string {
	g := ExtractResumeKeywordGroups(resumeText)
	out := make([]string, 0, len(g.Confirmed)+len(g.Learning))
	out = append(out, g.Confirmed...)
	out = append(out, g.Learning...)
	return out
}

// ExtractResumeSnippets pulls a small set of lines that likely describe projects/experience.
// It's intentionally simple and conservative: we prefer a few short anchors over noisy text.
func ExtractResumeSnippets(resumeText string) []string {
	text := strings.ReplaceAll(resumeText, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	// Markers that often appear around project/experience descriptions.
	markers := []string{
		"проект", "pet-project", "pet project", "url", "short", "shortener",
		"опыт", "практик", "стаж", "работа", "service", "сервис", "vac", "ваканс",
		"postgres", "gorm", "docker", "compose", "rest", "api",
	}

	seen := make(map[string]bool)
	out := make([]string, 0, 10)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		match := false
		for _, m := range markers {
			if strings.Contains(low, m) {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		// Deduplicate identical lines.
		key := low
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, line)
		if len(out) >= 12 {
			break
		}
	}

	return out
}

// Authstr returns the SuperJob OAuth authorization URL.
func Authstr() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	return "https://www.superjob.ru/authorize/?client_id=" + cfg.ClientID + "&redirect_uri=" + cfg.BaseURL + "%2Fcallback&state=custom", nil
}

// ReadTokens reads tokens from tokens.json and decodes them into types.Client.
func ReadTokens(log *zap.SugaredLogger) (types.Client, error) {
	var tokens types.Client

	body, err := os.ReadFile(consts.TokensFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// backward compatibility: раньше токены лежали в tokens.json в корне.
			if legacyBody, lerr := os.ReadFile("tokens.json"); lerr == nil {
				if uerr := json.Unmarshal(legacyBody, &tokens); uerr == nil {
					log.Warn("helpers: using legacy tokens.json; consider moving it to data/tokens.json")
					return tokens, nil
				}
			}
			log.Error("helpers: tokens file does not exist")
			return tokens, err // файла нет — токен невалиден
		}
		log.Errorf("helpers: error reading tokens file: %v", err)
		return tokens, err
	}

	err = json.Unmarshal(body, &tokens)
	if err != nil {
		log.Errorf("helpers: IsValidToken: error unmarshalling tokens: %v", err)
		return tokens, err
	}
	return tokens, nil
}

// IsValidToken checks that access and refresh tokens are present and not expired.
func IsValidToken(tokens *types.Client, log *zap.SugaredLogger) (bool, error) {
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		log.Infof("helpers: tokens are empty")
		return false, fmt.Errorf("tokens are empty") // токены пустые — невалиден
	}

	if time.Now().Unix() >= int64(tokens.Ttl) {
		log.Infof("helpers: token has expired")
		return false, fmt.Errorf("token has expired") // истёк
	}

	return true, nil // токен валиден
}

// VacancyFilters parses filters from an HTTP request's query parameters.
// This is suitable for HTTP handlers; for local stdin use a separate helper.
func VacancyFilters(r *http.Request, log *zap.SugaredLogger) (types.Filters, error) {
	var filters types.Filters
	if r == nil {
		log.Infof("helpers: VacancyFilters: nil request, returning empty filters")
		return filters, fmt.Errorf("helpers: VacancyFilters: nil request")
	}

	q := r.URL.Query()

	filters.Profession = strings.TrimSpace(q.Get("profession"))
	if filters.Profession == "" {
		return filters, fmt.Errorf("helpers: VacancyFilters: profession parameter is required")
	}

	filters.Town = strings.TrimSpace(q.Get("town"))

	filters.SalaryFrom = strings.TrimSpace(q.Get("salary_from"))

	filters.SalaryTo = strings.TrimSpace(q.Get("salary_to"))

	filters.Skills = strings.TrimSpace(q.Get("skills"))

	return filters, nil
}

// RequestString builds a SuperJob vacancies API URL using the provided filters.
func RequestString(filters types.Filters, log *zap.SugaredLogger) (string, error) {
	base := "https://api.superjob.ru/2.0/vacancies/?"

	u, err := url.Parse(base)
	if err != nil {
		log.Errorf("helpers: RequestString: error parsing base URL: %v", err)
		return "", err
	}

	q := u.Query()

	// Просим больше результатов за один запрос: это помогает при fallback-фильтрации на нашей стороне.
	// Если API проигнорирует параметр — ничего страшного.
	q.Set("count", "100")

	for _, f := range []struct {
		key   string
		value string
	}{
		{"keyword", filters.Profession},
		{"town", filters.Town},
		{"payment_from", filters.SalaryFrom},
		{"payment_to", filters.SalaryTo},
		{"skills", filters.Skills},
	} {
		if f.value != "" {
			q.Set(f.key, f.value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// DBWriteSJ upserts vacancies into Postgres.
// Ignores duplicates if external_id already exists.
func DBWriteSJ(db *gorm.DB, vacancies types.VacanciesResponse, log *zap.SugaredLogger) error {
	if len(vacancies.Objects) == 0 {
		log.Errorf("helpers: DBWriteSJ: no vacancies to write to DB")
		return fmt.Errorf("no vacancies to write to DB")
	}

	for i := range vacancies.Objects {
		v := &vacancies.Objects[i]

		if v.Town != nil {
			v.TownName = v.Town.Title
		} else {
			v.TownName = ""
		}

		if v.TypeOfWork != nil {
			v.TypeOfWorkTitle = v.TypeOfWork.Title
		} else {
			v.TypeOfWorkTitle = ""
		}

		if v.Experience != nil {
			v.ExperienceTitle = v.Experience.Title
		} else {
			v.ExperienceTitle = ""
		}

		fmt.Printf("vacancy link: %v\n", v.Link)

		v.Source = "www.superjob.ru"

		// OnConflict with DoNothing checks that external_id is unique.
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "external_id"}, {Name: "source"}}, // + host check needed in future
			DoNothing: true,
		}).Create(v).Error
		if err != nil {
			log.Errorf("helpers: error saving vacancy to database: %v", err)
			return err
		}

		log.Infof("helpers: vacancy ID: %d, Profession: %s", v.ID, v.Profession)
	}
	return nil
}

func CheckVacInDB(r *http.Request, db *gorm.DB, log *zap.SugaredLogger) (types.VacancySJ, error) {
	vacancyURL := strings.TrimSpace(r.URL.Query().Get("vacancy_url"))
	if vacancyURL == "" {
		// fallback: если vacancy_url пришёл как поле multipart/form-data
		vacancyURL = strings.TrimSpace(r.FormValue("vacancy_url"))
	}
	if vacancyURL == "" {
		return types.VacancySJ{}, fmt.Errorf("helpers: CheckVacInDB: vacancy_url is empty")
	}

	parsedURL, err := url.Parse(vacancyURL)
	if err != nil {
		log.Errorf("helpers: CheckVacInDB: error parsing vacancy_url: %v", err)
		return types.VacancySJ{}, err
	}

	vacancy := types.VacancySJ{}

	hostname := parsedURL.Host // "www.superjob.ru"
	log.Infof("helpers: CheckVacInDB: parsed hostname: %s", hostname)

	tmpl := parsedURL.Path // "/vacancies/razrab-go-12345.html"
	log.Infof("helpers: CheckVacInDB: parsed path: %s", tmpl)

	parts := strings.Split(tmpl, "-")
	log.Infof("helpers: CheckVacInDB: split path into parts: %v", parts)

	lastPart := parts[len(parts)-1]
	log.Infof("helpers: CheckVacInDB: last part of path (expected to contain ID): %s", lastPart)

	vacID := strings.Split(lastPart, ".html")[0] // "12345"
	log.Infof("helpers: CheckVacInDB: extracted vacancy ID: %s", vacID)

	externalID, convErr := strconv.Atoi(vacID)
	if convErr != nil {
		log.Errorf("helpers: CheckVacInDB: invalid vacancy ID %q: %v", vacID, convErr)
		return types.VacancySJ{}, convErr
	}

	err = db.Where("source = ? AND external_id = ?", hostname, externalID).Take(&vacancy).Error
	if err != nil {
		log.Errorf("helpers: CheckVacInDB: error querying vacancy in DB, inizialize API fetch")

		code := "https://api.superjob.ru/2.0/vacancies/" + vacID + "/"

		req, _ := http.NewRequest("GET", code, nil)

		req.Header.Set("X-Api-App-Id", os.Getenv("CLIENT_SECRET"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Errorf("helpers: CheckVacInDB: fail to get-request to superjob: %v", err)
			return types.VacancySJ{}, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Errorf("helpers: CheckVacInDB: fail to fetch vacancy, status code: %d", resp.StatusCode)
			return types.VacancySJ{}, fmt.Errorf("fail to fetch vacancy, status code: %d", resp.StatusCode)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("helpers: CheckVacInDB: fail to read response body: %v", err)
			return types.VacancySJ{}, err
		}

		err = json.Unmarshal(body, &vacancy)
		if err != nil {
			log.Errorf("helpers: CheckVacInDB: error unmarshalling vacancy data: %v", err)
			return types.VacancySJ{}, err
		}

		vacancy.Source = hostname

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

		err = db.Create(&vacancy).Error
		if err != nil {
			log.Errorf("helpers: CheckVacInDB: error saving fetched vacancy to database: %v", err)
			return types.VacancySJ{}, err
		}

		log.Infof("helpers: CheckVacInDB: fetched and saved vacancy ID: %d, Profession: %s", vacancy.ID, vacancy.Profession)
		return vacancy, nil
	}

	return vacancy, nil
}

// PDFToText extracts plain text from a PDF file using pdftotext.
// CLI-утилита pdftotext должна быть доступна в PATH или через env PDFTOTEXT_BIN.
func PDFToText(ctx context.Context, pdfPath string, log *zap.SugaredLogger) (string, error) {
	pdftotextBin := strings.TrimSpace(os.Getenv("PDFTOTEXT_BIN"))
	if pdftotextBin == "" {
		pdftotextBin = "pdftotext"
	}

	// "-" в конце значит "печать в stdout", чтобы мы забрали текст из Output/CombinedOutput.
	// Убрали -layout для простоты: LLM чаще лучше воспринимает линейный текст.
	cmd := exec.CommandContext(ctx, pdftotextBin, "-nopgbrk", pdfPath, "-")

	out, err := cmd.CombinedOutput()
	if err != nil {
		// CombinedOutput возвращает и stdout и stderr: так проще дебажить.
		log.Errorf("helpers: PDFToText: pdftotext failed: %v, out: %s", err, string(out))
		return "", err
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("helpers: PDFToText: empty text after pdftotext")
	}

	return text, nil
}

// OllamaGenerate sends prompt to Ollama /api/generate and returns "response" text.
// OLLAMA_BASE_URL должен указывать на ПК, где запущена Ollama (например http://192.168.x.x:11434).
func OllamaGenerate(ctx context.Context, prompt string, log *zap.SugaredLogger) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		return "", fmt.Errorf("helpers: OllamaGenerate: OLLAMA_MODEL is empty")
	}

	// Options tune generation quality.
	// temperature: lower => меньше "креативных" добавлений (полезно для правдивой адаптации резюме).
	// top_p: ограничение вероятностной выборки.
	temperature := 0.2
	if s := strings.TrimSpace(os.Getenv("OLLAMA_TEMPERATURE")); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			// keep it in a sane range
			if v < 0 {
				v = 0
			}
			if v > 2 {
				v = 2
			}
			temperature = v
		}
	}

	topP := 0.9
	if s := strings.TrimSpace(os.Getenv("OLLAMA_TOP_P")); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			topP = v
		}
	}

	reqBody := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": temperature,
			"top_p":       topP,
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	u := strings.TrimRight(baseURL, "/") + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(reqJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Таймаут дублируем на клиенте (помимо ctx), чтобы не висеть на сетевых проблемах.
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("helpers: OllamaGenerate: bad status: %d", resp.StatusCode)
	}

	var gen struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gen); err != nil {
		return "", err
	}

	out := strings.TrimSpace(gen.Response)
	if out == "" {
		return "", fmt.Errorf("helpers: OllamaGenerate: empty response")
	}

	return out, nil
}
