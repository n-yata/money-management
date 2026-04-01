package main

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	gojwt "github.com/golang-jwt/jwt/v5"
)

// jwksCache はJWKSのインメモリキャッシュ。
type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

var cache jwksCache

const cacheTTL = time.Hour

// httpClient はJWKSエンドポイント取得用HTTPクライアント。
// タイムアウトを設定することで、Auth0が無応答の場合にLambdaがブロックされるのを防ぐ。
var httpClient = &http.Client{Timeout: 5 * time.Second}

// jwks はAuth0のJWKSエンドポイントのレスポンス形式。
type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// getPublicKey はkidに対応するRSA公開鍵を返す。キャッシュが有効な場合はキャッシュを使用する。
func getPublicKey(kid string) (*rsa.PublicKey, error) {
	cache.mu.RLock()
	if time.Since(cache.fetchedAt) < cacheTTL {
		if key, ok := cache.keys[kid]; ok {
			cache.mu.RUnlock()
			return key, nil
		}
	}
	cache.mu.RUnlock()

	// キャッシュミスまたはTTL切れ → JWKSを再取得
	domain := os.Getenv("AUTH0_DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("AUTH0_DOMAIN is not set")
	}

	resp, err := httpClient.Get(fmt.Sprintf("https://%s/.well-known/jwks.json", domain))
	if err != nil {
		log.Printf("failed to fetch JWKS: %v", err)
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwksData jwks
	if err := json.NewDecoder(resp.Body).Decode(&jwksData); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwksData.Keys))
	for _, k := range jwksData.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := buildRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		newKeys[k.Kid] = pub
	}

	cache.mu.Lock()
	cache.keys = newKeys
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()

	key, ok := newKeys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found for kid: %s", kid)
	}
	return key, nil
}

// buildRSAPublicKey はJWKのn/eからRSA公開鍵を構築する。
func buildRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

// generatePolicy はIAM認可ポリシーを生成する。
func generatePolicy(sub, effect, resource string) events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: sub,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
		Context: map[string]interface{}{
			"sub": sub,
		},
	}
}

func handler(_ context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		authHeader = request.Headers["authorization"]
	}
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		log.Printf("auth header missing or invalid")
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("Unauthorized")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	domain := os.Getenv("AUTH0_DOMAIN")
	audience := os.Getenv("AUTH0_AUDIENCE")
	if domain == "" || audience == "" {
		log.Printf("env vars missing: domain=%q audience=%q", domain, audience)
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("Unauthorized")
	}

	token, err := gojwt.Parse(tokenStr,
		func(t *gojwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			kid, ok := t.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("kid not found in token header")
			}
			return getPublicKey(kid)
		},
		gojwt.WithIssuer(fmt.Sprintf("https://%s/", domain)),
		gojwt.WithAudience(audience),
		gojwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil || !token.Valid {
		log.Printf("token validation failed: %v", err)
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("Unauthorized")
	}

	claims, ok := token.Claims.(gojwt.MapClaims)
	if !ok {
		log.Printf("failed to parse claims")
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("Unauthorized")
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		log.Printf("sub claim missing or empty")
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("Unauthorized")
	}

	// MethodArnからAPI/ステージ部分を抽出し、同一API内の全リソースのみ許可する
	// 例: arn:aws:execute-api:region:account:apiId/stage/GET/path → apiId/stage/*/*
	parts := strings.Split(request.MethodArn, "/")
	var resource string
	if len(parts) >= 2 {
		resource = strings.Join(parts[:2], "/") + "/*/*"
	} else {
		resource = request.MethodArn
	}
	return generatePolicy(sub, "Allow", resource), nil
}

func main() {
	lambda.Start(handler)
}
