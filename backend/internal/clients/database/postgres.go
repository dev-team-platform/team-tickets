package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/configs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres opens and verifies a PostgreSQL connection. The caller must
// close the underlying sql.DB obtained through db.DB() when finished.
func ConnectPostgres(ctx context.Context, cfg configs.DBConfig) (*gorm.DB, error) {
	dsn := postgresDSN(cfg)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get postgres pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

func postgresDSN(cfg configs.DBConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + cfg.Name,
	}
	query := u.Query()
	if cfg.SSLMode != "" {
		query.Set("sslmode", cfg.SSLMode)
	}
	query.Set("TimeZone", "UTC")
	u.RawQuery = query.Encode()
	return u.String()
}
