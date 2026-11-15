package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/client/database"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/auditlog"
	"github.com/trannhanlv2004/team-tickets/internal/feature/common/uow"
	"github.com/trannhanlv2004/team-tickets/internal/feature/export"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging"
	"github.com/trannhanlv2004/team-tickets/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return err
	}
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := database.ConnectPostgres(connectCtx, cfg.DB)
	cancelConnect()
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	auditQueue, err := auditlog.Bootstrap(db, cfg.AuditLog, slog.Default())
	if err != nil {
		return err
	}
	exportQueue, err := export.Bootstrap(cfg.Export, slog.Default())
	if err != nil {
		_ = auditQueue.Shutdown(context.Background())
		return err
	}

	unit, err := uow.New(db, auditQueue, slog.Default())
	if err != nil {
		_ = exportQueue.Shutdown(context.Background())
		_ = auditQueue.Shutdown(context.Background())
		return err
	}

	app, err := server.New(cfg, unit)
	if err != nil {
		_ = exportQueue.Shutdown(context.Background())
		_ = auditQueue.Shutdown(context.Background())
		return err
	}

	messagingRuntime, err := messaging.Bootstrap(cfg.RabbitMQ, nil, slog.Default())
	if err != nil {
		_ = exportQueue.Shutdown(context.Background())
		_ = auditQueue.Shutdown(context.Background())
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("API listening", "address", app.Addr)
		if err := app.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	var serveErr error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case serveErr = <-serverErr:
		slog.Error("HTTP server failed", "error", serveErr)
	}
	stop()
	httpCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
	if err := app.Shutdown(httpCtx); err != nil {
		slog.Error("HTTP shutdown failed", "error", err)
		_ = app.Close()
	}
	cancelHTTP()
	messagingCtx, cancelMessaging := context.WithTimeout(context.Background(), time.Duration(cfg.RabbitMQ.ShutdownSeconds)*time.Second)
	if err := messagingRuntime.StopConsumers(messagingCtx); err != nil {
		slog.Error("messaging consumers shutdown failed", "error", err)
	}
	cancelMessaging()
	exportCtx, cancelExport := context.WithTimeout(context.Background(), 10*time.Second)
	if err := exportQueue.Shutdown(exportCtx); err != nil {
		slog.Error("export shutdown failed", "error", err)
	}
	cancelExport()
	auditCtx, cancelAudit := context.WithTimeout(context.Background(), time.Duration(cfg.AuditLog.ShutdownTimeoutSeconds)*time.Second)
	defer cancelAudit()
	if err := unit.Shutdown(auditCtx); err != nil {
		slog.Error("unit of work shutdown timed out", "error", err)
	}
	if err := auditQueue.Shutdown(auditCtx); err != nil {
		slog.Error("audit shutdown failed", "error", err)
	}
	finishMessagingCtx, finishMessaging := context.WithTimeout(context.Background(), time.Duration(cfg.RabbitMQ.ShutdownSeconds)*time.Second)
	if err := messagingRuntime.Shutdown(finishMessagingCtx); err != nil {
		slog.Error("messaging shutdown failed", "error", err)
	}
	finishMessaging()
	if serveErr != nil {
		return fmt.Errorf("HTTP server: %w", serveErr)
	}
	return nil
}
