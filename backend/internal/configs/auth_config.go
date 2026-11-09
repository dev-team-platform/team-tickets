package configs

import (
	"fmt"
	"os"
	"strings"
)

type AuthConfig struct {
	InternalJWTIssuer           string
	InternalJWTAudience         string
	InternalJWTPublicKeyPEMPath string
}

func (cfg AuthConfig) Validate() error {
	if strings.TrimSpace(cfg.InternalJWTIssuer) == "" || strings.TrimSpace(cfg.InternalJWTAudience) == "" || strings.TrimSpace(cfg.InternalJWTPublicKeyPEMPath) == "" {
		return fmt.Errorf("AUTH_INTERNAL_JWT_ISSUER, AUTH_INTERNAL_JWT_AUDIENCE, and AUTH_INTERNAL_JWT_PUBLIC_KEY_PEM_PATH are required")
	}
	return nil
}

func loadAuthConfig() (AuthConfig, error) {
	cfg := AuthConfig{
		InternalJWTIssuer: os.Getenv("AUTH_INTERNAL_JWT_ISSUER"), InternalJWTAudience: os.Getenv("AUTH_INTERNAL_JWT_AUDIENCE"), InternalJWTPublicKeyPEMPath: os.Getenv("AUTH_INTERNAL_JWT_PUBLIC_KEY_PEM_PATH"),
	}
	return cfg, cfg.Validate()
}
