package middlewares

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/trannhanlv2004/team-tickets/internal/configs"
	"github.com/trannhanlv2004/team-tickets/internal/features/common/contexts"
	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
)

// Identity contains verified token claims available to protected handlers.
type Identity = contexts.Identity

// IdentityFromContext returns the identity established by Authentication.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	return contexts.IdentityFromContext(ctx)
}

type internalClaims struct {
	PreferredUsername string           `json:"preferred_username"`
	Roles             jwt.ClaimStrings `json:"roles"`
	jwt.RegisteredClaims
}

// Authentication verifies internal JWT bearer tokens for protected routes.
type Authentication struct {
	key    *rsa.PublicKey
	parser *jwt.Parser
}

func NewAuthentication(cfg configs.AuthConfig) (*Authentication, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	keyBytes, err := os.ReadFile(cfg.InternalJWTPublicKeyPEMPath)
	if err != nil {
		return nil, fmt.Errorf("read internal JWT public key: %w", err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse internal JWT public key: %w", err)
	}
	return &Authentication{key: key, parser: jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(cfg.InternalJWTIssuer), jwt.WithAudience(cfg.InternalJWTAudience),
		jwt.WithExpirationRequired(), jwt.WithLeeway(time.Minute),
	)}, nil
}

func (a *Authentication) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := r.Header.Values("Authorization")
		if len(headers) != 1 {
			unauthorized(w)
			return
		}
		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(w)
			return
		}
		claims := &internalClaims{}
		token, err := a.parser.ParseWithClaims(parts[1], claims, func(*jwt.Token) (any, error) { return a.key, nil })
		if err != nil || !token.Valid {
			unauthorized(w)
			return
		}
		identity := Identity{
			Subject:           claims.Subject,
			PreferredUsername: claims.PreferredUsername,
			Roles:             append([]string(nil), claims.Roles...),
		}
		next.ServeHTTP(w, r.WithContext(contexts.WithIdentity(r.Context(), identity)))
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(appErrors.ErrorResponse{Message: "Unauthorized."})
}
