package helpers

import "github.com/Maltide/jobotparse/pkg/config"

// Authstr returns the SuperJob OAuth authorization URL.
func Authstr() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	return "https://www.superjob.ru/authorize/?client_id=" + cfg.ClientID + "&redirect_uri=" + cfg.BaseURL + "%2Fcallback&state=custom", nil
}
