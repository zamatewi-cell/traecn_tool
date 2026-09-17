package middleware

import (
	"net/http"
	"strings"
)

// APIKeyAuthMiddleware authenticates requests using API keys
type APIKeyAuthMiddleware struct {
	validKeys map[string]bool
}

// NewAPIKeyAuthMiddleware creates a new API key auth middleware
func NewAPIKeyAuthMiddleware(apiKeys []string) *APIKeyAuthMiddleware {
	keys := make(map[string]bool)
	for _, key := range apiKeys {
		keys[key] = true
	}
	return &APIKeyAuthMiddleware{
		validKeys: keys,
	}
}

// Middleware wraps the handler with API key authentication
func (m *APIKeyAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health check and root endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key in Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": {"message": "Missing Authorization header", "type": "authentication_error"}}`, http.StatusUnauthorized)
			return
		}

		// Support "Bearer <key>" format
		var apiKey string
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			apiKey = authHeader
		}

		// Check if API key is valid
		if !m.validKeys[apiKey] {
			http.Error(w, `{"error": {"message": "Invalid API key", "type": "authentication_error"}}`, http.StatusUnauthorized)
			return
		}

		// API key is valid, proceed
		next.ServeHTTP(w, r)
	})
}
