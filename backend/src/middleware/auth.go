// Package middleware はAuth0 JWT検証など認証関連の共通処理を提供する。
package middleware

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
)

// GetAuthSub はAPI GatewayのリクエストコンテキストからAuth0のsub IDを取得する。
// Lambda Authorizerが付与したコンテキストから取得するため、認証済みリクエストのみで有効。
// ローカル開発時（--disable-authorizer使用時）は LOCAL_AUTH0_SUB 環境変数にフォールバックする。
func GetAuthSub(request events.APIGatewayProxyRequest) (string, bool) {
	sub, ok := request.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		if localSub := os.Getenv("LOCAL_AUTH0_SUB"); localSub != "" {
			return localSub, true
		}
		return "", false
	}
	return sub, true
}
