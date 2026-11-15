package auditlog

import (
	"fmt"
	"log/slog"

	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	"gorm.io/gorm"
)

// Bootstrap registers audit callbacks and starts the configured worker pool.
// The caller must shut down the returned queue after audit producers stop.
func Bootstrap(db *gorm.DB, cfg config.AuditLogConfig, logger *slog.Logger) (*service.Queue, error) {
	persister, err := service.NewPersister(db, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("configure audit persister: %w", err)
	}
	if cfg.QueueCapacity < 1 || cfg.Workers < 1 {
		return nil, fmt.Errorf("audit queue capacity and worker count must be positive")
	}
	if err := service.RegisterCallbacks(db); err != nil {
		return nil, fmt.Errorf("register audit callbacks: %w", err)
	}
	queue, err := service.NewQueue(cfg.QueueCapacity, cfg.Workers, persister.Persist, logger)
	if err != nil {
		return nil, fmt.Errorf("start audit workers: %w", err)
	}
	return queue, nil
}
