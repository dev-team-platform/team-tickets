package middlewares

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/trannhanlv2004/team-tickets/internal/configs"
	"github.com/trannhanlv2004/team-tickets/internal/features/common/contexts"
)

func authenticationFixture(t *testing.T) (*Authentication, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustMarshalPublicKey(t, &key.PublicKey)})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	auth, err := NewAuthentication(configs.AuthConfig{InternalJWTIssuer: "issuer", InternalJWTAudience: "audience", InternalJWTPublicKeyPEMPath: path})
	if err != nil {
		t.Fatal(err)
	}
	return auth, key
}

func mustMarshalPublicKey(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	data, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func signToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAuthenticationValidatesBearerTokens(t *testing.T) {
	auth, key := authenticationFixture(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	valid := func() jwt.MapClaims {
		return jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "user-1", "preferred_username": "alex", "roles": []string{"admin"}, "exp": now.Add(time.Hour).Unix()}
	}
	type testCase struct {
		name   string
		header string
		want   int
	}
	cases := []testCase{{"missing", "", 401}, {"malformed", "Bearer malformed", 401}, {"wrong scheme", "Basic abc", 401}}
	add := func(name string, claims jwt.MapClaims, method jwt.SigningMethod, signingKey any, want int) {
		cases = append(cases, testCase{name, "Bearer " + signToken(t, method, signingKey, claims), want})
	}
	add("valid", valid(), jwt.SigningMethodRS256, key, 200)
	add("bad signature", valid(), jwt.SigningMethodRS256, otherKey, 401)
	add("wrong algorithm", valid(), jwt.SigningMethodHS256, []byte("secret"), 401)
	claims := valid()
	claims["iss"] = "other"
	add("wrong issuer", claims, jwt.SigningMethodRS256, key, 401)
	claims = valid()
	claims["aud"] = "other"
	add("wrong audience", claims, jwt.SigningMethodRS256, key, 401)
	claims = valid()
	delete(claims, "exp")
	add("missing expiration", claims, jwt.SigningMethodRS256, key, 401)
	claims = valid()
	claims["exp"] = now.Add(-2 * time.Minute).Unix()
	add("expired", claims, jwt.SigningMethodRS256, key, 401)
	claims = valid()
	claims["exp"] = now.Add(-30 * time.Second).Unix()
	add("clock skew", claims, jwt.SigningMethodRS256, key, 200)
	claims = valid()
	claims["nbf"] = now.Add(2 * time.Minute).Unix()
	add("not yet valid", claims, jwt.SigningMethodRS256, key, 401)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				subject, subjectErr := (contexts.CurrentUserContext{}).IdentitySubject(r.Context())
				if subjectErr != nil || subject != "user-1" {
					t.Errorf("unexpected subject: %q, error: %v", subject, subjectErr)
				}
				identity, ok := IdentityFromContext(r.Context())
				if !ok || identity.Subject != "user-1" || identity.PreferredUsername != "alex" || len(identity.Roles) != 1 || identity.Roles[0] != "admin" {
					t.Errorf("unexpected identity: %+v", identity)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tt.want || called != (tt.want == 200) {
				t.Fatalf("status %d, called %v", response.Code, called)
			}
			if tt.want == 401 && response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Error("missing bearer challenge")
			}
		})
	}
}

func TestNewAuthenticationRejectsInvalidKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(path, []byte("not a key"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := NewAuthentication(configs.AuthConfig{InternalJWTIssuer: "issuer", InternalJWTAudience: "audience", InternalJWTPublicKeyPEMPath: path})
	if err == nil {
		t.Fatal("expected invalid public key to fail")
	}
}
