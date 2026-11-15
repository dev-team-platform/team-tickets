package config

import (
	"fmt"
	"math"
	"time"
)

type AuditLogConfig struct {
	QueueConfig
	Workers                int
	ShutdownTimeoutSeconds int
	WriteTimeoutSeconds    int
}

func loadAuditLogConfig() (AuditLogConfig, error) {
	queue, err := loadQueueConfig("AUDIT_LOG", 10000, "1000,5000,10000", 3000)
	if err != nil {
		return AuditLogConfig{}, err
	}
	if len(queue.RetryDelayMS) < queue.MaxRetry {
		return AuditLogConfig{}, fmt.Errorf("AUDIT_LOG_RETRY_DELAY_MS must provide a delay for each retry")
	}
	workers, err := intEnv("AUDIT_LOG_WORKERS", 2)
	if err != nil {
		return AuditLogConfig{}, err
	}
	shutdown, err := intEnv("AUDIT_LOG_SHUTDOWN_TIMEOUT_SECONDS", 10)
	if err != nil {
		return AuditLogConfig{}, err
	}
	writeTimeout, err := intEnv("AUDIT_LOG_WRITE_TIMEOUT_SECONDS", 5)
	if err != nil {
		return AuditLogConfig{}, err
	}
	if workers < 1 || shutdown < 1 || writeTimeout < 1 || int64(shutdown) > math.MaxInt64/int64(time.Second) || int64(writeTimeout) > math.MaxInt64/int64(time.Second) {
		return AuditLogConfig{}, fmt.Errorf("AUDIT_LOG_WORKERS, AUDIT_LOG_SHUTDOWN_TIMEOUT_SECONDS, and AUDIT_LOG_WRITE_TIMEOUT_SECONDS must be positive and timeouts must fit in time.Duration")
	}
	return AuditLogConfig{
		QueueConfig:            queue,
		Workers:                workers,
		ShutdownTimeoutSeconds: shutdown,
		WriteTimeoutSeconds:    writeTimeout,
	}, nil
}
