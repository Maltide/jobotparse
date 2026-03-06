package db

import (
	"time"

	"github.com/Maltide/jobotparse/pkg/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnDB connects to PostgreSQL using configuration from .env.
// It retries a few times because in Docker Compose the DB container may not be ready when the app starts.
func ConnDB(log *zap.SugaredLogger) (*gorm.DB, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf("db: error getting config: %v", err)
		return nil, err
	}

	// DSN format for the postgres driver.
	dsn := "host=" + cfg.PgHost + " user=" + cfg.PgUser + " password=" + cfg.PgPassword + " dbname=" + cfg.PgDB + " port=" + cfg.PgPort + " sslmode=" + cfg.PgSSLMode

	var lastErr error
	var db *gorm.DB

	maxattempts := 12
	for attempt := 1; attempt <= maxattempts; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, err := db.DB()
			if err == nil {
				// Verify actual connectivity (gorm.Open may succeed before DB accepts connections).
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
		// Небольшой backoff: 1s, 2s, 3s ... до 5s
		sleepSec := attempt
		if sleepSec > 5 {
			sleepSec = 5
		}
		time.Sleep(time.Duration(sleepSec) * time.Second)
	}
	return nil, lastErr
}
