package config

import (
	"fmt"
	"runtime"
)

type ExportConfig struct {
	QueueConfig
	Workers int
}

func loadExportConfig() (ExportConfig, error) {
	queue, err := loadQueueConfig("EXPORT", 100, "1000,3000,5000", 2000)
	if err != nil {
		return ExportConfig{}, err
	}
	workers, err := intEnv("EXPORT_WORKERS", runtime.NumCPU())
	if err != nil {
		return ExportConfig{}, err
	}
	if workers < 1 {
		return ExportConfig{}, fmt.Errorf("EXPORT_WORKERS must be positive")
	}
	return ExportConfig{
		QueueConfig: queue,
		Workers:     workers,
	}, nil
}
