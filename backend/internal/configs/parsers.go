package configs

import (
	"fmt"
	"os"
	"strconv"
)

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func intEnv(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}

// portEnv returns zero for an unset optional port and validates configured ports.
func portEnv(name string, fallback int) (int, error) {
	if os.Getenv(name) == "" {
		return fallback, nil
	}
	port, err := intEnv(name, fallback)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s: port must be between 1 and 65535", name)
	}
	return port, nil
}
