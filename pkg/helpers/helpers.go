package helpers

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Maltide/JoBot/pkg/config"
	"github.com/Maltide/JoBot/pkg/consts"
	"github.com/Maltide/JoBot/pkg/types"
	"go.uber.org/zap"
)

func Authstr() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	return "https://www.superjob.ru/authorize/?client_id=" + cfg.ClientID + "&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&state=custom", nil
}

func IsValidToken(tokens *types.Client, log *zap.SugaredLogger) (bool, error) {
	body, err := os.ReadFile(consts.TokensFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // файла нет — токен невалиден
		}
		log.Errorf("helpers: error reading tokens file: %v", err)
		return false, err
	}

	err = json.Unmarshal(body, tokens)
	if err != nil {
		log.Errorf("helpers: error unmarshalling tokens: %v", err)
		return false, err
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		return false, nil // токены пустые — невалиден
	}

	if time.Now().Unix() >= int64(tokens.Ttl) {
		return false, nil // истёк
	}

	return true, nil // токен валиден
}
