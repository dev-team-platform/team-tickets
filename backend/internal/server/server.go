package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/common/uow"
	"github.com/trannhanlv2004/team-tickets/internal/middleware"
)

// New supplies the shared unit of work to route construction.
func New(config *config.AppConfig, unit uow.UnitOfWork) (*http.Server, error) {
	if unit == nil {
		return nil, fmt.Errorf("unit of work is required")
	}
	auth, err := middleware.NewAuthentication(config.Auth)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	protected := http.NewServeMux()
	public := http.NewServeMux()
	registerV1(public, protected, unit)
	public.Handle("/", auth.Require(protected))
	return &http.Server{
		Addr:              ":" + strconv.Itoa(config.Port),
		Handler:           middleware.AuditMetadata(middleware.ErrorHandler(public)),
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
