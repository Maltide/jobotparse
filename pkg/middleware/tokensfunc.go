package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/consts"
	"go.uber.org/zap"
)

// ProcessAccessToken exchanges an OAuth authorization code for access/refresh tokens.
func ProcessAccessToken(code string, log *zap.SugaredLogger) error {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("fail to get config: %v", err)
		return err
	}

	redirectURI := cfg.BaseURL + "/callback"
	// url.QueryEscape percent-encodes the redirect URI so it can be safely used as the value of a query parameter.
	// It encodes characters such as ':' '/' '?' '&' spaces and others into %HH sequences.
	resp, err := http.Get("https://api.superjob.ru/2.0/oauth2/access_token/?code=" + code + "&redirect_uri=" + url.QueryEscape(redirectURI) + "&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret)
	if err != nil {
		log.Errorf("fail to get-request to superjob: %v", err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("fail to refresh tokens, status code: %d", resp.StatusCode)
		return fmt.Errorf("superjob access token request failed, status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("fail to read response body: %v", err)
		return err
	}

	log.Infof("response access body: %s", string(body))

	// Store tokens with read permissions for the app user.
	// Mode 0644 means: owner read/write (6), group read (4), others read (4).
	// The leading 0 denotes an octal literal.
	err = os.WriteFile(consts.TokensFilePath, body, 0644)
	if err != nil {
		log.Errorf("fail to write tokens file: %v", err)
		return err
	}
	return nil
}

// RefreshTokens uses a refresh token to obtain a new access token.
// On success it overwrites tokens.json with the refreshed response payload.
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
		return fmt.Errorf("superjob refresh token request failed, status: %d", resp.StatusCode)
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

// GetAccessToken extracts the OAuth code from the callback request and calls ProcessAccessToken.
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
