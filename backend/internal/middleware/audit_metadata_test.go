package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
)

func TestAuditMetadataPreservesCorrelation(t *testing.T) {
	var got auditlogservices.Metadata
	handler := AuditMetadata(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = auditlogservices.MetadataFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "request")
	req.Header.Set("X-Client-Action-Id", "action")
	req.Header.Set("X-Client-Request-Id", "client")
	req.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.RequestID != "request" || got.ClientActionID != "action" || got.ClientRequestID != "client" || got.TraceID != "0123456789abcdef0123456789abcdef" || got.SpanID != "" || got.ActorID != "" {
		t.Fatalf("unexpected metadata: %+v", got)
	}
}
