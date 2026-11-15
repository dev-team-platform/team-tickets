package report

import (
	"context"
	"io"
	"time"
)

// GenerateReportsService generates a report from data in the requested time zone.
// The caller owns and must close the returned stream. Implementations should
// honor context cancellation and return an error for a nil time zone.
type GenerateReportsService[T any] interface {
	Generate(ctx context.Context, data T, timeZone *time.Location) (io.ReadCloser, error)
}
