# セキュリティレビュー
レビュー日: 2026-04-04

## サマリー

本レビューは、money-management アプリのフロントエンド（Angular + Auth0）、バックエンド（Go + AWS Lambda）、インフラ（AWS API Gateway + Lambda Authorizer）の三層を対象に実施した。

全体的な設計は堅牢であり、JWT 検証・所有権チェック・レート制限など、家族向けアプリとして必要な基本的なセキュリティ対策は適切に実装されている。一方で、いくつかの改善点が確認された。最も注意すべき問題は、開発用 Auth0 クレデンシャル（`client_id`）が `environment.ts` にハードコードされてリポジトリにコミットされていることと、`CORS_ALLOW_ORIGIN` のデフォルト値が `'*'` になっている点である。

---

## 良い点

**認証・認可の設計**
- Lambda Authorizer が全エンドポイントに適用されており、公開エンドポイントが存在しない設計になっている (`DefaultAuthorizer: LambdaAuthorizer`)
- JWT 検証では署名アルゴリズムを `RS256` に固定し、`alg: none` 攻撃を明示的に排除している (`WithValidMethods([]string{"RS256"})`)
- `issuer` と `audience` の両方を検証しており、異なる Auth0 テナントのトークンの流用を防いでいる
- JWKS はメモリキャッシュ（TTL: 1時間）を使用しており、Auth0 へのリクエスト数を削減しつつ、毎回の JWT 検証も設定で有効化されている (`ResultTtlInSeconds: 0`)

**所有権チェック**
- `FindOwnedChild` が `{_id: oid, user_id: userID}` の複合条件でクエリしており、他ユーザーの子どもへのアクセスを確実に遮断している
- `findOwnedAllowanceType` も同様の複合条件を使用している
- `createRecord` 時に `allowance_type_id` の所有権チェック（`user_id` による絞り込み）を実施している
- `deleteRecord` では `{_id: recordOID, child_id: child.ID}` で削除し、他の子どもの記録を削除できない設計になっている

**インジェクション対策**
- MongoDB に対して `bson.M` や `bson.D` による構造化クエリのみを使用しており、NoSQL インジェクションが実質的に不可能な実装になっている
- 文字列をそのまま MongoDB クエリに埋め込む箇所がない

**バリデーション**
- 各エンドポイントで入力バリデーション（文字数、数値範囲、日付形式）を実施している
- `utf8.RuneCountInString` を使用しており、マルチバイト文字の長さを正確に計測している
- 金額の上限（10,000,000円）が設定されており、異常値による残高計算への影響を抑制している

**インフラ設計**
- API Gateway レベルでレート制限（バースト: 20、レート: 10 req/s）を設定しており、DDoS に対する最低限の保護がある
- SAM テンプレートで `MongoDBUri` に `NoEcho: true` を指定しており、CloudFormation コンソールでの機密情報露出を防いでいる
- `CORS_ALLOW_ORIGIN` が環境変数化されており、本番では GitHub Pages の URL に制限できる設計になっている
- `ENVIRONMENT=local` の場合のみ `LOCAL_AUTH0_SUB` フォールバックを許可しており、本番での誤使用を防ぐ明示的なオプトイン設計になっている

**フロントエンド**
- `authInterceptor` が自社 API（`environment.apiBaseUrl`）以外のリクエストにはトークンを付与しない実装になっており、トークンの不要な外部送信を防いでいる
- Auth0 の Universal Login（リダイレクト方式）を採用しており、アプリ内でパスワードを扱わない
- 機密情報はプレースホルダー（`__API_BASE_URL__` 等）で管理し、GitHub Actions の Secrets から CI/CD で注入する設計になっている

**CI/CD セキュリティ**
- フロントエンドデプロイ前にユニットテストを必須化している
- バックエンドは middleware（90%）・authorizer（85%）・全体（70%）のカバレッジ閾値を CI で強制している

---

## 改善が必要な点

### 高優先度（即時対応）

#### [H-1] 開発用 Auth0 クレデンシャルがリポジトリにコミットされている

**ファイル:** `frontend/src/environments/environment.ts`

```
domain: 'dev-r25g73f2tlbnmbtt.us.auth0.com',
clientId: 'oK3CRIKQX4xnVF0ZGOFd4zkLIuUqv091',
```

Auth0 の `clientId` は「公開情報」に分類される（OAuth 2.0 の `client_id` はパブリッククライアントでは秘密にならない）という考え方もあるが、`domain` と `clientId` の組み合わせをリポジトリに公開することには以下のリスクがある。

- 攻撃者がこの情報を使って Auth0 のログインページを模倣したフィッシングページを作成できる
- 開発テナントへの不審なログイン試行のターゲットになる可能性がある
- `audience` も一緒に公開されているため、開発環境向けのトークン要求が容易になる

**対応案:** `environment.ts` の Auth0 設定も `environment.prod.ts` と同様にプレースホルダー化し、ローカル開発者は `.env` ファイルや `environment.local.ts`（`.gitignore` 済み）から読み込む方式に変更する。

#### [H-2] CORS_ALLOW_ORIGIN のデフォルト値が `'*'`（ワイルドカード）

**ファイル:** `backend/src/lib/response.go` (L18), `backend/template.yaml` (L46)

`response.go` の `allowOrigin()` 関数は、環境変数 `CORS_ALLOW_ORIGIN` が未設定の場合に `"*"` を返す。また `template.yaml` の `AllowOrigin` パラメータのデフォルト値が `"'*'"` になっている。

デプロイ時に `AllowOrigin` パラメータを明示的に指定し忘れた場合、任意のオリジンからの API アクセスが許可される状態になる。

**対応案:**
- `template.yaml` の `AllowOrigin` パラメータからデフォルト値 `"'*'"` を削除し、デプロイ時に必須パラメータとして強制する
- `response.go` の `allowOrigin()` は `CORS_ALLOW_ORIGIN` が空の場合にエラーログを出力し、フォールバックとして空文字列を返す（CORS ヘッダー自体を省略する方が安全）

### 中優先度

#### [M-1] JWKS キャッシュのロールオーバー時に古いキーが消える（Key Rollover の考慮不足）

**ファイル:** `backend/src/functions/authorizer/main.go` (L90-93)

```go
cache.mu.Lock()
cache.keys = newKeys  // 旧キーセットを完全に上書き
cache.fetchedAt = time.Now()
cache.mu.Unlock()
```

Auth0 が JWKS のキーをロールオーバー（新旧キーを並行提供）している最中に、キャッシュ更新タイミングによっては一時的に旧キーで署名されたまだ有効なトークンを検証できなくなる可能性がある。

Auth0 は通常、移行期間中は旧キーと新キーを同時に提供するため、`newKeys` に旧キーが含まれないケースが発生しうる（Auth0 側のキャッシュ設定による）。

**対応案:** JWKS 更新時に既存キャッシュと新キーセットをマージする（既存キーを新キーで上書きしつつ、新キーにない旧キーも一定期間保持する）。ただし現時点では Auth0 の JWKS ローテーション仕様上、実害が生じるケースは少ない。

#### [M-2] `deleteAllowanceType` でのトランザクション使用は MongoDB M0（無料枠）では動作しない可能性がある

**ファイル:** `backend/src/functions/allowance-types/main.go` (L210-227)

```go
session, err := db.Client().StartSession()
_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    // records の allowance_type_id をクリア → allowance_type を削除
})
```

CLAUDE.md のコメントには「MongoDB Atlas M0（無料枠）はレプリカセット非対応のためマルチドキュメントトランザクションを使用できない」と記載されている。`children` の削除処理ではトランザクションを使用していないが、`allowance-types` の削除では `WithTransaction` を使用している。

M0 は実際にはレプリカセット構成（MongoDB Atlas は M0 でもレプリカセット）だが、マルチドキュメントトランザクションのサポートが制限される場合がある。本番デプロイ後に動作確認を必ず行うこと。

#### [M-3] `ResultTtlInSeconds: 0` による Lambda Authorizer の毎回実行はコールドスタートリスクがある

**ファイル:** `backend/template.yaml` (L78)

`ResultTtlInSeconds: 0` はログアウト後のトークン即時無効化のために適切な設定だが、Lambda Authorizer 自体がコールドスタートした場合、Auth0 JWKS の取得（ネットワーク通信）が毎回の認証パスに加わるリスクがある。

`httpClient` の `Timeout: 5 * time.Second` は適切だが、Auth0 が応答しない場合のエンドユーザー体験（認証タイムアウトエラー）とセキュリティのトレードオフを考慮した設計であることを確認しておく。

#### [M-4] フロントエンドの `authGuard` が `isAuthenticated$` を使用している（レースコンディションのリスク）

**ファイル:** `frontend/src/app/auth/auth.guard.ts`

```typescript
return auth.isAuthenticated$.pipe(
  tap(isAuthenticated => {
    if (!isAuthenticated) {
      auth.loginWithRedirect();
    }
  })
);
```

`isAuthenticated$` はアプリ起動時の初期ロード完了前に `false` を返す可能性がある。Auth0 Angular SDK の `canActivate` 実装としては、`isLoading$` が `false` になってから `isAuthenticated$` を評価する `AuthGuard`（SDK 提供の `AuthGuard` クラス）の使用が推奨される場合がある。現在の実装では、認証済みユーザーが一瞬ログイン画面にリダイレクトされることがあるが、セキュリティ上の問題（バイパス）は起きない（ガードが `false` を返すだけ）。

#### [M-5] `environment.ts` の `apiBaseUrl` がローカルホスト固定になっている

**ファイル:** `frontend/src/environments/environment.ts`

```
apiBaseUrl: 'http://localhost:3000/api/v1',
```

`http://` (非 TLS) を使用しているが、これはローカル開発専用のため許容範囲内。ただし `authInterceptor` がこの URL へのリクエストにアクセストークンを付与するため、ローカル環境でトークンが HTTP で送信される点は認識しておく。

### 低優先度

#### [L-1] `samconfig.toml` が `.gitignore` に含まれているため、デプロイパラメータが共有されない

**ファイル:** `.gitignore` (L4)

`samconfig.toml` を `.gitignore` に含めているため、デプロイ時のパラメータ（`AllowOrigin` 等）がチームメンバー間で共有されない。`samconfig.toml` に機密情報が含まれる場合は現在の方針が正しいが、非機密パラメータ（`AllowOrigin`、`Environment` 等）のデフォルト値は `samconfig.toml.example` として共有する、または CI/CD でデプロイする仕組みを整備することを推奨する。

#### [L-2] `records` の `listRecords` レスポンスに全フィールドが含まれる

**ファイル:** `backend/src/functions/records/main.go` (L154)

`models.Record` 構造体の全フィールドが `data` としてレスポンスに含まれる。現時点では `child_id` や `allowance_type_id`（ObjectID）が含まれており、これ自体は機密情報ではないが、フロントエンドで不要なフィールドを除外したレスポンス型（DTO）を使用することで、将来的なフィールド追加時の情報漏洩リスクを低減できる。

#### [L-3] Authorizer の IAM ポリシーが同一 API 内の全リソースを許可している

**ファイル:** `backend/src/functions/authorizer/main.go` (L192-196)

```go
resource = strings.Join(parts[:2], "/") + "/*/*"
```

認証成功時に `{apiId}/{stage}/*/*` (全メソッド・全パス) を許可する IAM ポリシーを発行している。最小権限の原則から言えば、要求されたリソースのみを許可する方が望ましい（`request.MethodArn` をそのまま使用する）。ただし `ResultTtlInSeconds: 0` のため IAM ポリシーキャッシュは使用されておらず、実害は限定的である。

#### [L-4] ログ出力にトークンの情報が含まれない設計は良いが、エラーログが詳細すぎる可能性

**ファイル:** `backend/src/functions/authorizer/main.go` (L172)

```go
log.Printf("token validation failed: %v", err)
```

JWT 検証失敗の理由（有効期限切れ、署名不正、発行者不正 等）が CloudWatch Logs に記録される。これはデバッグには有用だが、攻撃者が CloudWatch Logs へのアクセス権を得た場合に検証ロジックの詳細が分かる。低リスクではあるが、本番では検証失敗の理由をログに出力しない選択肢も検討できる。

---

## レイヤー別レビュー詳細

### フロントエンド

| 観点 | 評価 | 詳細 |
|------|------|------|
| XSS 対策 | 良好 | Angular のテンプレートエンジンはデフォルトで HTML エスケープを行う。`innerHTML` 等の危険な API の使用なし |
| トークンの扱い | 良好 | Auth0 Angular SDK が `localStorage` / `memory` のいずれかでトークンを管理。アプリコード内でトークン文字列を直接取得・保存していない |
| CORS 設定 | 該当なし | SPA のため CORS はバックエンド側の設定が支配的 |
| AuthGuard | 概ね良好 | 全保護ルートに `authGuard` が適用されている。レースコンディションは実害なし（[M-4] 参照） |
| インターセプター | 良好 | 自社 API 以外へのトークン送信を防ぐ実装になっている |
| クレデンシャルのコミット | 要改善 | 開発用 Auth0 `clientId` がリポジトリに公開されている（[H-1] 参照） |
| 依存関係 | 概ね良好 | Angular 19・Auth0 Angular 2.7 等、比較的新しいバージョンを使用。`npm audit` による定期チェックを推奨 |

### バックエンド

| 観点 | 評価 | 詳細 |
|------|------|------|
| JWT 検証 | 良好 | RS256 固定・issuer/audience 検証・alg:none 対策がすべて実装されている |
| 所有権チェック | 良好 | 全 CRUD 操作で `user_id` による所有権確認が実施されている |
| NoSQL インジェクション | 良好 | 構造化クエリ（bson.M）のみ使用。文字列結合によるクエリ構築なし |
| 入力バリデーション | 良好 | 文字数・数値範囲・日付形式の検証が全エンドポイントで実施されている |
| エラーレスポンス | 良好 | 内部エラーの詳細（スタックトレース等）をクライアントに返していない |
| CORS | 要確認 | デフォルト値 `'*'` のリスクあり（[H-2] 参照） |
| ローカル開発バイパス | 概ね良好 | `ENVIRONMENT=local` の場合のみ有効化。本番では無効化されている |
| 依存関係 | 良好 | `golang-jwt/jwt/v5`・`mongo-driver/v2` 等、メンテナンスされているライブラリを使用 |

### インフラ

| 観点 | 評価 | 詳細 |
|------|------|------|
| Lambda Authorizer の適用 | 良好 | 全エンドポイントに `DefaultAuthorizer` として適用。OPTIONS (CORS preflight) は除外 |
| IAM ポリシー | 要改善 | 全リソース許可のポリシーを発行している（[L-3] 参照）。実害は限定的 |
| レート制限 | 良好 | バースト 20・レート 10 req/s。家族向けアプリとして適切な設定 |
| Secrets 管理 | 概ね良好 | `NoEcho: true` 設定・GitHub Secrets からの注入で機密情報を保護 |
| `.gitignore` | 良好 | `.env`・`samconfig.toml`・ビルド成果物が適切に除外されている |
| CI/CD パーミッション | 概ね良好 | `contents: write` のみ付与。`pull-requests: write` 等の不要な権限なし |
| 環境分離 | 良好 | `ENVIRONMENT` パラメータによる本番/ローカルの明示的な分離 |

---

## 推奨アクション

優先度順に実施することを推奨する。

### 即時対応

1. **[H-1] `environment.ts` の Auth0 クレデンシャルをリポジトリから削除する**
   - `environment.ts` の `domain`・`clientId`・`audience` をプレースホルダーに変更する
   - ローカル開発者向けに `environment.local.ts.example` を作成し、セットアップ手順を `CLAUDE.md` に追記する
   - `.gitignore` に `frontend/src/environments/environment.local.ts` を追加する
   - Git 履歴からの削除は `git filter-repo` 等で実施することを検討する（開発用テナントのため緊急度は低いが、対応を推奨）

2. **[H-2] `AllowOrigin` パラメータからデフォルト値 `'*'` を削除する**
   - `template.yaml` の `AllowOrigin` パラメータの `Default:` 行を削除する
   - `response.go` の `allowOrigin()` でフォールバック `"*"` を削除し、未設定時のログ出力に変更する
   - デプロイ手順書（`samconfig.toml.example` 等）に本番 URL の設定を明記する

### 中期的な改善

3. **[M-1] JWKS キャッシュのマージ処理を追加する**（Auth0 のキーロールオーバー対応）

4. **[M-4] Auth0 Angular SDK の推奨する `AuthGuard` パターンへの移行を検討する**（`isLoading$` 待機）

5. **`npm audit` と `go mod audit` の定期実行を CI に組み込む**（依存関係の脆弱性チェック）

### 継続的な運用

6. **MongoDB Atlas のネットワークアクセス設定で Lambda の送信元 IP を制限する**（Lambda の NAT Gateway を固定 IP 化した場合）

7. **CloudWatch Logs のログ保存期間を設定し、不要なログが長期間残らないようにする**

8. **Auth0 の Anomaly Detection（ブルートフォース保護）とログ監視を有効化する**
