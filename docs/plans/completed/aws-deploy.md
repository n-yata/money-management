# AWS デプロイ計画

## 概要

AWS SAM を使って Lambda + API Gateway をデプロイし、フロントエンド（GitHub Pages）と接続するまでの手順。

---

## 前提条件の確認

- [ ] AWS アカウントを持っている
- [ ] AWS CLI がインストール済み（`aws --version` で確認）
- [ ] AWS SAM CLI がインストール済み（`sam --version` で確認）
- [ ] MongoDB Atlas M0 クラスターが作成済み
- [ ] Auth0 テナント・Application・API が作成済み

---

## ステップ 1: AWS CLI の認証設定

```powershell
aws configure
```

以下を入力：

| 項目 | 値 |
|---|---|
| AWS Access Key ID | IAM ユーザーのアクセスキー |
| AWS Secret Access Key | IAM ユーザーのシークレットキー |
| Default region name | `ap-northeast-1` |
| Default output format | `json` |

### IAM ユーザーに必要な権限

デプロイ用 IAM ユーザーには以下のポリシーが必要：

- `AWSLambda_FullAccess`
- `AmazonAPIGatewayAdministrator`
- `AWSCloudFormationFullAccess`
- `IAMFullAccess`（Lambda 実行ロール作成のため）
- `AmazonS3FullAccess`（SAM デプロイ用バケットのため）

---

## ステップ 2: SAM ビルド

```powershell
cd backend
sam build
```

---

## ステップ 3: 初回デプロイ（ガイド付き）

初回は `--guided` オプションで対話形式で設定する。

```powershell
sam deploy --guided
```

対話形式で以下を入力：

| 項目 | 推奨値 |
|---|---|
| Stack Name | `money-management` |
| AWS Region | `ap-northeast-1` |
| MongoDBUri | MongoDB Atlas の接続URI |
| Auth0Domain | `dev-r25g73f2tlbnmbtt.us.auth0.com` |
| Auth0Audience | `https://api.money-management.com` |
| AllowOrigin | `'https://n-yata.github.io'` |
| Environment | `production` |
| LocalAuth0Sub | （空のままEnter） |
| Confirm changes before deploy | `y` |
| Allow SAM CLI IAM role creation | `y` |
| Save arguments to configuration file | `y` |
| SAM configuration file | `samconfig.toml` |
| SAM configuration environment | `default` |

> `Save arguments to configuration file: y` にすると `samconfig.toml` にデプロイ設定が保存され、次回以降は `sam deploy` だけで実行できる。
> **`samconfig.toml` は `.gitignore` に追加済みのため、機密情報が Git に含まれる心配はない。**

---

## ステップ 4: デプロイ結果の確認

デプロイ完了後、以下のように API Gateway の URL が表示される：

```
CloudFormation outputs from deployed stack
---------------------------------------------
Key   ApiUrl
Value https://xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com/prod
```

この URL をメモしておく（次のステップで使用）。

AWS コンソールでも確認できる：
- **CloudFormation** → `money-management` スタック → **Outputs** タブ

---

## ステップ 5: GitHub Secrets に API_BASE_URL を設定

1. GitHub リポジトリ → **Settings** → **Secrets and variables** → **Actions**
2. `API_BASE_URL` を追加：

```
https://xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com/prod/api/v1
```

> ⚠️ API Gateway URL の末尾に `/api/v1` を付けること。

---

## ステップ 6: フロントエンドの再デプロイ

`API_BASE_URL` Secret を設定したら、GitHub Actions からフロントエンドを再デプロイする。

GitHub リポジトリ → **Actions** → **Deploy Frontend to GitHub Pages** → **Run workflow**

---

## ステップ 7: 動作確認

- [ ] `https://n-yata.github.io/money-management/` にアクセスできる
- [ ] Auth0 でログインできる
- [ ] ダッシュボードが表示される（子ども一覧が取得できる）
- [ ] 子ども追加・編集・削除ができる
- [ ] 収支記録の追加・削除ができる

---

## 2回目以降のデプロイ

コードを変更して再デプロイする場合：

```powershell
cd backend
sam build
sam deploy
```

`samconfig.toml` にパラメータが保存されているため、追加入力は不要。

---

## トラブルシューティング

### CORS エラーが出る場合

`AllowOrigin` パラメータの値を確認。シングルクォートで囲む必要がある：

```
AllowOrigin = "'https://n-yata.github.io'"
```

### 401 Unauthorized が返る場合

- Auth0 の `AUTH0_DOMAIN` / `AUTH0_AUDIENCE` が正しいか確認
- Lambda Authorizer のログを CloudWatch で確認

### Lambda のログ確認

```powershell
aws logs tail /aws/lambda/money-management-ChildrenFunction --follow
```
