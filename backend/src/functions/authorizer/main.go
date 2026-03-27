package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	// TODO: #3 Lambda Authorizer実装（Auth0 JWT検証）
	return events.APIGatewayCustomAuthorizerResponse{}, nil
}

func main() {
	lambda.Start(handler)
}
