# 子どものおこずかい管理アプリ

子どもがお手伝いした回数に応じて今月のおこずかいを決定し、親子で収支を管理するPWAアプリです。

## 技術スタック

| レイヤー | 技術 |
|----------|------|
| フロントエンド | Angular (最新安定版) + GitHub Pages |
| PWA | Angular Service Worker (`@angular/pwa`) |
| 認証 | Auth0 (`@auth0/auth0-angular`) |
| バックエンド | AWS Lambda + API Gateway |
| ランタイム | Go (Lambda) |
| データベース | MongoDB Atlas (M0 無料枠, ap-northeast-1) |
| IaC | AWS SAM |

## アーキテクチャ

```
[Angular PWA]
    │  HTTPS + Access Token
    ▼
[AWS API Gateway]
    │
    ├──▶ [Lambda Authorizer] ──▶ [Auth0 JWKS] で署名検証
    │
    ├──▶ [Lambda: children]
    ├──▶ [Lambda: allowance-types]
    └──▶ [Lambda: records]
              │
              ▼
       [MongoDB Atlas M0]
```

## ディレクトリ構成

```
money-management/
├── frontend/                   # Angularアプリ (PWA)
│   └── src/app/
│       ├── auth/               # Auth0認証関連
│       ├── core/               # シングルトンサービス、ガード
│       ├── shared/             # 共有コンポーネント、パイプ
│       └── features/
│           ├── dashboard/      # ダッシュボード（親向け）
│           ├── children/       # 子ども管理
│           └── records/        # 収支記録
└── backend/                    # AWS Lambda関数群
    ├── src/
    │   ├── functions/          # Lambda関数（エンドポイント単位）
    │   │   ├── authorizer/     # JWT検証
    │   │   ├── children/       # 子ども管理API
    │   │   ├── allowance-types/# おこづかい種類API
    │   │   └── records/        # 収支記録API
    │   ├── models/             # MongoDBモデル
    │   ├── middleware/         # 共通ミドルウェア
    │   └── lib/                # DB接続など共通処理
    ├── template.yaml           # AWS SAM定義
    └── .env                    # 秘密情報（gitignore済み）
```

## セットアップ

### 前提条件

- Node.js 18+
- Go 1.25+
- AWS SAM CLI
- Docker（`sam local start-api` の実行に必要。Docker Desktop 推奨）
- Auth0 テナント（事前作成が必要）
- MongoDB Atlas クラスター（事前作成が必要）

### フロントエンド

**macOS / Linux:**
```bash
cd frontend
npm install
ng serve
```

**Windows (PowerShell):**
```powershell
cd frontend
npm install
ng serve
```

`src/environments/environment.ts` に Auth0 の情報を設定します（後述）。

> **注意:** `environment.ts` は `.gitignore` により追跡されません。初回は `environment.ts.example` をコピーして作成してください。
> ```bash
> cp src/environments/environment.ts.example src/environments/environment.ts
> # environment.ts を編集して Auth0 情報を設定
> ```

### バックエンド

**macOS / Linux:**
```bash
cd backend
cp .env.example .env
# .env を編集して秘密情報を設定（後述）
sam build && sam local start-api
```

**Windows (PowerShell):**
```powershell
cd backend
Copy-Item .env.example .env
# .env を編集して秘密情報を設定（後述）
sam build
sam local start-api
```

---

## 秘密情報・環境設定

### フロントエンド: `frontend/src/environments/environment.ts`

**このファイルは `.gitignore` により追跡されません。`environment.ts.example` をコピーして作成してください。**

```bash
cp frontend/src/environments/environment.ts.example frontend/src/environments/environment.ts
# environment.ts を編集して Auth0 情報を設定
```

```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api/v1',
  auth0: {
    domain: 'YOUR_AUTH0_DOMAIN',       // 例: your-tenant.auth0.com
    clientId: 'YOUR_AUTH0_CLIENT_ID',  // Auth0 Application の Client ID
    audience: 'YOUR_AUTH0_AUDIENCE',   // 例: https://api.money-management.example.com
  },
};
```

本番用は `environment.prod.ts` を同様に編集します（`apiBaseUrl` は AWS API Gateway の URL に変更）。

#### Auth0 Application の設定

Auth0 コンソールで **Single Page Application** を作成し、以下を設定します。

| 設定項目 | 値 |
|---|---|
| Allowed Callback URLs | `http://localhost:4200, https://YOUR_GITHUB_USERNAME.github.io/money-management/` |
| Allowed Logout URLs | `http://localhost:4200, https://YOUR_GITHUB_USERNAME.github.io/money-management/` |
| Allowed Web Origins | `http://localhost:4200, https://YOUR_GITHUB_USERNAME.github.io` |

> **注意:** 設定値のパスの有無に注意してください。
> - Callback / Logout URLs → **パス付き** (`/money-management/`) が必要。ログイン後のリダイレクト先 `redirect_uri` と完全一致させること。
> - Web Origins → **ドメインのみ**（パスなし）。トークン取得のオリジン検証に使われる。
> - ローカル開発用の `http://localhost:4200` も忘れずに追加すること。

#### Auth0 API の設定

Auth0 コンソールで **API** を作成し、`audience` に設定した値を `Identifier` に使います。

### バックエンド: `backend/.env`

**このファイルは `.gitignore` により追跡されません。**

`.env.example` をコピーして編集します。

```bash
# MongoDB Atlas
MONGODB_URI=mongodb+srv://<username>:<password>@<cluster>.mongodb.net/money-management?retryWrites=true&w=majority

# Auth0
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_AUDIENCE=https://api.money-management.example.com

# ローカル開発専用（本番では設定しないこと）
ENVIRONMENT=local
LOCAL_AUTH0_SUB=auth0|your-user-id   # Lambda Authorizer をスキップして使用するユーザーID
```

| 変数名 | 説明 | 取得元 |
|---|---|---|
| `MONGODB_URI` | MongoDB Atlas の接続URI | Atlas コンソール > Connect > Drivers |
| `AUTH0_DOMAIN` | Auth0 テナントドメイン | Auth0 コンソール > Settings |
| `AUTH0_AUDIENCE` | Auth0 API の Identifier | Auth0 コンソール > APIs |
| `ENVIRONMENT` | 実行環境（`local` / `production`）。`local` のみ `LOCAL_AUTH0_SUB` フォールバックが有効 | ローカルは `local` 固定 |
| `LOCAL_AUTH0_SUB` | `sam local` で Authorizer をスキップする際の代替ユーザーID。`ENVIRONMENT=local` のときのみ有効 | Auth0 コンソール > Users |

### AWS SAM デプロイ時のパラメータ

SAM デプロイ時は `samconfig.toml` にパラメータを記述します（このファイルは `.gitignore` に追加してください）。

```bash
# 初回セットアップ
cp backend/samconfig.toml.example backend/samconfig.toml
# samconfig.toml を編集して各値を設定してからデプロイ
cd backend && sam build && sam deploy
```

または `--parameter-overrides` でパラメータを直接渡すこともできます。

```bash
sam deploy \
  --parameter-overrides \
    MongoDBUri="mongodb+srv://..." \
    Auth0Domain="your-tenant.auth0.com" \
    Auth0Audience="https://api.money-management.example.com" \
    AllowOrigin="'https://<your-github-pages-url>'" \
    Environment="production"
```

| パラメータ | 説明 |
|---|---|
| `MongoDBUri` | MongoDB Atlas の接続URI |
| `Auth0Domain` | Auth0 テナントドメイン |
| `Auth0Audience` | Auth0 API の Identifier |
| `AllowOrigin` | CORS 許可オリジン。本番は GitHub Pages の URL をシングルクォートで囲む（例: `"'https://your-account.github.io'"` ） |
| `Environment` | 実行環境。本番は必ず `production` を指定（`local` では認証バイパスが有効になるため） |

### GitHub Actions Secrets

GitHub リポジトリの **Settings → Secrets and variables → Actions** で設定します。

#### フロントエンド自動デプロイ（GitHub Pages）

| Secret 名 | 説明 | 取得元 |
|---|---|---|
| `API_BASE_URL` | AWS API Gateway の URL（例: `https://xxxx.execute-api.ap-northeast-1.amazonaws.com/prod/api/v1`） | AWS デプロイ後に判明 |
| `AUTH0_DOMAIN` | Auth0 テナントドメイン | Auth0 コンソール > Settings |
| `AUTH0_CLIENT_ID` | Auth0 Application の Client ID | Auth0 コンソール > Applications |
| `AUTH0_AUDIENCE` | Auth0 API の Identifier | Auth0 コンソール > APIs |

> `API_BASE_URL` は AWS バックエンドのデプロイ完了後に設定してください。
> `GITHUB_TOKEN` は GitHub が自動で提供するため設定不要です。

#### バックエンド自動デプロイ（AWS SAM）

| Secret 名 | 説明 | 取得元 |
|---|---|---|
| `AWS_ACCESS_KEY_ID` | AWS IAM ユーザーのアクセスキーID | AWS コンソール > IAM |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM ユーザーのシークレットアクセスキー | AWS コンソール > IAM |
| `MONGODB_URI` | MongoDB Atlas の接続URI | Atlas コンソール > Connect |
| `AUTH0_DOMAIN` | Auth0 テナントドメイン（フロントエンドと共通） | Auth0 コンソール > Settings |
| `AUTH0_AUDIENCE` | Auth0 API の Identifier（フロントエンドと共通） | Auth0 コンソール > APIs |
| `CORS_ALLOW_ORIGIN` | CORS 許可オリジン（例: `https://your-account.github.io`） | GitHub Pages URL |

---

## ユーザー登録について

**アプリ画面からのユーザー登録機能はありません。**

Auth0 コンソールで親ユーザーを手動登録してください。

1. Auth0 コンソール > User Management > Users
2. 「Create User」から親のメールアドレスとパスワードを登録
3. 登録したユーザーでアプリにログイン

---

## API 一覧

すべてのエンドポイントは Auth0 JWT 認証が必須です。

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/api/v1/children` | 子ども一覧（残高含む） |
| POST | `/api/v1/children` | 子ども追加 |
| GET | `/api/v1/children/:id` | 子ども詳細 |
| PUT | `/api/v1/children/:id` | 子ども更新 |
| DELETE | `/api/v1/children/:id` | 子ども削除 |
| GET | `/api/v1/allowance-types` | おこづかい種類一覧 |
| POST | `/api/v1/allowance-types` | 種類追加 |
| PUT | `/api/v1/allowance-types/:id` | 種類更新 |
| DELETE | `/api/v1/allowance-types/:id` | 種類削除 |
| GET | `/api/v1/children/:id/records` | 収支記録一覧（`?year=&month=` フィルタ可） |
| POST | `/api/v1/children/:id/records` | 収支記録追加 |
| DELETE | `/api/v1/children/:id/records/:recordId` | 収支記録削除 |

## テスト

**macOS / Linux:**
```bash
# バックエンド
cd backend && go test ./...

# フロントエンド
cd frontend && ng test
```

**Windows (PowerShell):**
```powershell
# バックエンド
cd backend
go test ./...

# フロントエンド
cd frontend
ng test
```
