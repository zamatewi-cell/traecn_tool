package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuthMiddleware_RejectEmptyOrWhitespaceKeys(t *testing.T) {
	// 传入包含空串、纯空格以及有效 key 的切片
	mw := NewAPIKeyAuthMiddleware([]string{"", "   ", "valid_key_123"})

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	handler := mw.Middleware(dummyHandler)

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "公开路径跳过鉴权",
			path:       "/health",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "缺少 Authorization 头返回 401",
			path:       "/v1/chat/completions",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "空 Bearer 返回 401",
			path:       "/v1/chat/completions",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "纯空格 Bearer 返回 401",
			path:       "/v1/chat/completions",
			authHeader: "Bearer    ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "错误 Key 返回 401",
			path:       "/v1/chat/completions",
			authHeader: "Bearer wrong_key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "有效 Key 正常通过返回 200",
			path:       "/v1/chat/completions",
			authHeader: "Bearer valid_key_123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "x-api-key 携带有效 Key 正常通过",
			path:       "/v1/chat/completions",
			authHeader: "valid_key_123",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
