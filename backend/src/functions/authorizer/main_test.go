package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	gojwt "github.com/golang-jwt/jwt/v5"
)

// ─── テストヘルパー ──────────────────────────────────────────

// generateTestKeyPair はテスト用RSA鍵ペアを生成する。
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("RSA鍵ペア生成失敗: %v", err)
	}
	return priv, &priv.PublicKey
}

// makeTestJWT はテスト用JWTを生成する。
func makeTestJWT(t *testing.T, privKey *rsa.PrivateKey, kid, issuer, audience, sub string, expiry time.Time) string {
	t.Helper()
	claims := gojwt.MapClaims{
		"sub": sub,
		"iss": issuer,
		"aud": []string{audience},
		"exp": expiry.Unix(),
		"iat": time.Now().Unix(),
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("JWT署名失敗: %v", err)
	}
	return signed
}

// setTestCache はテスト用にキャッシュを直接設定し、ネットワークアクセスを回避する。
func setTestCache(key *rsa.PublicKey, kid string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.keys = map[string]*rsa.PublicKey{kid: key}
	cache.fetchedAt = time.Now()
}

// resetCache はキャッシュをゼロ値にリセットする。
func resetCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.keys = nil
	cache.fetchedAt = time.Time{}
}

// ─── buildRSAPublicKey のテスト ──────────────────────────────

func TestBuildRSAPublicKey(t *testing.T) {
	// 有効なRSAキーのn/eを用意する
	_, pub := generateTestKeyPair(t)
	validN := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := new(big.Int).SetInt64(int64(pub.E)).Bytes()
	validE := base64.RawURLEncoding.EncodeToString(eBytes)

	tests := []struct {
		name    string
		nStr    string
		eStr    string
		wantErr bool
	}{
		{
			name:    "有効なRSA公開鍵のn/eから正しく構築される",
			nStr:    validN,
			eStr:    validE,
			wantErr: false,
		},
		{
			name:    "nのbase64が不正な場合はエラー",
			nStr:    "!!!invalid-base64!!!",
			eStr:    validE,
			wantErr: true,
		},
		{
			name:    "eのbase64が不正な場合はエラー",
			nStr:    validN,
			eStr:    "!!!invalid-base64!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildRSAPublicKey(tt.nStr, tt.eStr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("buildRSAPublicKey() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildRSAPublicKey() unexpected error = %v", err)
			}
			// 元のキーと同じ値であることを確認
			if got.N.Cmp(pub.N) != 0 {
				t.Errorf("PublicKey.N が一致しない")
			}
			if got.E != pub.E {
				t.Errorf("PublicKey.E = %d, want %d", got.E, pub.E)
			}
		})
	}
}

// ─── generatePolicy のテスト ─────────────────────────────────

func TestGeneratePolicy(t *testing.T) {
	tests := []struct {
		name           string
		sub            string
		effect         string
		resource       string
		wantEffect     string
		wantPrincipal  string
		wantContextSub string
	}{
		{
			name:           "Allowポリシーが正しく生成される",
			sub:            "auth0|user123",
			effect:         "Allow",
			resource:       "arn:aws:execute-api:ap-northeast-1:123:api/prod/*",
			wantEffect:     "Allow",
			wantPrincipal:  "auth0|user123",
			wantContextSub: "auth0|user123",
		},
		{
			name:           "Denyポリシーが正しく生成される",
			sub:            "auth0|user456",
			effect:         "Deny",
			resource:       "arn:aws:execute-api:*:*:*",
			wantEffect:     "Deny",
			wantPrincipal:  "auth0|user456",
			wantContextSub: "auth0|user456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generatePolicy(tt.sub, tt.effect, tt.resource)

			// PrincipalID の検証
			if got.PrincipalID != tt.wantPrincipal {
				t.Errorf("PrincipalID = %q, want %q", got.PrincipalID, tt.wantPrincipal)
			}

			// PolicyDocument のStatement検証
			if len(got.PolicyDocument.Statement) != 1 {
				t.Fatalf("Statement の件数 = %d, want 1", len(got.PolicyDocument.Statement))
			}
			stmt := got.PolicyDocument.Statement[0]
			if stmt.Effect != tt.wantEffect {
				t.Errorf("Statement.Effect = %q, want %q", stmt.Effect, tt.wantEffect)
			}
			if len(stmt.Action) != 1 || stmt.Action[0] != "execute-api:Invoke" {
				t.Errorf("Statement.Action = %v, want [execute-api:Invoke]", stmt.Action)
			}
			if len(stmt.Resource) != 1 || stmt.Resource[0] != tt.resource {
				t.Errorf("Statement.Resource = %v, want [%s]", stmt.Resource, tt.resource)
			}

			// Context["sub"] の検証
			contextSub, ok := got.Context["sub"]
			if !ok {
				t.Error("Context[\"sub\"] が存在しない")
			} else if contextSub != tt.wantContextSub {
				t.Errorf("Context[\"sub\"] = %v, want %q", contextSub, tt.wantContextSub)
			}
		})
	}
}

// ─── handler のテスト（テーブル駆動）────────────────────────

func TestHandler(t *testing.T) {
	const (
		testDomain   = "test.auth0.com"
		testAudience = "https://api.test.com"
		testIssuer   = "https://test.auth0.com/"
		testKid      = "test-key-id"
		testSub      = "auth0|test-user"
	)

	// テスト用鍵ペアを一度だけ生成する（各サブテストで共有）
	validPriv, validPub := generateTestKeyPair(t)
	// 署名不正テスト用に別の鍵ペアを生成する
	wrongPriv, _ := generateTestKeyPair(t)

	tests := []struct {
		name        string
		setupCache  func()           // キャッシュの事前設定
		setupEnv    func(t *testing.T) // 環境変数のセットアップ
		headers     map[string]string
		makeToken   func() string    // Authorizationヘッダー用のトークン生成
		wantErr     bool
		wantEffect  string           // エラーなし時に期待するポリシーのEffect
	}{
		{
			name: "正常: 有効なJWTでAllowポリシーが返る",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				return makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
			},
			wantErr:    false,
			wantEffect: "Allow",
		},
		{
			name: "Authorizationヘッダーなし → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken:  func() string { return "" }, // 空文字 = ヘッダーなし
			wantErr:    true,
		},
		{
			name: "Bearer プレフィックスなし → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				// "Bearer " なしの生トークン
				tok := makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
				return tok // Bearerプレフィックスなしでそのまま渡す
			},
			// makeToken の返り値をそのまま Authorization ヘッダーにセットするが、
			// "Bearer " プレフィックスがないため弾かれることを確認する。
			// headers フィールドで上書きする。
			wantErr: true,
		},
		{
			name: "環境変数未設定（AUTH0_DOMAIN なし） → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				// AUTH0_DOMAIN を設定しない
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				return makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
			},
			wantErr: true,
		},
		{
			name: "環境変数未設定（AUTH0_AUDIENCE なし） → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				// AUTH0_AUDIENCE を設定しない
			},
			makeToken: func() string {
				return makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
			},
			wantErr: true,
		},
		{
			name: "署名に使っていない鍵でのトークン（署名不正） → error",
			setupCache: func() {
				// 検証キャッシュには validPub を登録するが、トークンは wrongPriv で署名
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				// wrongPriv で署名したトークン（validPub での検証は失敗する）
				return makeTestJWT(t, wrongPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
			},
			wantErr: true,
		},
		{
			name: "期限切れトークン → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				// 1時間前に期限切れ
				return makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(-time.Hour))
			},
			wantErr: true,
		},
		{
			name: "subが空のトークン → error",
			setupCache: func() {
				setTestCache(validPub, testKid)
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH0_DOMAIN", testDomain)
				t.Setenv("AUTH0_AUDIENCE", testAudience)
			},
			makeToken: func() string {
				// subを空文字にする
				return makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, "", time.Now().Add(time.Hour))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト後にキャッシュをリセットして独立性を確保
			t.Cleanup(resetCache)

			// キャッシュの事前設定
			tt.setupCache()

			// 環境変数のセットアップ（t.Setenv はテスト終了後に自動リセット）
			tt.setupEnv(t)

			// トークン生成
			token := tt.makeToken()

			// Authorizationヘッダーの組み立て
			var authHeader string
			switch tt.name {
			case "Authorizationヘッダーなし → error":
				authHeader = "" // ヘッダーなし
			case "Bearer プレフィックスなし → error":
				authHeader = token // プレフィックスなしで渡す
			default:
				if token != "" {
					authHeader = "Bearer " + token
				}
			}

			req := events.APIGatewayCustomAuthorizerRequestTypeRequest{
				MethodArn: "arn:aws:execute-api:ap-northeast-1:123456789:testapi/prod/GET/api/v1/children",
				Headers: map[string]string{
					"Authorization": authHeader,
				},
			}

			resp, err := handler(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("handler() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("handler() unexpected error = %v", err)
			}

			// Allowポリシーの検証
			if len(resp.PolicyDocument.Statement) != 1 {
				t.Fatalf("Statement の件数 = %d, want 1", len(resp.PolicyDocument.Statement))
			}
			gotEffect := resp.PolicyDocument.Statement[0].Effect
			if gotEffect != tt.wantEffect {
				t.Errorf("Statement.Effect = %q, want %q", gotEffect, tt.wantEffect)
			}

			// subの検証
			if resp.PrincipalID != testSub {
				t.Errorf("PrincipalID = %q, want %q", resp.PrincipalID, testSub)
			}
			contextSub, ok := resp.Context["sub"]
			if !ok {
				t.Error("Context[\"sub\"] が存在しない")
			} else if contextSub != testSub {
				t.Errorf("Context[\"sub\"] = %v, want %q", contextSub, testSub)
			}
		})
	}
}

// ─── handler: 小文字 authorization ヘッダーのテスト ──────────

func TestHandlerLowercaseAuthHeader(t *testing.T) {
	const (
		testDomain   = "test.auth0.com"
		testAudience = "https://api.test.com"
		testIssuer   = "https://test.auth0.com/"
		testKid      = "test-key-id"
		testSub      = "auth0|test-user"
	)
	validPriv, validPub := generateTestKeyPair(t)
	setTestCache(validPub, testKid)
	t.Cleanup(resetCache)
	t.Setenv("AUTH0_DOMAIN", testDomain)
	t.Setenv("AUTH0_AUDIENCE", testAudience)

	token := makeTestJWT(t, validPriv, testKid, testIssuer, testAudience, testSub, time.Now().Add(time.Hour))
	req := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		MethodArn: "arn:aws:execute-api:ap-northeast-1:123456789:testapi/prod/GET/api/v1/children",
		Headers: map[string]string{
			"authorization": "Bearer " + token, // 小文字キー
		},
	}

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler() error = %v, want nil（小文字ヘッダーでも認証できる）", err)
	}
	if resp.PolicyDocument.Statement[0].Effect != "Allow" {
		t.Errorf("Effect = %q, want Allow", resp.PolicyDocument.Statement[0].Effect)
	}
}

// ─── getPublicKey のテスト ───────────────────────────────────

func TestGetPublicKey(t *testing.T) {
	t.Run("AUTH0_DOMAINが未設定の場合エラー", func(t *testing.T) {
		resetCache()
		t.Cleanup(resetCache)
		t.Setenv("AUTH0_DOMAIN", "")

		_, err := getPublicKey("any-kid")
		if err == nil {
			t.Error("getPublicKey() error = nil, want error（AUTH0_DOMAIN未設定）")
		}
	})

	t.Run("JWKSエンドポイントからkidが見つかった場合に公開鍵を返す", func(t *testing.T) {
		resetCache()
		t.Cleanup(resetCache)

		// テスト用RSA鍵ペアを生成してJWKS形式に変換
		_, pub := generateTestKeyPair(t)
		nStr := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
		eBytes := new(big.Int).SetInt64(int64(pub.E)).Bytes()
		eStr := base64.RawURLEncoding.EncodeToString(eBytes)
		const kid = "jwks-test-kid"

		// TLSテストサーバーでJWKSエンドポイントをモック
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(jwks{
				Keys: []jwk{{Kid: kid, Kty: "RSA", N: nStr, E: eStr}},
			})
		}))
		defer server.Close()

		// httpClient.Transport を一時的にテストサーバー用に差し替え（C-5対応: httpClient使用）
		origTransport := httpClient.Transport
		t.Cleanup(func() { httpClient.Transport = origTransport })
		httpClient.Transport = server.Client().Transport

		host := strings.TrimPrefix(server.URL, "https://")
		t.Setenv("AUTH0_DOMAIN", host)

		got, err := getPublicKey(kid)
		if err != nil {
			t.Fatalf("getPublicKey() error = %v", err)
		}
		if got.N.Cmp(pub.N) != 0 {
			t.Error("返された公開鍵のNが一致しない")
		}
	})

	t.Run("JWKSエンドポイントにkidが存在しない場合エラー", func(t *testing.T) {
		resetCache()
		t.Cleanup(resetCache)

		// kidが空のJWKSを返すサーバー
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(jwks{Keys: []jwk{}})
		}))
		defer server.Close()

		origTransport := httpClient.Transport
		t.Cleanup(func() { httpClient.Transport = origTransport })
		httpClient.Transport = server.Client().Transport

		host := strings.TrimPrefix(server.URL, "https://")
		t.Setenv("AUTH0_DOMAIN", host)

		_, err := getPublicKey("non-existent-kid")
		if err == nil {
			t.Error("getPublicKey() error = nil, want error（kidが存在しない）")
		}
	})

	t.Run("RSA以外のkty（EC）はスキップされる", func(t *testing.T) {
		resetCache()
		t.Cleanup(resetCache)

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(jwks{
				Keys: []jwk{{Kid: "ec-kid", Kty: "EC", N: "", E: ""}},
			})
		}))
		defer server.Close()

		origTransport := httpClient.Transport
		t.Cleanup(func() { httpClient.Transport = origTransport })
		httpClient.Transport = server.Client().Transport

		host := strings.TrimPrefix(server.URL, "https://")
		t.Setenv("AUTH0_DOMAIN", host)

		_, err := getPublicKey("ec-kid")
		if err == nil {
			t.Error("getPublicKey() error = nil, want error（EC鍵はスキップ）")
		}
	})
}
