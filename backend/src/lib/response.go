package lib

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// allowOrigin は CORS_ALLOW_ORIGIN 環境変数からオリジンを取得する。
// SAM テンプレートは API Gateway 用にシングルクォートで値を渡す（例: "'*'"）ため、
// 環境変数経由で受け取った場合はクォートを除去する。
func allowOrigin() string {
	origin := os.Getenv("CORS_ALLOW_ORIGIN")
	if origin == "" {
		log.Printf("WARNING: CORS_ALLOW_ORIGIN is not set.")
		return ""
	}
	return strings.Trim(origin, "'")
}

// JSONResponse は JSON レスポンスを生成する。
// json.Marshal に失敗した場合は 500 エラーレスポンスを返す。
func JSONResponse(status int, body any) events.APIGatewayProxyResponse {
	b, err := json.Marshal(body)
	if err != nil {
		log.Printf("json.Marshal error: %v", err)
		b = []byte(`{"error":{"code":"INTERNAL_ERROR","message":"レスポンスの生成に失敗しました"}}`)
		status = http.StatusInternalServerError
	}
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": allowOrigin(),
		},
		Body: string(b),
	}
}

// ErrorResponse はエラーレスポンスを生成する。
func ErrorResponse(status int, code, message string) events.APIGatewayProxyResponse {
	return JSONResponse(status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
