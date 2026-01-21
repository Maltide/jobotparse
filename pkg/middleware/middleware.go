package middleware

import (
	"github.com/Maltide/jobotparse/pkg/helpers"
	"go.uber.org/zap"
)

func BeforeRequest(log *zap.SugaredLogger) error {
	tokens, err := helpers.ReadTokens(log)
	if err != nil {
		return err
	}

	ok, err := helpers.IsValidToken(&tokens, log)
	if err != nil {
		return err
	}
	if !ok {
		log.Infof("middleware: token is invalid or expired, trying refresh")
		err := RefreshTokens(tokens.RefreshToken, log)
		if err != nil {
			log.Error("middleware: failed to refresh token")
			return err
		}
		tokens, err = helpers.ReadTokens(log)
		if err != nil {
			return err
		}
		ok, err = helpers.IsValidToken(&tokens, log)
		if err != nil {
			return err
		}
		if !ok {
			log.Error("middleware: tokens are still invalid after refresh, need to authorize")
			return err
		}
	}

	return nil
}
