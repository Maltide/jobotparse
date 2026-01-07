package server

import (
	"fmt"
	"net/http"

	"github.com/Maltide/JoBot/pkg/handlers"
	"github.com/Maltide/JoBot/pkg/middleware"
	"go.uber.org/zap"
)

func GetServer(log *zap.SugaredLogger) error {
	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Сервер запущен")
	})

	router.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		err := handlers.Authorize(w, r, log)
		if err != nil {
			log.Errorf("from server.go: error in authorize handler: %v", err)
		}
	})

	router.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		err := middleware.GetAccessToken(w, r, log)
		if err != nil {
			log.Errorf("from server.go: error in get access token middleware: %v", err)
		}
	})

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Errorf("fail to create server: %v", err)
		return err
	}
	return nil
}
