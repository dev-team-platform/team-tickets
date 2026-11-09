// Package contexts provides request-scoped application context values.
package contexts

import (
	"context"
	"strings"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
)

// Identity contains claims from a verified authentication token.
type Identity struct {
	Subject           string
	PreferredUsername string
	Roles             []string
}

type identityKey struct{}

// WithIdentity attaches an identity established by trusted authentication middleware.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

// IdentityFromContext returns the identity attached to the request, if present.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// CurrentUserContext reads the authenticated user from an operation's context.
type CurrentUserContext struct{}

// IdentitySubject returns the token's subject or an unauthorized application error.
func (CurrentUserContext) IdentitySubject(ctx context.Context) (string, error) {
	identity, ok := IdentityFromContext(ctx)
	if !ok || strings.TrimSpace(identity.Subject) == "" {
		return "", appErrors.NewUnauthorized("User is not authenticated.", nil)
	}
	return identity.Subject, nil
}
