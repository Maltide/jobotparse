package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Maltide/jobotparse/pkg/consts"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

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
