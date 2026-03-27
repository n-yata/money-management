package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// TODO: #5 おこづかい種類API実装（CRUD）
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: `{"data":[]}`}, nil
}

func main() {
	lambda.Start(handler)
}
