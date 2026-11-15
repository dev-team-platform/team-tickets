package middleware

import (
	"encoding/hex"
	"net/http"
	"strings"

	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
)

// AuditMetadata preserves request correlation information for service operations.
// The service must supply the actor separately after business validation.
func AuditMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := auditlogservices.MetadataFromContext(r.Context())
		metadata.RequestID = r.Header.Get("X-Request-Id")
		metadata.ClientActionID = r.Header.Get("X-Client-Action-Id")
		metadata.ClientRequestID = r.Header.Get("X-Client-Request-Id")
		parts := strings.Split(r.Header.Get("traceparent"), "-")
		if len(parts) == 4 && len(parts[1]) == 32 && len(parts[2]) == 16 {
			_, traceErr := hex.DecodeString(parts[1])
			_, spanErr := hex.DecodeString(parts[2])
			if traceErr == nil && spanErr == nil {
				if metadata.TraceID == "" {
					metadata.TraceID = parts[1]
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(auditlogservices.WithMetadata(r.Context(), metadata)))
	})
}
