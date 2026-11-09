package configs

import (
	"fmt"
	"os"
)

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

func loadRedisConfig() (RedisConfig, error) {
	port, err := portEnv("REDIS_PORT", 0)
	if err != nil {
		return RedisConfig{}, err
	}
	if os.Getenv("REDIS_HOST") == "" {
		return RedisConfig{}, fmt.Errorf("REDIS_HOST is required")
	}
	return RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Port:     port,
		Password: os.Getenv("REDIS_PASSWORD"),
	}, nil
}
