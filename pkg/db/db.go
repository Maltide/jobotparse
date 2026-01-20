package db

import (
	"time"

	"github.com/Maltide/jobotparse/pkg/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnDB(log *zap.SugaredLogger) (*gorm.DB, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("db: error getting config: %v", err)
		return nil, err
	}

	dsn := "host=" + cfg.PgHost + " user=" + cfg.PgUser + " password=" + cfg.PgPassword + " dbname=" + cfg.PgDB + " port=" + cfg.PgPort + " sslmode=" + cfg.PgSSLMode

	var lastErr error
	var db *gorm.DB

	maxattempts := 3
	for attempt := 1; attempt <= maxattempts; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, err := db.DB()
			if err == nil {
				errPing := sqlDB.Ping()
				if errPing == nil {
					return db, nil
				} else {
					err = errPing
				}
			}
		}

		lastErr = err
		log.Infof("db: ConnDB: connect attempt %d failed: %v", attempt, lastErr)
		time.Sleep(time.Second)
	}
	return nil, lastErr
}
