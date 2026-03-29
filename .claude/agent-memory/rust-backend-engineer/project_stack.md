---
name: プロジェクト技術スタックと構成
description: money-management バックエンドの技術スタック、テスト方針、CIパターンの記録
type: project
---

## バックエンド技術スタック

- 言語: Go 1.25.0
- ランタイム: AWS Lambda + API Gateway (SAM)
- DB: MongoDB Atlas (mongo-driver/v2)
- 認証: Auth0 JWT (golang-jwt/jwt/v5)
- テストDB: testcontainers-go/modules/mongodb (統合テスト)

## テストパターン

### ユニットテスト（ビルドタグなし）
- `go test ./...` で実行可能
- ネットワーク・DB不要
- authorizer の handler テストはキャッシュ（`var cache jwksCache`）を直接操作してJWKS取得をバイパス

### 統合テスト（`//go:build integration`）
- `go test -tags=integration ./src/functions/... ./src/lib/...`
- testcontainers-go が Docker で MongoDB コンテナを自動起動
- サービスコンテナ定義不要

### キャッシュテスト用ヘルパーパターン（authorizer）
```go
func setTestCache(key *rsa.PublicKey, kid string) {
    cache.mu.Lock(); defer cache.mu.Unlock()
    cache.keys = map[string]*rsa.PublicKey{kid: key}
    cache.fetchedAt = time.Now()
}
func resetCache() {
    cache.mu.Lock(); defer cache.mu.Unlock()
    cache.keys = nil; cache.fetchedAt = time.Time{}
}
// 各テストで: t.Cleanup(resetCache)
```

## カバレッジ目標

| 対象 | 目標 |
|------|------|
| middleware (GetAuthSub等) | 90%以上 |
| AuthorizerのhandlerおよびJWT検証 | 90%以上 |
| バックエンド全体 | 70%以上 |

## テストしない範囲

- `main()` 関数（Lambda エントリーポイント）
- `getPublicKey` のHTTP実際呼び出し部分（キャッシュでバイパス）
