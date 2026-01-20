package middleware

import (
	"io"
	"net/http"
	"os"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/consts"
	"go.uber.org/zap"
)

func ProcessAccessToken(code string, log *zap.SugaredLogger) error {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("fail to get config: %v", err)
		return err
	}

	resp, err := http.Get("https://api.superjob.ru/2.0/oauth2/access_token/?code=" + code + "&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret)
	if err != nil {
		log.Errorf("fail to get-request to superjob: %v", err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("fail to refresh tokens, status code: %d", resp.StatusCode)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("fail to read response body: %v", err)
		return err
	}

	log.Infof("response access body: %s", string(body))

	err = os.WriteFile(consts.TokensFilePath, body, 0644)
	if err != nil {
		log.Errorf("fail to write tokens file: %v", err)
		return err
	}
	return nil
}

func RefreshTokens(refToken string, log *zap.SugaredLogger) error {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("fail to get config: %v", err)
		return err
	}

	resp, err := http.Get("https://api.superjob.ru/2.0/oauth2/refresh_token/?refresh_token=" + refToken + "&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret)
	if err != nil {
		log.Errorf("fail to get-request to superjob: %v", err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("fail to refresh tokens, status code: %d", resp.StatusCode)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("fail to read response body: %v", err)
		return err
	}

	log.Infof("response refresh body: %s", string(body))

	err = os.WriteFile(consts.TokensFilePath, body, 0644)
	if err != nil {
		log.Errorf("fail to write tokens file: %v", err)
		return err
	}

	return nil
}

func GetAccessToken(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Errorf("no code in request")
		return nil
	}

	err := ProcessAccessToken(code, log)
	if err != nil {
		return err
	}

	return nil
}
