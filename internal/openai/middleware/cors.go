package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// CORSMiddleware handles CORS headers and local loopback origin protection.
type CORSMiddleware struct {
	hasAuth        bool
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(hasAuth bool) *CORSMiddleware {
	return &CORSMiddleware{
		hasAuth:        hasAuth,
		allowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		allowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With", "X-API-Key", "x-api-key"},
	}
}

// isLoopbackOrigin checks if the origin belongs to local loopback or safe desktop extensions
func isLoopbackOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	lower := strings.ToLower(origin)
	if strings.HasPrefix(lower, "vscode-webview://") ||
		strings.HasPrefix(lower, "app://") ||
		strings.HasPrefix(lower, "file://") ||
		strings.HasPrefix(lower, "electron://") {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := u.Hostname()
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Middleware wraps the handler with CORS support and cross-site protection
func (m *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Security: When server runs without API Key authentication, block cross-origin
		// requests from untrusted external web origins to prevent localhost CSRF / drive-by attacks.
		if !m.hasAuth && origin != "" && !isLoopbackOrigin(origin) {
			http.Error(w, `{"error":{"message":"Cross-origin request from untrusted origin blocked when API Key authentication is disabled","type":"security_error"}}`, http.StatusForbidden)
			return
		}

		// Apply CORS response headers
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			// No origin header (e.g. non-browser CLI, tests, or direct requests)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", joinStrings(m.allowedMethods))
		w.Header().Set("Access-Control-Allow-Headers", joinStrings(m.allowedHeaders))

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// joinStrings joins a slice of strings with commas
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += ", " + s
	}
	return result
}
