package main

import (
	"os"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/db"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/logger"
	"github.com/Maltide/jobotparse/pkg/ollama"
	"github.com/Maltide/jobotparse/pkg/server"
	"github.com/Maltide/jobotparse/pkg/superjob"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		return
	}

	log, err := logger.GetLogger(cfg.LogLevel)
	if err != nil {
		os.Exit(1)
		return
	}

	log.Infof("logger is working in main.go")

	database, err := db.ConnDB(log)
	if err != nil {
		return
	}

	sj := superjob.NewSuperJobClient(nil, cfg.ClientSecret)

	apis := []interfaces.VacanciesProvider{sj}

	ollamasession := ollama.CreateOllama(log)

	err = server.GetServer(log, database, apis, ollamasession)
	if err != nil {
		return
	}

}
