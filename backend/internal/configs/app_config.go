package configs

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Env           string
	Port          int
	LogDir        string
	DB            DBConfig
	Auth          AuthConfig
	Redis         RedisConfig
	RabbitMQ      RabbitMQConfig
	ObjectStorage ObjectStorageConfig
}

// LoadAppConfig reads environment variables, optionally supplemented by a local .env file.
// Variables already present in the environment take precedence over the file.
func LoadAppConfig() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}
	port, err := portEnv("APP_PORT", 8200)
	if err != nil {
		return nil, err
	}
	db, err := loadDBConfig()
	if err != nil {
		return nil, err
	}
	auth, err := loadAuthConfig()
	if err != nil {
		return nil, err
	}

	objectStorage, err := loadObjectStorageConfig()
	if err != nil {
		return nil, err
	}

	redis, err := loadRedisConfig()
	if err != nil {
		return nil, err
	}

	rabbitMQ, err := loadRabbitMQConfig()
	if err != nil {
		return nil, err
	}

	origins := []string{}
	for _, origin := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	return &AppConfig{
		Env:           envOr("APP_ENV", "development"),
		Port:          port,
		LogDir:        os.Getenv("LOG_DIR"),
		DB:            db,
		Auth:          auth,
		Redis:         redis,
		RabbitMQ:      rabbitMQ,
		ObjectStorage: objectStorage,
	}, nil
}
