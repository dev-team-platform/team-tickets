package service

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Persister struct {
	db      *gorm.DB
	config  config.AuditLogConfig
	logger  *slog.Logger
	retries atomic.Uint64
	insert  func(context.Context, Batch) error
}

func NewPersister(db *gorm.DB, cfg config.AuditLogConfig, logger *slog.Logger) (*Persister, error) {
	if db == nil || cfg.MaxRetry < 0 || len(cfg.RetryDelayMS) < cfg.MaxRetry || cfg.WriteTimeoutSeconds < 1 || cfg.JitterFromMS < 0 || cfg.JitterToMS < cfg.JitterFromMS {
		return nil, fmt.Errorf("invalid audit persister configuration")
	}
	if int64(cfg.WriteTimeoutSeconds) > math.MaxInt64/int64(time.Second) {
		return nil, fmt.Errorf("audit write timeout exceeds duration range")
	}
	if _, err := util.Jitter(cfg.JitterFromMS, cfg.JitterToMS); err != nil {
		return nil, err
	}
	for _, delay := range cfg.RetryDelayMS[:cfg.MaxRetry] {
		if delay < 0 || int64(delay) > math.MaxInt64/int64(time.Millisecond)-int64(cfg.JitterToMS) {
			return nil, fmt.Errorf("audit retry delay exceeds duration range")
		}
	}
	if logger == nil {
		logger = slog.Default()
	}
	p := &Persister{
		db:     db,
		config: cfg,
		logger: logger,
	}
	p.insert = p.insertBatch
	return p, nil
}

func (p *Persister) insertBatch(ctx context.Context, batch Batch) error {
	return p.db.WithContext(internalWrite(ctx)).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{
				Name: "id",
			},
		},
		DoNothing: true,
	}).Create(&batch).Error
}

// Persist retries a complete idempotent batch on transient errors. MaxRetry is
// the number of retries after the initial attempt, with one configured delay per retry.
func (p *Persister) Persist(ctx context.Context, batch Batch) error {
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		writeCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.WriteTimeoutSeconds)*time.Second)
		err := p.insert(writeCtx, batch)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !transient(err) || attempt >= p.config.MaxRetry {
			return fmt.Errorf("audit batch persistence failed (transient=%t)", transient(err))
		}
		jitter, jitterErr := util.Jitter(p.config.JitterFromMS, p.config.JitterToMS)
		if jitterErr != nil {
			return jitterErr
		}
		delay := time.Duration(p.config.RetryDelayMS[attempt])*time.Millisecond + jitter
		p.retries.Add(1)
		attrs := []any{
			"transaction_id",
			batchTransactionID(batch),
			"retry",
			attempt + 1,
			"delay_ms",
			delay.Milliseconds(),
		}
		if len(batch) > 0 {
			attrs = append(attrs, "actor_id", batch[0].ActorID)
			if batch[0].TraceID != nil {
				attrs = append(attrs, "trace_id", *batch[0].TraceID)
			}
		}
		p.logger.Warn("retrying audit batch", attrs...)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (p *Persister) RetryAttempts() uint64 { return p.retries.Load() }

func transient(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, driver.ErrBadConn) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "08") ||
			strings.HasPrefix(pgErr.Code, "40") ||
			strings.HasPrefix(pgErr.Code, "53") ||
			pgErr.Code == "55P03" ||
			strings.HasPrefix(pgErr.Code, "57P0")
	}
	return false
}
