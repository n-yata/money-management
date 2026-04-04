# CI/CD・インフラ レビュー
レビュー日: 2026-04-04

## サマリー

家族向けおこづかい管理アプリとして、全体的に堅実な構成が取れている。セキュリティ面（`NoEcho`、Lambda Authorizer の強制適用）とコスト面（スロットリング、無料枠内のサービス選定）は適切に設計されている。一方で、バックエンドのデプロイ自動化が未実装であること、`deploy-frontend.yml` にテスト失敗時のデプロイ継続リスクが存在すること、SAM テンプレートの IAM ロール・ログ設定が明示されていないことが主な改善ポイントとなる。

---

## 良い点

### セキュリティ

- `MongoDBUri` に `NoEcho: true` を設定しており、CloudFormation コンソールで接続文字列が露出しない。
- Lambda Authorizer を `DefaultAuthorizer` に設定し、`AddDefaultAuthorizerToCorsPreflight: false` を明示している。プリフライトには認証を要求しない正しい設定。
- `ResultTtlInSeconds: 0` でキャッシュを無効化しており、ログアウト後のトークンが即座に無効化される。セキュリティと引き換えに Lambda Authorizer のコールド起動コストは増えるが、家族アプリ規模では問題ない判断。
- `ENVIRONMENT` パラメーターと `AllowedValues` で `production` / `local` のみ許可し、`LOCAL_AUTH0_SUB` による認証バイパスを本番で防ぐ仕組みが入っている。
- `environment.prod.ts` にプレースホルダー文字列 (`__XXX__`) を使用し、機密値をコードに埋め込まない設計が徹底されている。
- GitHub Actions の `permissions: contents: write` のスコープが最小限（ワークフロー全体ではなく必要な権限のみ）に絞られている。

### コスト最適化

- Lambda メモリを `128 MB`（最小値）に設定。Go バイナリは小さくコールドスタートも速いため、この設定で十分。
- API Gateway のスロットリングを `RateLimit: 10 rps / BurstLimit: 20` に設定。家族アプリとして適切で、AWS 無料枠を使い切るリクエスト数の到達を防ぐ。
- `provided.al2023` ランタイム（カスタムランタイム）を使用しており、Go バイナリを直接実行するため実行効率が高い。

### CI/CD の信頼性

- `backend-test.yml` でユニットテストと統合テストを分離し、それぞれ独立したジョブとして定義している。
- 統合テストは `testcontainers-go` で MongoDB を自動起動するため、外部サービスへの依存なしに CI が完結する。
- カバレッジ閾値をパッケージ別に設定（middleware: 90%、authorizer: 85%、全体: 70%）し、重要度に応じた品質管理ができている。
- `go-version: stable` で常に最新安定版の Go を使用し、バージョン固定の管理コストを削減している。
- `npm ci` を使用し、`package-lock.json` による再現性を確保している。
- アクションのバージョンを `@v4` に固定しており、予期せぬ破壊的変更の影響を受けない。

---

## 改善が必要な点

### 高優先度

#### 1. バックエンドのデプロイ自動化が未実装

`.github/workflows/` にバックエンドのデプロイワークフローが存在しない。現状では `sam build && sam deploy` を手動実行する必要があり、ヒューマンエラーやデプロイ忘れのリスクがある。

**推奨対応:** `deploy-backend.yml` を新規作成し、`main` ブランチへのプッシュかつ `backend/**` の変更時に `sam build` → `sam deploy` を自動実行する。AWS 認証情報は `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` を GitHub Secrets で管理する。

#### 2. `deploy-frontend.yml` でテスト失敗時にデプロイが継続するリスク

現在「Run unit tests」ステップは存在するが、テストが失敗してもジョブは次のステップ（ビルド・デプロイ）に進んでしまう可能性がある。`ng test` のプロセス終了コードが正しく伝播されるかどうかは環境依存。

**推奨対応:** テストステップに明示的な失敗条件を追加するか、または `--no-progress` フラグを付けて出力を整理し、CI ログでテスト結果を確認しやすくする。また、テストジョブとデプロイジョブを分離し `needs: test` で依存関係を明示することで、テスト失敗時にデプロイを完全に停止できる。

```yaml
# 推奨構成（概略）
jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - ...
      - name: Run unit tests
        working-directory: frontend
        run: npx ng test --watch=false --browsers=ChromeHeadless --no-progress

  deploy:
    name: Deploy
    needs: test  # テスト成功を前提条件とする
    runs-on: ubuntu-latest
    steps:
      - ...
```

#### 3. `deploy-frontend.yml` が `workflow_dispatch` でのみ手動実行できるが、バックエンドワークフロー変更時はトリガーされない

`deploy-frontend.yml` は `frontend/**` の変更のみを監視している。ワークフローファイル自体 (`.github/workflows/deploy-frontend.yml`) の変更はトリガーに含まれておらず、ワークフロー修正後の動作確認が難しい。`backend-test.yml` には `.github/workflows/backend-test.yml` のパスが含まれており、対称性がない。

**推奨対応:** `paths` に `".github/workflows/deploy-frontend.yml"` を追加する。

### 中優先度

#### 4. Lambda 関数に明示的な IAM ロールが定義されていない

SAM テンプレートに `Role` プロパティが記述されておらず、SAM がデフォルトで生成する IAM ロール (`AWSLambdaBasicExecutionRole`) が使用される。このロールには CloudWatch Logs への書き込み権限のみが含まれるが、意図しない権限が追加されないことを明示的に保証する記述がない。

**推奨対応:** `Policies` または `Role` を明示的に定義し、最小権限の原則（Principle of Least Privilege）を文書化する。

```yaml
# 例: Globals または各 Function に追加
Policies:
  - AWSLambdaBasicExecutionRole
```

#### 5. Lambda 関数に CloudWatch Logs の設定が不足している

ログのリテンション期間が設定されていない。デフォルトでは CloudWatch Logs は無期限保存となり、ログが蓄積されると予期しない CloudWatch コストが発生する可能性がある（家族アプリでも長期運用時は注意が必要）。

**推奨対応:** SAM テンプレートに `LoggingConfig` または `AWS::Logs::LogGroup` リソースを追加し、リテンション期間を設定する（例: 30日）。

```yaml
# Globals に追加
Globals:
  Function:
    LoggingConfig:
      LogFormat: JSON
    # LogGroup は別リソースで RetentionInDays を設定
```

#### 6. SAM デプロイ時のパラメーター管理が不明確

`template.yaml` に多数のパラメーター（`MongoDBUri`, `Auth0Domain`, etc.）が定義されているが、これらをデプロイ時にどう渡すかの仕組みが明示されていない。`samconfig.toml` や `--parameter-overrides` の使用方法が定義されておらず、手動デプロイ手順書が不足している。

**推奨対応:** `samconfig.toml` を作成して非機密パラメーター（`Auth0Domain`, `AllowOrigin`, `Environment`）を管理し、`MongoDBUri` のみ GitHub Secrets から注入する方式を文書化する。（`samconfig.toml` は `.gitignore` に追加するか、機密値を含まない形でコミットする。）

#### 7. `deploy-frontend.yml` の `permissions` が `contents: write` のみ

GitHub Pages へのデプロイには `pages: write` と `id-token: write` が必要な場合がある（GitHub 公式の Pages デプロイアクションを使用する場合）。`peaceiris/actions-gh-pages` は `GITHUB_TOKEN` の `contents: write` で動作するため現状は問題ないが、将来的に公式の `actions/deploy-pages` に移行する際に権限変更が必要になる。

### 低優先度

#### 8. Lambda タイムアウトが 30 秒に設定されているが、MongoDB コールドスタートの考慮が不明確

`Timeout: 30` は一般的に十分だが、MongoDB Atlas M0 (無料枠) はコールドスタート時に接続確立に数秒かかる場合がある。グローバルスコープでの接続再利用（CLAUDE.md に記載あり）が実装されているか、Lambda のウォームアップ戦略（定期的な ping）の有無を確認すること。

#### 9. `go-version: stable` による再現性の懸念

`go-version: stable` は常に最新の安定版 Go を使用するため、Go のマイナーバージョンアップ時に予期せぬビルドエラーが発生する可能性がある。

**推奨対応:** バージョンを `go-version: "1.23"` のように固定し、意図的なアップグレード時のみ変更するようにする。（`go.mod` に記載の Go バージョンと揃えると管理しやすい。）

#### 10. API Gateway の CORS 設定で `AllowCredentials` が未設定

`Auth0` を使用した認証フローでは、ブラウザが `Authorization` ヘッダーを送信するため CORS は問題ないが、将来的に Cookie ベース認証に切り替える場合は `AllowCredentials: "'true'"` が必要になる。現状の JWT Bearer 方式では問題なし。

---

## ファイル別レビュー詳細

### `.github/workflows/deploy-frontend.yml`

| 項目 | 評価 | コメント |
|------|------|---------|
| トリガー設定 | △ | `frontend/**` の変更のみ監視。ワークフローファイル自体の変更がトリガーに含まれない |
| テスト実行 | △ | テストジョブとデプロイジョブが分離されておらず、テスト失敗時の安全装置が弱い |
| 環境変数注入 | ○ | `sed` によるプレースホルダー置換で機密値をコードから分離できている |
| ビルド設定 | ○ | `--configuration production` と `--base-href` が正しく設定されている |
| キャッシュ | ○ | `cache-dependency-path: frontend/package-lock.json` でキャッシュが効く |
| デプロイ | ○ | `peaceiris/actions-gh-pages@v4` で安定したデプロイができている |
| 権限 | ○ | `contents: write` のみで最小限 |

### `.github/workflows/backend-test.yml`

| 項目 | 評価 | コメント |
|------|------|---------|
| トリガー設定 | ○ | PR と push の両方に対応。ワークフローファイル自体の変更もトリガーに含む |
| ユニットテスト | ○ | カバレッジ計測と閾値チェックが適切に実装されている |
| 統合テスト | ○ | `testcontainers-go` で外部依存なし。`-coverpkg=./...` でクロスパッケージカバレッジも計測 |
| カバレッジ戦略 | ○ | 重要度に応じた閾値設定（middleware: 90%, authorizer: 85%, 全体: 70%）が適切 |
| Go バージョン | △ | `go-version: stable` で再現性にやや懸念あり |
| アーティファクト | ○ | カバレッジレポートをアップロードしており、後から確認できる |
| タイムアウト | ○ | 統合テストに `-timeout=120s` を設定しており、ハング時の保護がある |

### `backend/template.yaml`

| 項目 | 評価 | コメント |
|------|------|---------|
| Lambda メモリ | ○ | 128 MB（最小値）で Go には十分。コスト最適 |
| Lambda タイムアウト | ○ | 30 秒は MongoDB 接続込みでも十分な余裕がある |
| ランタイム | ○ | `provided.al2023` + Go バイナリで高効率 |
| IAM ロール | △ | 明示的な定義なし。デフォルトロールに依存 |
| CloudWatch Logs | △ | ログリテンション設定なし。長期運用でコスト増の可能性 |
| API Gateway スロットリング | ○ | 家族アプリに適切な低めの設定（10 rps / 20 burst） |
| CORS 設定 | ○ | `AllowOrigin` をパラメーター化し、本番・開発で切り替え可能 |
| Lambda Authorizer | ○ | `DefaultAuthorizer` 適用、プリフライト除外、キャッシュ無効化が正しく設定 |
| `NoEcho` | ○ | `MongoDBUri` にのみ適用。適切 |
| `LocalAuth0Sub` バイパス | ○ | `ENVIRONMENT` パラメーターで本番での使用を防ぐ仕組みあり |
| Outputs | ○ | API URL と Authorizer ARN を出力しており、デプロイ後の確認が容易 |

### `backend/Makefile`

| 項目 | 評価 | コメント |
|------|------|---------|
| クロスコンパイル設定 | ○ | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` が正しく設定されている |
| Windows/Linux 共通対応 | ○ | `export` 構文で環境変数を設定し、インライン記法を避けている |
| ビルドターゲット | ○ | 各 Lambda 関数が独立したターゲットで定義されており、個別ビルドが可能 |

---

## 推奨アクション

優先度順に並べる。

### 即時対応（高優先度）

1. **バックエンドデプロイワークフローの作成**
   - `.github/workflows/deploy-backend.yml` を新規作成
   - トリガー: `main` ブランチへの `backend/**` 変更時
   - ステップ: `aws configure` → `sam build` → `sam deploy --no-confirm-changeset`
   - 必要な Secrets: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `MONGODB_URI`, `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`, `CORS_ALLOW_ORIGIN`

2. **`deploy-frontend.yml` のテスト・デプロイジョブ分離**
   - `test` ジョブと `deploy` ジョブを分離し、`needs: test` で依存関係を設定
   - テスト失敗時にデプロイが実行されないことを保証する

3. **`deploy-frontend.yml` のトリガーにワークフローファイル自体を追加**
   ```yaml
   paths:
     - "frontend/**"
     - ".github/workflows/deploy-frontend.yml"
   ```

### 近いうちに対応（中優先度）

4. **CloudWatch Logs のリテンション設定追加**
   - `AWS::Logs::LogGroup` リソースを Lambda 関数ごとに定義
   - `RetentionInDays: 30` を設定してコスト増を防ぐ

5. **IAM ロールの明示的定義**
   - `Globals.Function.Policies` に `AWSLambdaBasicExecutionRole` を明示

6. **`samconfig.toml` の作成と手動デプロイ手順の文書化**
   - 非機密パラメーターをファイルで管理
   - README または CLAUDE.md にデプロイコマンドを記載

### 余裕があれば対応（低優先度）

7. **`go-version: stable` を具体的なバージョンに固定**
   ```yaml
   go-version: "1.23"
   ```

8. **Lambda ウォームアップの検討**
   - MongoDB 接続のコールドスタート遅延が問題になる場合、EventBridge による定期 ping を検討
