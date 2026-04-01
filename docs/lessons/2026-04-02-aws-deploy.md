# AWS デプロイ時のトラブルシューティング知見

日付: 2026-04-02

---

## 1. SAM Lambda Authorizer は `FunctionPayloadType: REQUEST` が必須

### 問題
SAM のデフォルトは TOKEN タイプの Authorizer。TOKEN タイプでは Lambda に渡されるイベントの形式が異なり、`request.Headers` が空になる。

### 症状
- Authorizer のログに `received headers: map[]`
- 毎回 `Unauthorized` になる

### 解決策
`template.yaml` の Authorizer 定義に `FunctionPayloadType: REQUEST` を追加する。

```yaml
Auth:
  DefaultAuthorizer: LambdaAuthorizer
  Authorizers:
    LambdaAuthorizer:
      FunctionArn: !GetAtt AuthorizerFunction.Arn
      FunctionPayloadType: REQUEST   # ← これが必須
      Identity:
        Headers:
          - Authorization
```

---

## 2. CORS プリフライト（OPTIONS）に `AddDefaultAuthorizerToCorsPreflight: false` が必須

### 問題
SAM で `DefaultAuthorizer` を設定すると、OPTIONS リクエストにも Authorizer が適用される。
プリフライトリクエストには Authorization ヘッダーがないため、401 が返り CORS エラーになる。

### 症状
```
Access to fetch ... has been blocked by CORS policy:
Response to preflight request doesn't pass access control check
```

### 解決策
```yaml
Auth:
  DefaultAuthorizer: LambdaAuthorizer
  AddDefaultAuthorizerToCorsPreflight: false   # ← これが必須
```

---

## 3. MongoDB Atlas の IP アクセスリストに `0.0.0.0/0` を追加

### 問題
Lambda の IP は動的に変わるため、Atlas のデフォルト設定（自分の IP のみ許可）では Lambda から接続できない。

### 症状
- Lambda が 504 Gateway Timeout を返す
- Lambda ログが一切残らない（Lambda 自体が起動できていない）

### 解決策
MongoDB Atlas コンソール → **Security** → **Database & Network Access** → **IP Access List** → `0.0.0.0/0` を追加。

---

## 4. PWA の Service Worker キャッシュはデプロイ後に手動クリアが必要

### 問題
GitHub Actions で新しいビルドをデプロイしても、ブラウザが Service Worker のキャッシュから古い JS バンドルを読み込む。

### 症状
- GitHub Secrets を更新して再デプロイしても、古い API URL が使われる

### 解決策
Chrome DevTools → **Application** → **Service Workers** → **Unregister** してリロード。
またはシークレットウィンドウで確認。

---

## 5. Authorizer のデバッグは詳細ログ追加が有効

### 教訓
Authorizer が `Unauthorized` を返すだけでは原因が特定できない。
デバッグ時は以下のようにログを追加して原因を絞り込む。

```go
log.Printf("received headers: %v", request.Headers)
log.Printf("token validation failed: %v", err)
log.Printf("failed to fetch JWKS: %v", err)
```

---

## 6. Lambda Authorizer の正しい設定チェックリスト

AWS SAM で Lambda Authorizer を使う際の必須確認事項：

- [ ] `FunctionPayloadType: REQUEST` を指定している
- [ ] `AddDefaultAuthorizerToCorsPreflight: false` を指定している
- [ ] MongoDB Atlas の IP アクセスリストに `0.0.0.0/0` を追加している
- [ ] `AUTH0_DOMAIN` / `AUTH0_AUDIENCE` の環境変数が正しく設定されている
