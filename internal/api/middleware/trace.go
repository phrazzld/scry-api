package middleware

import (
	"log/slog"
	"net/http"

	"github.com/phrazzld/scry-api/internal/api/shared"
)

// TraceMiddleware adds a trace ID to the request context.
// This middleware should be applied early in the middleware chain to ensure
// that all subsequent handlers have access to the trace ID.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add a trace ID to the context
		ctx := shared.SetTraceID(r.Context())

		// Get the trace ID for logging
		traceID := shared.GetTraceID(ctx)

		// Add trace ID to the logger context
		log := slog.With(slog.String("trace_id", traceID))

		// Log the incoming request with trace ID
		log.Debug("request started",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr))

		// Continue with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
