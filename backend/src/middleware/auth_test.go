// Package middleware_test はmiddlewareパッケージのテストを提供する。
package middleware_test

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/n-yata/money-management/backend/src/middleware"
)

func TestGetAuthSubWithLocalFallback(t *testing.T) {
	emptyReq := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{},
		},
	}

	t.Run("ENVIRONMENT=local のとき LOCAL_AUTH0_SUB にフォールバックする", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "local")
		t.Setenv("LOCAL_AUTH0_SUB", "auth0|localuser")

		gotSub, gotOK := middleware.GetAuthSub(emptyReq)

		if !gotOK {
			t.Error("GetAuthSub() ok = false, want true")
		}
		if gotSub != "auth0|localuser" {
			t.Errorf("GetAuthSub() sub = %q, want %q", gotSub, "auth0|localuser")
		}
	})

	t.Run("ENVIRONMENT=production のとき LOCAL_AUTH0_SUB があっても無視する", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("LOCAL_AUTH0_SUB", "auth0|localuser")

		gotSub, gotOK := middleware.GetAuthSub(emptyReq)

		if gotOK {
			t.Error("GetAuthSub() ok = true, want false（本番でのバイパス禁止）")
		}
		if gotSub != "" {
			t.Errorf("GetAuthSub() sub = %q, want %q", gotSub, "")
		}
	})

	t.Run("ENVIRONMENT未設定（デフォルト）のとき LOCAL_AUTH0_SUB を無視する", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "")
		t.Setenv("LOCAL_AUTH0_SUB", "auth0|localuser")

		gotSub, gotOK := middleware.GetAuthSub(emptyReq)

		if gotOK {
			t.Error("GetAuthSub() ok = true, want false（デフォルトはproduction扱い）")
		}
		if gotSub != "" {
			t.Errorf("GetAuthSub() sub = %q, want %q", gotSub, "")
		}
	})
}

func TestGetAuthSub(t *testing.T) {
	tests := []struct {
		name       string
		authorizer map[string]interface{}
		wantSub    string
		wantOK     bool
	}{
		{
			name: "subが正常に取得できる",
			authorizer: map[string]interface{}{
				"sub": "auth0|abc123",
			},
			wantSub: "auth0|abc123",
			wantOK:  true,
		},
		{
			name:       "Authorizerが空のマップ",
			authorizer: map[string]interface{}{},
			wantSub:    "",
			wantOK:     false,
		},
		{
			name: "subが空文字列",
			authorizer: map[string]interface{}{
				"sub": "",
			},
			wantSub: "",
			wantOK:  false,
		},
		{
			name: "subが文字列以外の型（int）",
			authorizer: map[string]interface{}{
				"sub": 12345,
			},
			wantSub: "",
			wantOK:  false,
		},
		{
			name: "subが文字列以外の型（bool）",
			authorizer: map[string]interface{}{
				"sub": true,
			},
			wantSub: "",
			wantOK:  false,
		},
		{
			name: "subが文字列以外の型（nil）",
			authorizer: map[string]interface{}{
				"sub": nil,
			},
			wantSub: "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: tt.authorizer,
				},
			}

			gotSub, gotOK := middleware.GetAuthSub(req)

			if gotOK != tt.wantOK {
				t.Errorf("GetAuthSub() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotSub != tt.wantSub {
				t.Errorf("GetAuthSub() sub = %q, want %q", gotSub, tt.wantSub)
			}
		})
	}
}
