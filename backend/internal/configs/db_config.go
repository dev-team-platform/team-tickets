package configs

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Host, User, Password, Name, SSLMode         string
	Port                                        int
	MaxOpenConns, MaxIdleConns, ConnMaxLifetime int
}

func loadDBConfig() (DBConfig, error) {
	port, err := portEnv("DB_PORT", 0)
	if err != nil {
		return DBConfig{}, err
	}
	maxOpen, err := intEnv("DB_MAX_OPEN_CONNS", 10)
	if err != nil {
		return DBConfig{}, err
	}
	maxIdle, err := intEnv("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return DBConfig{}, err
	}
	lifetime, err := intEnv("DB_CONN_MAX_LIFETIME", 300)
	if err != nil {
		return DBConfig{}, err
	}
	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_USER") == "" || os.Getenv("DB_NAME") == "" {
		return DBConfig{}, fmt.Errorf("DB_HOST, DB_USER, and DB_NAME are required")
	}
	if maxOpen < 0 || maxIdle < 0 || lifetime < 0 {
		return DBConfig{}, fmt.Errorf("DB_MAX_OPEN_CONNS, DB_MAX_IDLE_CONNS, and DB_CONN_MAX_LIFETIME cannot be negative")
	}
	return DBConfig{
		Host:            os.Getenv("DB_HOST"),
		Port:            port,
		User:            os.Getenv("DB_USER"),
		Password:        os.Getenv("DB_PASSWORD"),
		Name:            os.Getenv("DB_NAME"),
		SSLMode:         os.Getenv("DB_SSL_MODE"),
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: lifetime,
	}, nil
}
