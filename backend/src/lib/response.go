package lib

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

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
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

// ErrorResponse はエラーレスポンスを生成する。
func ErrorResponse(status int, code, message string) events.APIGatewayProxyResponse {
	return JSONResponse(status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
