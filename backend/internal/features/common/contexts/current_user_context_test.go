package contexts

import (
	"context"
	"errors"
	"net/http"
	"testing"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
)

func TestIdentitySubjectReturnsAuthenticatedSubject(t *testing.T) {
	ctx := WithIdentity(context.Background(), Identity{Subject: "user-42"})
	subject, err := (CurrentUserContext{}).IdentitySubject(ctx)
	if err != nil || subject != "user-42" {
		t.Fatalf("subject = %q, error = %v", subject, err)
	}
}

func TestIdentitySubjectRejectsMissingSubject(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"no identity":   context.Background(),
		"empty subject": WithIdentity(context.Background(), Identity{}),
		"blank subject": WithIdentity(context.Background(), Identity{Subject: "  "}),
	} {
		t.Run(name, func(t *testing.T) {
			subject, err := (CurrentUserContext{}).IdentitySubject(ctx)
			var appErr *appErrors.Error
			if subject != "" || !errors.As(err, &appErr) || appErr.StatusCode != http.StatusUnauthorized || appErr.Message != "User is not authenticated." {
				t.Fatalf("subject = %q, error = %v", subject, err)
			}
		})
	}
}
