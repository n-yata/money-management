# セキュリティソースコードレビュー

**実施日:** 2026-03-30
**対象:** money-management プロジェクト全体
**観点:** 認証・認可、インジェクション、入力バリデーション、機密情報管理、CORS、エラーハンドリング、依存パッケージ

---

## 全体評価

| 観点 | 評価 | コメント |
|------|------|---------|
| 認証・認可 | 良好 | JWT + Lambda Authorizer で実装。キャッシュ TTL の調整が必要 |
| 入力バリデーション | 良好 | バックエンド・フロントエンド両方で適切に実装 |
| CORS 設定 | **要改善** | Lambda が常に `*` を返す。環境変数で制御する必要 |
| 機密情報管理 | 良好 | `.env` は .gitignore で除外済み。パスワードローテーション推奨 |
| エラーハンドリング | 良好 | クライアントには詳細情報を返さない |
| MongoDB セキュリティ | 要監視 | インジェクション対策OK。トランザクションはM0で未サポート |
| 依存パッケージ | 良好 | 既知の脆弱性なし（2026-03-30時点） |

---

## 検出された問題

### [高] Lambda Authorizer キャッシュ TTL が長すぎる

**ファイル:** `backend/template.yaml` 行64

**問題:**
```yaml
ResultTtlInSeconds: 300  # 5分間
```

Lambda Authorizer が JWT トークンを 5分間キャッシュしており、その間はトークン検証をスキップする。
- ログアウト後も最大5分間、そのトークンで API を呼び出せる可能性がある
- 親が子どものデバイスを没収した直後にも操作できてしまう

**推奨対策:**
```yaml
ResultTtlInSeconds: 0   # 毎回検証（推奨）
# または
ResultTtlInSeconds: 60  # 1分（パフォーマンスとのバランス）
```

---

### [中] CORS AllowOrigin が常に `*`（Lambda で上書きされる）

**ファイル:** `backend/src/lib/response.go` 行24

**問題:**
```go
"Access-Control-Allow-Origin": "*",  // 常に * を返している
```

SAM テンプレートの `AllowOrigin` パラメータで制限的な設定をしても、Lambda のレスポンスヘッダーで常に `*` が上書きされる。
JWT 使用のため CSRF リスクは低いが、任意のドメインからのリクエストを許可してしまう。

**推奨対策:**
```go
// response.go
func JSONResponse(status int, body any) events.APIGatewayProxyResponse {
    allowOrigin := os.Getenv("CORS_ALLOW_ORIGIN")
    if allowOrigin == "" {
        allowOrigin = "https://your-account.github.io"
    }
    return events.APIGatewayProxyResponse{
        Headers: map[string]string{
            "Access-Control-Allow-Origin": allowOrigin,
            // ...
        },
    }
}
```

SAM テンプレートにも `CORS_ALLOW_ORIGIN` 環境変数を追加し、本番では GitHub Pages の URL を指定する。

---

### [中] LOCAL_AUTH0_SUB 環境変数が本番で設定されると認証バイパスになる

**ファイル:** `backend/src/middleware/auth.go` 行16

**問題:**
```go
if localSub := os.Getenv("LOCAL_AUTH0_SUB"); localSub != "" {
    return localSub, true  // ローカル開発用フォールバック
}
```

本番環境で誤って `LOCAL_AUTH0_SUB` が設定されると、すべてのリクエストが同一ユーザーとして扱われ、認証が事実上バイパスされる。

**推奨対策:**
```go
func GetAuthSub(request events.APIGatewayProxyRequest) (string, bool) {
    sub, ok := request.RequestContext.Authorizer["sub"].(string)
    if !ok || sub == "" {
        // 本番環境ではフォールバックを完全禁止
        if os.Getenv("ENVIRONMENT") == "production" {
            return "", false
        }
        if localSub := os.Getenv("LOCAL_AUTH0_SUB"); localSub != "" {
            return localSub, true
        }
        return "", false
    }
    return sub, true
}
```

または SAM テンプレートで本番 `ENVIRONMENT=production` を必ず設定する運用ルールを設ける。

---

### [低] エラーログに JWT アルゴリズム名が含まれる

**ファイル:** `backend/src/functions/authorizer/main.go` 行156

**問題:**
```go
return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
```

JWT ヘッダーのアルゴリズム値がログに出力される。
クライアントには返されないため実害は低いが、CloudWatch ログへのアクセスが漏洩した場合に情報が露出する。

**推奨対策:**
```go
log.Printf("[WARN] JWT validation failed: unexpected signing method")
return nil, fmt.Errorf("unexpected signing method")
```

---

### [低] MongoDB トランザクションが M0（無料枠）では原子性を保証しない

**ファイル:** `backend/src/functions/children/main.go` 行328-343

**問題:**
```go
_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    // records削除 → child削除 の2ステップ
})
```

MongoDB Atlas M0（無料枠）は単一ノード構成のため、`WithTransaction` を使っても実質的な原子性は保証されない。
子ども削除時に records の削除と child の削除の間で障害が発生すると、孤立した records レコードが残る可能性がある。

**推奨対策（短期）:**
- 削除順序を変えない（records → child の順は正しい）
- 定期的な孤立レコード検出・クリーンアップのバッチ処理を検討

**推奨対策（長期）:**
- 金銭データのため MongoDB M5 以上（レプリカセット）への移行を検討

---

## 適切に実装されている項目

| 項目 | 評価 |
|------|------|
| JWT 署名検証（RS256 明示チェック） | 適切 |
| ユーザー所有権チェック（全エンドポイントで user_id 検証） | 適切 |
| 入力バリデーション（バックエンド・フロントエンド） | 適切 |
| MongoDB インジェクション対策（`bson.M` パラメータ化クエリ） | 適切 |
| クライアントへのエラー詳細非公開 | 適切 |
| Auth0 SDK の正しい使用 | 適切 |
| `.env` の .gitignore 除外 | 適切 |

---

## 推奨対応スケジュール

### 即時対応（1〜2週間以内）
1. `ResultTtlInSeconds: 300` → `0` または `60` に変更
2. `response.go` の `Access-Control-Allow-Origin` を環境変数化

### 短期対応（1ヶ月以内）
1. `LOCAL_AUTH0_SUB` の本番環境バイパス防止
2. MongoDB パスワードのローテーション

### 中期対応（3ヶ月以内）
1. AWS Secrets Manager への秘密情報移行検討
2. MongoDB M5 以上への upgrade 検討（トランザクション完全サポートのため）
