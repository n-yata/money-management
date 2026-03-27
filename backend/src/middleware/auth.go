// Package middleware はAuth0 JWT検証など認証関連の共通処理を提供する。
package middleware

import (
	"github.com/aws/aws-lambda-go/events"
)

// GetAuthSub はAPI GatewayのリクエストコンテキストからAuth0のsub IDを取得する。
// Lambda Authorizerが付与したコンテキストから取得するため、認証済みリクエストのみで有効。
func GetAuthSub(request events.APIGatewayProxyRequest) (string, bool) {
	sub, ok := request.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		return "", false
	}
	return sub, true
}
