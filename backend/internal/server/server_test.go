package server

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

	"github.com/trannhanlv2004/team-tickets/internal/configs"
)

func TestPublicHealthAndProtectedRoutes(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)})
	if err := os.WriteFile(keyPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	app, err := New(&configs.AppConfig{Port: 8000, Auth: configs.AuthConfig{InternalJWTIssuer: "issuer", InternalJWTAudience: "audience", InternalJWTPublicKeyPEMPath: keyPath}})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	app.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil))
	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected health response: status=%d body=%q", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	app.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized protected route, got %d", response.Code)
	}
}

func TestNewRejectsInvalidAuthenticationConfiguration(t *testing.T) {
	if _, err := New(&configs.AppConfig{}); err == nil {
		t.Fatal("expected missing authentication configuration to fail")
	}
}
