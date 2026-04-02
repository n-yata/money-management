# API設計書

## 基本仕様

| 項目 | 仕様 |
|------|------|
| ベースURL | `https://<api-id>.execute-api.ap-northeast-1.amazonaws.com/prod` |
| プレフィックス | `/api/v1` |
| データ形式 | JSON |
| 文字コード | UTF-8 |
| 認証 | Lambda Authorizer（Auth0 JWTトークン） |

---

## 認証

### Lambda Authorizer

すべてのエンドポイントはAPI GatewayのLambda Authorizerで保護される。

**リクエストヘッダー（必須）:**
```
Authorization: Bearer <Auth0アクセストークン>
```

**検証フロー:**
```
1. Authorization ヘッダーからトークンを取得
2. Auth0 JWKS エンドポイントから公開鍵を取得
3. JWTの署名・有効期限・issuerを検証
4. 検証OK → IAM Policy (Allow) を発行
5. 検証NG → 401 Unauthorized
```

**認証エラーレスポンス:**
```json
// 401 Unauthorized（トークンなし・無効）
{ "message": "Unauthorized" }

// 403 Forbidden（権限なし）
{ "message": "Forbidden" }
```

---

## 共通仕様

### レスポンス形式

**成功時:**
```json
// 200 OK / 201 Created
{ "data": { ... } }          // 単一リソース
{ "data": [ ... ] }          // 複数リソース
```

**エラー時:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "名前は必須です"
  }
}
```

### エラーコード一覧

| HTTPステータス | code | 説明 |
|----------------|------|------|
| 400 | `VALIDATION_ERROR` | バリデーションエラー |
| 401 | `UNAUTHORIZED` | 認証トークンなし・無効 |
| 403 | `FORBIDDEN` | 他ユーザーのリソースへのアクセス |
| 404 | `NOT_FOUND` | リソースが存在しない |
| 409 | `DUPLICATE_CHORE` | 同日・同種類のお手伝いがすでに登録済み |
| 500 | `INTERNAL_ERROR` | サーバー内部エラー |

### IDの形式
- すべてのリソースIDはMongoDB ObjectId（24文字の16進数文字列）

---

## エンドポイント一覧

### 子ども管理

| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/api/v1/children` | 子ども一覧取得 |
| `POST` | `/api/v1/children` | 子ども追加 |
| `GET` | `/api/v1/children/:id` | 子ども詳細取得 |
| `PUT` | `/api/v1/children/:id` | 子ども情報更新 |
| `DELETE` | `/api/v1/children/:id` | 子ども削除 |

### おこづかいの種類

| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/api/v1/allowance-types` | 種類一覧取得 |
| `POST` | `/api/v1/allowance-types` | 種類追加 |
| `GET` | `/api/v1/allowance-types/:id` | 種類詳細取得 |
| `PUT` | `/api/v1/allowance-types/:id` | 種類更新 |
| `DELETE` | `/api/v1/allowance-types/:id` | 種類削除 |

### 収支記録

| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/api/v1/children/:id/records` | 収支記録一覧取得 |
| `POST` | `/api/v1/children/:id/records` | 収支記録追加 |
| `DELETE` | `/api/v1/children/:id/records/:recordId` | 収支記録削除 |

---

## 子ども管理 API

### GET `/api/v1/children`

ログイン中の親ユーザーに紐づく子ども一覧を取得する。各子どもの現在残高を含む。

**リクエスト:** なし

**レスポンス `200 OK`:**
```json
{
  "data": [
    {
      "id": "661a2b3c4d5e6f7a8b9c0d1e",
      "name": "たろう",
      "age": 8,
      "base_allowance": 1000,
      "balance": 1200,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    },
    {
      "id": "661a2b3c4d5e6f7a8b9c0d1f",
      "name": "はなこ",
      "age": 6,
      "base_allowance": 800,
      "balance": 800,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ]
}
```

---

### POST `/api/v1/children`

新しい子どもを登録する。

**リクエストボディ:**
```json
{
  "name": "たろう",
  "age": 8,
  "base_allowance": 1000
}
```

**バリデーション:**
| フィールド | 制約 |
|-----------|------|
| `name` | 必須、1〜20文字 |
| `age` | 必須、整数、1〜18 |
| `base_allowance` | 必須、整数、0以上 |

**レスポンス `201 Created`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0d1e",
    "name": "たろう",
    "age": 8,
    "base_allowance": 1000,
    "balance": 0,
    "created_at": "2026-03-27T10:00:00Z",
    "updated_at": "2026-03-27T10:00:00Z"
  }
}
```

---

### GET `/api/v1/children/:id`

特定の子どもの詳細情報を取得する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |

**レスポンス `200 OK`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0d1e",
    "name": "たろう",
    "age": 8,
    "base_allowance": 1000,
    "balance": 1200,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-03-01T00:00:00Z"
  }
}
```

**エラー:**
- `404 Not Found`: 指定IDの子どもが存在しない、または他ユーザーのリソース

---

### PUT `/api/v1/children/:id`

子どもの情報を更新する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |

**リクエストボディ:**
```json
{
  "name": "たろう",
  "age": 9,
  "base_allowance": 1200
}
```

**バリデーション:** `POST /api/v1/children` と同じ

**レスポンス `200 OK`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0d1e",
    "name": "たろう",
    "age": 9,
    "base_allowance": 1200,
    "balance": 1200,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-03-27T10:00:00Z"
  }
}
```

---

### DELETE `/api/v1/children/:id`

子どもを削除する。関連する収支記録もすべて削除する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |

**レスポンス `200 OK`:**
```json
{ "data": null }
```

**エラー:**
- `404 Not Found`: 指定IDの子どもが存在しない、または他ユーザーのリソース

---

## 収支記録 API

### GET `/api/v1/children/:id/records`

特定の子どもの収支記録一覧を取得する。月フィルタが必須。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |

**クエリパラメータ:**
| パラメータ | 必須 | 説明 | 例 |
|-----------|------|------|-----|
| `year` | 必須 | 取得対象の年 | `2026` |
| `month` | 必須 | 取得対象の月（1〜12） | `3` |

**レスポンス `200 OK`:**
```json
{
  "data": [
    {
      "id": "661a2b3c4d5e6f7a8b9c0e1f",
      "child_id": "661a2b3c4d5e6f7a8b9c0d1e",
      "type": "income",
      "amount": 500,
      "description": "お手伝い",
      "date": "2026-03-27",
      "created_at": "2026-03-27T10:00:00Z"
    },
    {
      "id": "661a2b3c4d5e6f7a8b9c0e2f",
      "child_id": "661a2b3c4d5e6f7a8b9c0d1e",
      "type": "expense",
      "amount": 200,
      "description": "おかし",
      "date": "2026-03-25",
      "created_at": "2026-03-25T15:00:00Z"
    }
  ]
}
```

- `created_at` の降順（登録が新しい順）で返す
- 該当月のデータが0件の場合は空配列 `[]` を返す
- `date` フィールドは ISO 8601 形式（例: `2026-03-27T00:00:00Z`）で返却される。フロントエンドは `yyyy/MM/dd` にフォーマットして表示する

---

### POST `/api/v1/children/:id/records`

収支記録を追加する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |

**リクエストボディ:**
```json
// 種類を選択した場合
{
  "type": "income",
  "amount": 50,
  "description": "お皿洗い",
  "date": "2026-03-27",
  "allowance_type_id": "661a2b3c4d5e6f7a8b9c0f1a"
}

// 種類を選択しない場合（allowance_type_id は省略可）
{
  "type": "expense",
  "amount": 200,
  "description": "おかし",
  "date": "2026-03-27"
}
```

**バリデーション:**
| フィールド | 制約 |
|-----------|------|
| `type` | 必須、`"income"` または `"expense"` |
| `amount` | 必須、整数、1以上 |
| `description` | 必須、1〜50文字 |
| `date` | 必須、`YYYY-MM-DD` 形式 |
| `allowance_type_id` | 任意、指定する場合は自分のユーザーに紐づく種類IDであること。同じ `child_id`・`allowance_type_id`・`date` の組み合わせは1件のみ登録可能（1日1回制限） |

**レスポンス `201 Created`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0e1f",
    "child_id": "661a2b3c4d5e6f7a8b9c0d1e",
    "type": "income",
    "amount": 50,
    "description": "お皿洗い",
    "date": "2026-03-27",
    "allowance_type_id": "661a2b3c4d5e6f7a8b9c0f1a",
    "created_at": "2026-03-27T10:00:00Z"
  }
}
```

---

**エラー:**
- `404 Not Found`: 子どもが存在しない、または他ユーザーのリソース
- `409 Conflict` (`DUPLICATE_CHORE`): `allowance_type_id` を指定した場合、同日・同種類がすでに登録済み

---

### DELETE `/api/v1/children/:id/records/:recordId`

収支記録を削除する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 子どものID |
| `recordId` | 収支記録のID |

**レスポンス `200 OK`:**
```json
{ "data": null }
```

**エラー:**
- `404 Not Found`: 指定IDの収支記録が存在しない、または対象の子どもが他ユーザーのリソース

---

## おこづかいの種類 API

### GET `/api/v1/allowance-types`

ログイン中のユーザーに紐づく種類一覧を取得する。

**リクエスト:** なし

**レスポンス `200 OK`:**
```json
{
  "data": [
    {
      "id": "661a2b3c4d5e6f7a8b9c0f1a",
      "name": "お皿洗い",
      "amount": 50,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    },
    {
      "id": "661a2b3c4d5e6f7a8b9c0f1b",
      "name": "掃除機かけ",
      "amount": 80,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET `/api/v1/allowance-types/:id`

特定の種類の詳細情報を取得する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 種類のID |

**レスポンス `200 OK`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0f1a",
    "name": "お皿洗い",
    "amount": 50,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**エラー:**
- `404 Not Found`: 指定IDの種類が存在しない、または他ユーザーのリソース

---

### POST `/api/v1/allowance-types`

新しい種類を登録する。

**リクエストボディ:**
```json
{
  "name": "お皿洗い",
  "amount": 50
}
```

**バリデーション:**
| フィールド | 制約 |
|-----------|------|
| `name` | 必須、1〜30文字 |
| `amount` | 必須、整数、1以上 |

**レスポンス `201 Created`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0f1a",
    "name": "お皿洗い",
    "amount": 50,
    "created_at": "2026-03-27T10:00:00Z",
    "updated_at": "2026-03-27T10:00:00Z"
  }
}
```

---

### PUT `/api/v1/allowance-types/:id`

種類の情報を更新する。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 種類のID |

**リクエストボディ:**
```json
{
  "name": "お皿洗い",
  "amount": 60
}
```

**バリデーション:** `POST /api/v1/allowance-types` と同じ

**レスポンス `200 OK`:**
```json
{
  "data": {
    "id": "661a2b3c4d5e6f7a8b9c0f1a",
    "name": "お皿洗い",
    "amount": 60,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-03-27T10:00:00Z"
  }
}
```

---

### DELETE `/api/v1/allowance-types/:id`

種類を削除する。この種類を参照している収支記録の `allowance_type_id` は `null` になる。

**パスパラメータ:**
| パラメータ | 説明 |
|-----------|------|
| `id` | 種類のID |

**レスポンス `200 OK`:**
```json
{ "data": null }
```

**エラー:**
- `404 Not Found`: 指定IDの種類が存在しない、または他ユーザーのリソース

---

## データモデル

### Child

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `id` | string | MongoDB ObjectId |
| `name` | string | 子どもの名前 |
| `age` | integer | 年齢 |
| `base_allowance` | integer | 基本おこずかい額（円） |
| `balance` | integer | 現在の残高（円）※レスポンス時に計算 |
| `created_at` | string | 作成日時（ISO 8601） |
| `updated_at` | string | 更新日時（ISO 8601） |

### AllowanceType

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `id` | string | MongoDB ObjectId |
| `name` | string | 種類名 |
| `amount` | integer | 報酬金額（円、正の整数） |
| `created_at` | string | 作成日時（ISO 8601） |
| `updated_at` | string | 更新日時（ISO 8601） |

### Record

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `id` | string | MongoDB ObjectId |
| `child_id` | string | 子どものID |
| `allowance_type_id` | string \| null | 種類のID（任意） |
| `type` | string | `"income"` または `"expense"` |
| `amount` | integer | 金額（円、正の整数） |
| `description` | string | 説明・メモ |
| `date` | string | 収支発生日（`YYYY-MM-DD`） |
| `created_at` | string | 作成日時（ISO 8601） |

---

## セキュリティ仕様

### 所有権チェック
- `children`・`allowance-types` リソースへのアクセス時、Lambda関数はトークンの `sub` クレームと `users.auth0_sub` を照合する
- 一致しない場合は `404 Not Found` を返す（`403` ではなくリソースの存在を隠蔽する）
- `allowance_type_id` を指定して収支記録を作成する際、その種類が自分のユーザーに紐づくか検証する

### Lambda Authorizer キャッシュ
- Authorizerの結果はキャッシュしない（TTL: 0秒）
- ログアウト後の即時無効化を保証するため、毎リクエストでJWT検証を行う
- キャッシュキーはAuthorizationヘッダーのトークン値
