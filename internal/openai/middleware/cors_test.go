package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_NoAuth_UntrustedOriginBlocked(t *testing.T) {
	mw := NewCORSMiddleware(false)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for untrusted origin when no auth, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("did not expect Access-Control-Allow-Origin header on blocked request")
	}
}

func TestCORSMiddleware_NoAuth_TrustedOrigins(t *testing.T) {
	mw := NewCORSMiddleware(false)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	trusted := []string{
		"http://localhost:8080",
		"http://127.0.0.1:3000",
		"http://[::1]:9090",
		"vscode-webview://webview-panel-id",
		"app://obsidian.md",
		"file:///local/path",
		"electron://app",
	}

	for _, origin := range trusted {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for trusted origin %s, got %d", origin, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("expected Access-Control-Allow-Origin %s, got %s", origin, got)
		}
	}
}

func TestCORSMiddleware_WithAuth_AllowsAnyOrigin(t *testing.T) {
	mw := NewCORSMiddleware(true)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://external-web-client.org")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when auth is enabled, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://external-web-client.org" {
		t.Fatalf("expected reflected origin, got %s", got)
	}
}

func TestCORSMiddleware_PreflightOPTIONS(t *testing.T) {
	mw := NewCORSMiddleware(false)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS preflight")
	}))

	req := httptest.NewRequest("OPTIONS", "/v1/chat/completions", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("missing Access-Control-Allow-Methods")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("missing Access-Control-Allow-Headers")
	}
}

func TestCORSMiddleware_NoOriginHeader_AllowsCLIAndTests(t *testing.T) {
	mw := NewCORSMiddleware(false)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for CLI without origin, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected * for CLI without origin, got %s", got)
	}
}
