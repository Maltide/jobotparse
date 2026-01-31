package middleware

import (
	"github.com/Maltide/jobotparse/pkg/helpers"
	"go.uber.org/zap"
)

// BeforeRequest ensures that a valid access token exists before calling API.
// If the access token is expired, it attempts to refresh it using the refresh token.
// If refresh fails, callers should redirect the user to /auth to re-authorize.
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
		// RefreshTokens updates tokens.json
		err := RefreshTokens(tokens.RefreshToken, log)
		if err != nil {
			log.Error("middleware: failed to refresh token")
			return err
		}
		// Re-read and re-validate after refresh to ensure we have a usable token.
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
