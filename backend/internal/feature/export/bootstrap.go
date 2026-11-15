package export

import (
	"fmt"
	"log/slog"

	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/export/service"
)

// Bootstrap starts the configured export queue and workers.
// The caller must shut down the returned queue after export producers stop.
func Bootstrap(cfg config.ExportConfig, logger *slog.Logger) (*service.Queue, error) {
	if logger == nil {
		logger = slog.Default()
	}
	queue, err := service.NewQueue(cfg.Workers, cfg.QueueCapacity, func(err error) {
		logger.Error("export job failed", "error", err)
	})
	if err != nil {
		return nil, fmt.Errorf("start export workers: %w", err)
	}
	return queue, nil
}
