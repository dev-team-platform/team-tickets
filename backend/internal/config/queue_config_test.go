package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadAppConfigQueueSettings(t *testing.T) {
	setValidEnvironment(t)
	for name, value := range map[string]string{
		"EXPORT_WORKERS":           "4",
		"EXPORT_QUEUE_CAPACITY":    "500",
		"EXPORT_MAX_RETRY":         "2",
		"EXPORT_RETRY_DELAY_MS":    "10, 20",
		"EXPORT_JITTER_FROM_MS":    "5",
		"EXPORT_JITTER_TO_MS":      "8",
		"AUDIT_LOG_QUEUE_CAPACITY": "900",
		"AUDIT_LOG_MAX_RETRY":      "1",
		"AUDIT_LOG_RETRY_DELAY_MS": "30,40",
		"AUDIT_LOG_JITTER_FROM_MS": "3",
		"AUDIT_LOG_JITTER_TO_MS":   "9",
	} {
		t.Setenv(name, value)
	}
	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Export.Workers != 4 || cfg.Export.QueueCapacity != 500 || cfg.Export.MaxRetry != 2 || !reflect.DeepEqual(cfg.Export.RetryDelayMS, []int{10, 20}) || cfg.Export.JitterFromMS != 5 || cfg.Export.JitterToMS != 8 {
		t.Fatalf("unexpected export config: %+v", cfg.Export)
	}
	if cfg.AuditLog.QueueCapacity != 900 || cfg.AuditLog.MaxRetry != 1 || !reflect.DeepEqual(cfg.AuditLog.RetryDelayMS, []int{30, 40}) || cfg.AuditLog.JitterFromMS != 3 || cfg.AuditLog.JitterToMS != 9 {
		t.Fatalf("unexpected audit config: %+v", cfg.AuditLog)
	}
}

func TestLoadAppConfigRejectsInvalidQueueSettings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "EXPORT_QUEUE_CAPACITY", value: "0"},
		{name: "AUDIT_LOG_QUEUE_CAPACITY", value: "bad"},
		{name: "EXPORT_MAX_RETRY", value: "-1"},
		{name: "AUDIT_LOG_RETRY_DELAY_MS", value: "10,no"},
		{name: "EXPORT_RETRY_DELAY_MS", value: "-1"},
		{name: "AUDIT_LOG_JITTER_FROM_MS", value: "-1"},
		{name: "EXPORT_JITTER_TO_MS", value: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tc.name, tc.value)
			_, err := LoadAppConfig()
			if err == nil || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("expected error for %s, got %v", tc.name, err)
			}
		})
	}
}

func TestLoadAuditLogWorkerSettings(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("AUDIT_LOG_WORKERS", "3")
	t.Setenv("AUDIT_LOG_SHUTDOWN_TIMEOUT_SECONDS", "20")
	t.Setenv("AUDIT_LOG_WRITE_TIMEOUT_SECONDS", "7")
	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuditLog.Workers != 3 || cfg.AuditLog.ShutdownTimeoutSeconds != 20 || cfg.AuditLog.WriteTimeoutSeconds != 7 {
		t.Fatalf("unexpected audit worker config: %+v", cfg.AuditLog)
	}
}

func TestLoadAuditLogRejectsInvalidWorkerSettings(t *testing.T) {
	for _, name := range []string{"AUDIT_LOG_WORKERS", "AUDIT_LOG_SHUTDOWN_TIMEOUT_SECONDS", "AUDIT_LOG_WRITE_TIMEOUT_SECONDS"} {
		t.Run(name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(name, "0")
			_, err := LoadAppConfig()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("expected error for %s, got %v", name, err)
			}
		})
	}
}
