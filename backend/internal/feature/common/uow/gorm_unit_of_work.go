package uow

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	userrepos "github.com/trannhanlv2004/team-tickets/internal/feature/user/repository"
	"gorm.io/gorm"
)

type transactionKey struct{}

type BatchEnqueuer interface {
	Enqueue(auditlogservices.Batch) bool
}

type gormUnitOfWork struct {
	db       *gorm.DB
	queue    BatchEnqueuer
	logger   *slog.Logger
	mu       sync.Mutex
	active   int
	stopping bool
	drained  chan struct{}
}

func New(db *gorm.DB, queue BatchEnqueuer, logger *slog.Logger) (UnitOfWork, error) {
	if db == nil || queue == nil {
		return nil, errors.New("unit of work requires a database and audit queue")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &gormUnitOfWork{
		db:      db,
		queue:   queue,
		logger:  logger,
		drained: make(chan struct{}),
	}, nil
}

func (u *gormUnitOfWork) Execute(ctx context.Context, fn func(repos Repositories) error) error {
	if fn == nil {
		return errors.New("unit of work requires a function")
	}
	parent, nested := auditlogservices.CollectorFromContext(ctx)
	if !nested {
		u.mu.Lock()
		if u.stopping {
			u.mu.Unlock()
			return errors.New("unit of work is shutting down")
		}
		u.active++
		u.mu.Unlock()
		defer func() {
			u.mu.Lock()
			u.active--
			if u.stopping && u.active == 0 {
				close(u.drained)
			}
			u.mu.Unlock()
		}()
	}
	var txID uuid.UUID
	if nested {
		txID = parent.TransactionID
	} else {
		var err error
		txID, err = uuid.NewV7()
		if err != nil {
			return err
		}
	}
	collector := auditlogservices.NewCollector(txID)
	ctx = auditlogservices.WithCollector(ctx, collector)
	db := u.db
	if nested {
		var ok bool
		db, ok = ctx.Value(transactionKey{}).(*gorm.DB)
		if !ok || db == nil {
			return errors.New("nested unit of work requires its transaction context")
		}
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, transactionKey{}, tx)
		return fn(newRepositories(tx.WithContext(txCtx), txCtx))
	})
	if err != nil {
		return err
	}
	if nested {
		parent.Merge(collector)
	} else if events := collector.Events(); len(events) > 0 && !u.queue.Enqueue(events) {
		// A queue failure cannot undo a committed business transaction.
		u.logger.Error("audit enqueue failed after transaction commit", "transaction_id", txID, "events", len(events))
	}
	return nil
}

type gormRepositories struct {
	userRepository userrepos.UserRepository
	ctx            context.Context
}

func (g *gormRepositories) User() userrepos.UserRepository { return g.userRepository }
func (g *gormRepositories) Context() context.Context       { return g.ctx }

func newRepositories(tx *gorm.DB, ctx context.Context) Repositories {
	return &gormRepositories{
		userRepository: userrepos.NewGormUserRepository(tx),
		ctx:            ctx,
	}
}

// Shutdown prevents new top-level transactions and waits for active producers.
func (u *gormUnitOfWork) Shutdown(ctx context.Context) error {
	u.mu.Lock()
	if !u.stopping {
		u.stopping = true
		if u.active == 0 {
			close(u.drained)
		}
	}
	u.mu.Unlock()
	select {
	case <-u.drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
