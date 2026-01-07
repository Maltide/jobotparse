package main

import (
	"os"

	"github.com/Maltide/JoBot/pkg/config"
	"github.com/Maltide/JoBot/pkg/logger"
	"github.com/Maltide/JoBot/pkg/server"
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

	err = server.GetServer(log)
	if err != nil {
		return
	}

}
