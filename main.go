package main

import (
	"os"

	"github.com/Maltide/jobotparse/pkg/config"
	"github.com/Maltide/jobotparse/pkg/db"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/logger"
	"github.com/Maltide/jobotparse/pkg/server"
	"github.com/Maltide/jobotparse/pkg/superjob"
	"github.com/Maltide/jobotparse/pkg/types"
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

	err = database.AutoMigrate(&types.Vacancy{})
	if err != nil {
		log.Errorf("main: error automigrating Vacancy table: %v", err)
		return
	}

	sj := superjob.NewSuperJobClient(nil, cfg.ClientSecret)
	apis := []interfaces.VacanciesProvider{sj}

	err = server.GetServer(log, database, apis)
	if err != nil {
		return
	}

}
