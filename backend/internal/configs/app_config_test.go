package configs

import (
	"strings"
	"testing"
)

func TestLoadAppConfigReadsEnvironment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("APP_PORT", "8123")
	t.Setenv("DB_MAX_OPEN_CONNS", "12")
	t.Setenv("AUTH_SELF_REGISTRATION_ENABLED", "true")
	config, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Port != 8123 || config.DB.MaxOpenConns != 12 {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestLoadAppConfigRejectsInvalidInteger(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "invalid")
	_, err := LoadAppConfig()
	if err == nil || !strings.Contains(err.Error(), "DB_MAX_OPEN_CONNS") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAppConfigReadsIntegrationSettings(t *testing.T) {
	setValidEnvironment(t)
	values := map[string]string{
		"REDIS_HOST": "redis", "REDIS_PORT": "6379", "REDIS_PASSWORD": "redis-pass",
		"RABBITMQ_DEFAULT_USER": "rabbit-user", "RABBITMQ_DEFAULT_PASS": "rabbit-pass", "RABBITMQ_DEFAULT_VHOST": "/app", "RABBITMQ_HOST": "rabbit", "RABBITMQ_PORT": "5672",
		"OBJECT_STORAGE_ENDPOINT": "https://storage.example", "OBJECT_STORAGE_BUCKET_NAME": "bucket", "OBJECT_STORAGE_REGION": "region", "OBJECT_STORAGE_ACCESS_KEY": "access", "OBJECT_STORAGE_SECRET_KEY": "secret", "OBJECT_STORAGE_FORCE_PATH_STYLE": "1",
		"AUTH_INTERNAL_JWT_ISSUER": "issuer", "AUTH_INTERNAL_JWT_AUDIENCE": "audience", "AUTH_INTERNAL_JWT_PUBLIC_KEY_PEM_PATH": "/keys/public.pem",
	}
	for name, value := range values {
		t.Setenv(name, value)
	}
	config, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Redis != (RedisConfig{Host: "redis", Port: 6379, Password: "redis-pass"}) {
		t.Error("incorrect Redis configuration")
	}
	if config.RabbitMQ != (RabbitMQConfig{User: "rabbit-user", Password: "rabbit-pass", VHost: "/app", Host: "rabbit", Port: 5672}) {
		t.Error("incorrect RabbitMQ configuration")
	}
	if config.ObjectStorage != (ObjectStorageConfig{Endpoint: "https://storage.example", BucketName: "bucket", Region: "region", AccessKey: "access", SecretKey: "secret", ForcePathStyle: true}) {
		t.Error("incorrect object storage configuration")
	}
	if config.Auth.InternalJWTIssuer != "issuer" || config.Auth.InternalJWTAudience != "audience" || config.Auth.InternalJWTPublicKeyPEMPath != "/keys/public.pem" {
		t.Error("incorrect internal JWT configuration")
	}
}

func TestLoadAppConfigRejectsInvalidObjectStoragePathStyle(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("OBJECT_STORAGE_FORCE_PATH_STYLE", "invalid")
	_, err := LoadAppConfig()
	if err == nil || !strings.Contains(err.Error(), "OBJECT_STORAGE_FORCE_PATH_STYLE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAppConfigRejectsInvalidPorts(t *testing.T) {
	setValidEnvironment(t)
	for _, name := range []string{"APP_PORT", "DB_PORT", "REDIS_PORT", "RABBITMQ_PORT"} {
		for _, value := range []string{"invalid", "0", "65536"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				t.Setenv(name, value)
				_, err := LoadAppConfig()
				if err == nil || !strings.Contains(err.Error(), name) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	}
}

func setValidEnvironment(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{
		"DB_HOST": "db", "DB_USER": "user", "DB_NAME": "tickets", "REDIS_HOST": "redis",
		"AUTH_INTERNAL_JWT_ISSUER": "issuer", "AUTH_INTERNAL_JWT_AUDIENCE": "audience", "AUTH_INTERNAL_JWT_PUBLIC_KEY_PEM_PATH": "/keys/public.pem",
		"OBJECT_STORAGE_REGION": "region", "OBJECT_STORAGE_BUCKET_NAME": "bucket",
		"OBJECT_STORAGE_ENDPOINT": "", "OBJECT_STORAGE_ACCESS_KEY": "", "OBJECT_STORAGE_SECRET_KEY": "",
	} {
		t.Setenv(name, value)
	}
}

func TestLoadAppConfigRejectsInvalidClientConfiguration(t *testing.T) {
	tests := []struct{ name, value, expected string }{
		{"DB_HOST", "", "DB_HOST"}, {"DB_USER", "", "DB_USER"}, {"DB_NAME", "", "DB_NAME"},
		{"DB_MAX_OPEN_CONNS", "-1", "DB_MAX_OPEN_CONNS"}, {"DB_MAX_IDLE_CONNS", "-1", "DB_MAX_IDLE_CONNS"}, {"DB_CONN_MAX_LIFETIME", "-1", "DB_CONN_MAX_LIFETIME"},
		{"REDIS_HOST", "", "REDIS_HOST"}, {"OBJECT_STORAGE_REGION", "", "OBJECT_STORAGE_REGION"}, {"OBJECT_STORAGE_BUCKET_NAME", "", "OBJECT_STORAGE_BUCKET_NAME"},
		{"OBJECT_STORAGE_ACCESS_KEY", "key", "OBJECT_STORAGE_ACCESS_KEY"}, {"OBJECT_STORAGE_SECRET_KEY", "secret", "OBJECT_STORAGE_SECRET_KEY"},
		{"OBJECT_STORAGE_ENDPOINT", "storage", "OBJECT_STORAGE_ENDPOINT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tt.name, tt.value)
			_, err := LoadAppConfig()
			if err == nil || !strings.Contains(err.Error(), tt.expected) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
