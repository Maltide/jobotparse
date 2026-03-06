package ollama

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatSession is the request payload we keep in-memory between calls.
// Ollama requires sending the full messages history on each request.
type ChatSession struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ChatResponse matches Ollama /api/chat response (non-stream).
type ChatResponse struct {
	Model   string  `json:"model"`
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

func CreateOllama(log *zap.SugaredLogger) *ChatSession {
	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		model = "gpt-oss:120b-cloud"
	}

	return &ChatSession{
		Model:    model,
		Messages: []Message{},
		Stream:   false,
	}
}
func OllamaRequest(ollama *ChatSession, log *zap.SugaredLogger) (Message, error) {
	u := "https://ollama.com/api/chat"

	ollamaBytes, err := json.Marshal(ollama)
	if err != nil {
		log.Errorf("Failed to marshal request body: %v", err)
		return Message{}, err
	}

	req, err := http.NewRequest("POST", u, bytes.NewBuffer(ollamaBytes))
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	if apiKey := os.Getenv("OLLAMA_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	var client http.Client

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to send request: %v", err)
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Received non-OK response: %s", resp.Status)
		return Message{}, errors.New("no 200")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read response body: %v", err)
		return Message{}, err
	}

	var response ChatResponse

	if err := json.Unmarshal(body, &response); err != nil {
		log.Errorf("Failed to unmarshal response into ChatResponse: %v", err)
		return Message{}, err
	}

	return response.Message, nil
}
