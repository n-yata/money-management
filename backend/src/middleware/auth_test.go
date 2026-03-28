// Package middleware_test はmiddlewareパッケージのテストを提供する。
package middleware_test

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/n-yata/money-management/backend/src/middleware"
)

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
