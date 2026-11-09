package database

import (
	"net/url"
	"testing"

	"github.com/trannhanlv2004/team-tickets/internal/configs"
)

func TestPostgresDSNEscapesCredentialsAndUsesConnectionSettings(t *testing.T) {
	cfg := configs.DBConfig{Host: "::1", Port: 5432, User: "user@name", Password: "p@ss:word/?", Name: "tickets", SSLMode: "require"}
	dsn := postgresDSN(cfg)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := u.User.Password()
	if u.Host != "[::1]:5432" || u.User.Username() != cfg.User || password != cfg.Password || u.Query().Get("sslmode") != "require" || u.Path != "/tickets" {
		t.Fatal("PostgreSQL DSN does not preserve the configured connection settings")
	}
}
