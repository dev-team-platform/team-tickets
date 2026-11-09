package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/configs"
	"github.com/trannhanlv2004/team-tickets/internal/middlewares"
)

func New(config *configs.AppConfig) (*http.Server, error) {
	auth, err := middlewares.NewAuthentication(config.Auth)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	protected := http.NewServeMux()
	public := http.NewServeMux()
	registerV1(public, protected)
	public.Handle("/", auth.Require(protected))
	return &http.Server{Addr: ":" + strconv.Itoa(config.Port), Handler: middlewares.ErrorHandler(public), ReadHeaderTimeout: 5 * time.Second}, nil
}
