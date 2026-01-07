package handlers

import (
	"net/http"

	"github.com/Maltide/JoBot/pkg/helpers"
	"github.com/Maltide/JoBot/pkg/types"
	"go.uber.org/zap"
)

func Authorize(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	var tokensinfo types.Client

	ok, err := helpers.IsValidToken(&tokensinfo, log)
	if err != nil {
		return err
	}
	if !ok {
		log.Infof("handlers: tokens are present, no need to authorize")
		return nil
	}

	url, err := helpers.Authstr()
	if err != nil {
		log.Errorf("handlers: error getting auth URL: %v", err)
		return err
	}

	http.Redirect(w, r, url, http.StatusFound)

	return nil
}
