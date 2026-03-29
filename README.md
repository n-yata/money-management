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
- Go 1.21+
- AWS SAM CLI
- Auth0 テナント（事前作成が必要）
- MongoDB Atlas クラスター（事前作成が必要）

### フロントエンド

```bash
cd frontend
npm install
```

`src/environments/environment.ts` に Auth0 の情報を設定します（後述）。

```bash
ng serve
```

### バックエンド

```bash
cd backend
cp .env.example .env
# .env を編集して秘密情報を設定（後述）
go build ./...
sam build && sam local start-api
```

---

## 秘密情報・環境設定

### フロントエンド: `frontend/src/environments/environment.ts`

**gitにコミットしないでください（Auth0の情報が含まれます）。**

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
| Allowed Callback URLs | `http://localhost:4200, https://<your-github-pages-url>` |
| Allowed Logout URLs | `http://localhost:4200, https://<your-github-pages-url>` |
| Allowed Web Origins | `http://localhost:4200, https://<your-github-pages-url>` |

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
```

| 変数名 | 説明 | 取得元 |
|---|---|---|
| `MONGODB_URI` | MongoDB Atlas の接続URI | Atlas コンソール > Connect > Drivers |
| `AUTH0_DOMAIN` | Auth0 テナントドメイン | Auth0 コンソール > Settings |
| `AUTH0_AUDIENCE` | Auth0 API の Identifier | Auth0 コンソール > APIs |

### AWS SAM デプロイ時のパラメータ

SAM デプロイ時は `--parameter-overrides` でパラメータを渡します（コードには含めません）。

```bash
sam deploy \
  --parameter-overrides \
    MongoDBUri="mongodb+srv://..." \
    Auth0Domain="your-tenant.auth0.com" \
    Auth0Audience="https://api.money-management.example.com"
```

または `samconfig.toml` に記述します（このファイルも `.gitignore` に追加してください）。

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

```bash
# バックエンド
cd backend
go test ./...

# フロントエンド
cd frontend
ng test
```
