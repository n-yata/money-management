// Package middleware はAuth0 JWT検証など認証関連の共通処理を提供する。
package middleware

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
)

// GetAuthSub はAPI GatewayのリクエストコンテキストからAuth0のsub IDを取得する。
// Lambda Authorizerが付与したコンテキストから取得するため、認証済みリクエストのみで有効。
// ENVIRONMENT=local のときのみ LOCAL_AUTH0_SUB へのフォールバックを許可する（明示的なオプトイン）。
// それ以外のすべての値（"production" や未設定）ではフォールバックを禁止する。
func GetAuthSub(request events.APIGatewayProxyRequest) (string, bool) {
	sub, ok := request.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		if os.Getenv("ENVIRONMENT") == "local" {
			if localSub := os.Getenv("LOCAL_AUTH0_SUB"); localSub != "" {
				return localSub, true
			}
		}
		return "", false
	}
	return sub, true
}
