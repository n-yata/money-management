package lib_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/n-yata/money-management/backend/src/lib"
)

func TestJSONResponse(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		body             any
		corsOrigin       string
		wantStatus       int
		wantBodyContains string
		wantOrigin       string
	}{
		{
			name:             "CORS_ALLOW_ORIGINが未設定の場合は空文字を返す",
			status:           http.StatusOK,
			body:             map[string]string{"key": "value"},
			corsOrigin:       "",
			wantStatus:       http.StatusOK,
			wantBodyContains: `"key":"value"`,
			wantOrigin:       "",
		},
		{
			name:             "CORS_ALLOW_ORIGINが設定されている場合はそれを使う",
			status:           http.StatusCreated,
			body:             map[string]string{"id": "123"},
			corsOrigin:       "https://example.com",
			wantStatus:       http.StatusCreated,
			wantBodyContains: `"id":"123"`,
			wantOrigin:       "https://example.com",
		},
		{
			name:             "SAM形式のシングルクォートを除去して返す",
			status:           http.StatusOK,
			body:             map[string]any{"ok": true},
			corsOrigin:       "'https://example.com'",
			wantStatus:       http.StatusOK,
			wantBodyContains: `"ok":true`,
			wantOrigin:       "https://example.com",
		},
		{
			name:             "シリアライズ不可能なbodyは500を返す",
			status:           http.StatusOK,
			body:             make(chan int), // json.Marshal 不可
			corsOrigin:       "",
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: "INTERNAL_ERROR",
			wantOrigin:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CORS_ALLOW_ORIGIN", tt.corsOrigin)

			resp := lib.JSONResponse(tt.status, tt.body)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if resp.Headers["Content-Type"] != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", resp.Headers["Content-Type"])
			}
			if resp.Headers["Access-Control-Allow-Origin"] != tt.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q",
					resp.Headers["Access-Control-Allow-Origin"], tt.wantOrigin)
			}
			if !strings.Contains(resp.Body, tt.wantBodyContains) {
				t.Errorf("Body = %q, want to contain %q", resp.Body, tt.wantBodyContains)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		message string
	}{
		{
			name:    "404 NOT_FOUNDレスポンスを返す",
			status:  http.StatusNotFound,
			code:    "NOT_FOUND",
			message: "見つかりません",
		},
		{
			name:    "400 VALIDATION_ERRORレスポンスを返す",
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "入力が不正です",
		},
		{
			name:    "401 UNAUTHORIZEDレスポンスを返す",
			status:  http.StatusUnauthorized,
			code:    "UNAUTHORIZED",
			message: "認証が必要です",
		},
		{
			name:    "500 INTERNAL_ERRORレスポンスを返す",
			status:  http.StatusInternalServerError,
			code:    "INTERNAL_ERROR",
			message: "サーバーエラーが発生しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := lib.ErrorResponse(tt.status, tt.code, tt.message)

			if resp.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.status)
			}
			if !strings.Contains(resp.Body, tt.code) {
				t.Errorf("Body = %q, want to contain code %q", resp.Body, tt.code)
			}
			if !strings.Contains(resp.Body, tt.message) {
				t.Errorf("Body = %q, want to contain message %q", resp.Body, tt.message)
			}
		})
	}
}
