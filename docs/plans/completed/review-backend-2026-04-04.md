# バックエンド コードレビュー
レビュー日: 2026-04-03

---

## サマリー

全体評価: **高品質** (4.2 / 5.0)

主要な発見事項:
- セキュリティ設計は堅牢。Lambda Authorizer による全エンドポイント保護、`auth0_sub` による所有権チェックが一貫して実装されている。
- テストが充実しており、統合テスト（testcontainers）でモックなしの実クエリ検証を行っている点は特に優れている。
- パフォーマンス面では `$lookup` によるN+1解消、DB接続の `sync.Once` によるコネクション再利用が適切に実施されている。
- 改善余地として「TestHandler のケース分岐に文字列リテラルを使った条件分岐」「`ResolveUser` での `updated_at` 非更新」「重複するテストセットアップコード」などが挙げられる。いずれも機能・セキュリティには影響しない品質改善レベルの指摘。

---

## 良い点

### 1. セキュリティ設計の一貫性
- すべてのハンドラーが先頭で `middleware.GetAuthSub()` を呼び出し、取得できなければ即 401 を返している。認証バイパスの経路が存在しない。
- `FindOwnedChild` / `findOwnedAllowanceType` がクエリフィルタに `user_id` を必ず含めるため、他ユーザーのリソースへのアクセスはDB層でシャットアウトされる（クエリインジェクションのリスクも `bson.M` 型安全な構造体でほぼゼロ）。
- `ENVIRONMENT=local` でのみ `LOCAL_AUTH0_SUB` フォールバックを有効にする設計が明示的オプトイン方式になっており、本番誤設定のリスクを低減している。
- Lambda Authorizer が `ResultTtlInSeconds: 0` でキャッシュ無効化 → ログアウト後も即時無効化。

### 2. JWT 検証の実装品質
- RSA 公開鍵を `sync.RWMutex` でスレッドセーフにキャッシュし、TTL (1時間) 切れ後に再取得する設計が適切。
- `gojwt.WithValidMethods([]string{"RS256"})` で署名アルゴリズムをホワイトリスト制限しており、`alg: none` 攻撃を防いでいる。
- `issuer` / `audience` 検証を `gojwt.Parse` オプションで行っているため、クレーム検証の抜け漏れがない。
- `httpClient` に 5 秒タイムアウトを設定し、Auth0 無応答時のハング防止を実装している。

### 3. パフォーマンスの考慮
- DB接続を `sync.Once` でシングルトン化してコールドスタート後のコネクション再利用を実現している。
- `listChildren` で `$lookup + $addFields + $project` の Aggregation Pipeline を使い、N+1 クエリを解消している。
- `calcBalanceForChild` で全件ロードを避け、DB側で集計している。
- `records` コレクションに `(child_id, date)` 複合インデックスを設定しており、月フィルタクエリが効率的に実行される。

### 4. テストの品質
- テーブル駆動テストを基本とし、境界値・異常系・所有権チェックのケースが網羅されている。
- `testcontainers` によるインメモリMongoDBではなく実際の Docker コンテナを使い、実クエリで検証している（モック差異のリスクなし）。
- `t.Cleanup` / `t.Setenv` を使ってテスト間の独立性が確保されている。
- Authorizer のテストで `httptest.NewTLSServer` を使ってJWKSエンドポイントをモック → ネットワーク依存なしにHTTPS通信まで検証している。
- `getPublicKey` のキャッシュを直接操作する `setTestCache` / `resetCache` ヘルパーが実装されており、ネットワークアクセスを回避しつつキャッシュの動作を検証できる設計。

### 5. コード構成
- `lib/` パッケージに共通処理（`FindOwnedChild`, `ResolveUser`, `CalcBalance`, `JSONResponse`）を切り出し、Lambda 関数間の重複を排除している。
- `models/` がデータ構造の単一の真実の源泉になっており、コレクション名定数が一元管理されている。
- `ChildResponse` で `UserID` を JSON から除外する設計（`json:"-"`）が適切。内部識別子の情報漏洩を防いでいる。
- `deleteAllowanceType` でモンゴDBトランザクションを使い、`records.allowance_type_id` の null 化と種類削除をアトミックに実行している。
- `deleteChild` のコメントで「M0 はトランザクション非対応のため逐次削除」とその理由・リスクが明示されており、将来の変更者への情報共有が行き届いている。

---

## 改善が必要な点

### 高優先度

#### H-1: `TestHandler` のケース識別に文字列リテラルを使っている
**ファイル:** `src/functions/authorizer/main_test.go` (354〜365行目)

```go
switch tt.name {
case "Authorizationヘッダーなし → error":
    authHeader = ""
case "Bearer プレフィックスなし → error":
    authHeader = token
...
```

テストケース名をスイッチ条件に使うのは、テスト名のリファクタリング時にサイレントに動作が変わるリスクがある。ケース固有のロジックは構造体フィールドで表現すべき。

**推奨修正方針:**
```go
// テストケース構造体に authHeader フィールドを追加する
type testCase struct {
    ...
    authHeaderFn func(token string) string  // トークンからヘッダー文字列を生成する関数
}
```

#### H-2: `ResolveUser` が `updated_at` を更新しない
**ファイル:** `src/lib/user.go`

現在の `$setOnInsert` は新規作成時のみ `updated_at` を設定する。既存ユーザーが再ログインした場合、`updated_at` は更新されない。`$set` を組み合わせれば解決できる（ユーザーの最終ログイン日時の把握・監査目的）。

```go
bson.M{
    "$setOnInsert": bson.M{
        "_id":        bson.NewObjectID(),
        "auth0_sub":  auth0Sub,
        "created_at": now,
    },
    "$set": bson.M{
        "updated_at": now,
    },
},
```

ただし、現状で機能上の問題はない。監査・運用目的での改善として捉えてよい。

---

### 中優先度

#### M-1: テストセットアップコードが各テストファイルで重複している
**ファイル:** 統合テストファイル3ファイル（children, allowance-types, records）

`TestMain`, `newTestDB`, `newTestUser` が3ファイルに重複している。これはGoの `package main` の制約（`testcontainers` の統合テストが `main` パッケージで行われるため共有パッケージに切り出しにくい）から生じる構造上の制約であり、完全な解消は難しいが、認識として記録しておく。

将来的にパッケージ構成をリファクタリングする場合（例: Lambda を `internal/` パッケージで関数を公開する形に変更する場合）は共有ヘルパーを `testutil` パッケージに切り出せる。

#### M-2: `EnsureIndexes` が `init()` から毎回呼ばれる
**ファイル:** `src/functions/allowance-types/main.go`, `src/functions/children/main.go`, `src/functions/records/main.go`

`init()` はコールドスタート時だけでなく、同一コンテナが再利用されても Lambda の初期化が再実行されるタイミング（再デプロイ後など）に呼ばれる。MongoDB の `CreateIndex` は冪等なので問題はないが、毎リクエストではなく `sync.Once` で保護することでコールドスタート時のオーバーヘッドを明示的に制御できる。

```go
var indexOnce sync.Once

func init() {
    indexOnce.Do(func() {
        ctx := context.Background()
        if err := lib.EnsureIndexes(ctx); err != nil {
            log.Printf("EnsureIndexes warning: %v", err)
        }
    })
}
```

ただし `init()` 自体が Lambda コンテナ起動時に1回しか呼ばれないため、現状でも実質的な問題はない（`sync.Once` の追加はより防御的なコーディングとしての改善）。

#### M-3: `records` への月フィルタが必須になっており、全期間取得ができない
**ファイル:** `src/functions/records/main.go` (113行目)

```go
if yearStr == "" || monthStr == "" {
    return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "yearとmonthは必須です"), nil
}
```

現在の残高計算は Aggregation で全期間集計しているが、もし将来「全期間の収支履歴をCSVエクスポートする」などの要件が追加された場合、API変更が必要になる。現時点の要件では問題なし。設計上の制約として記録しておく。

#### M-4: `go.mod` の Go バージョンが不正
**ファイル:** `go.mod` (3行目)

```
go 1.25.0
```

Go 1.25 は執筆時点（2026-04-03）で存在しないバージョン。おそらく `1.23.0` または `1.24.0` の誤記。`go mod tidy` を実行するか、正しいバージョンに修正すること。ビルドには影響しない可能性があるが、toolchain 解決に影響する場合がある。

#### M-5: `records` の `ChildID` が JSON レスポンスに含まれる
**ファイル:** `src/models/record.go`

```go
ChildID bson.ObjectID `bson:"child_id" json:"child_id"`
```

`ChildID` はURLパスで既に確定しており、レスポンスボディに含める必要はない。フロントエンドへの不要な情報露出を避けるために `json:"-"` にすることも検討できる（ただしクライアントが利用している場合は Breaking Change になる）。

---

### 低優先度

#### L-1: `allowance-types` の `deleteAllowanceType` がAtlas M0でトランザクションを使っている
**ファイル:** `src/functions/allowance-types/main.go` (205〜222行目)

`deleteChild` のコメントに「M0 はトランザクション非対応」と記載されているが、`deleteAllowanceType` ではトランザクションを使用している。実際にはAtlas M0はレプリカセットを内包しているためトランザクションが使用できるケースもあるが、公式サポートとの差異がある。

テストでは `WithReplicaSet("rs0")` を使っておりトランザクション動作を確認済みではあるが、本番環境（M0）でのトランザクション動作は公式には保証されていない点を認識しておく必要がある。`deleteChild` と同様に逐次削除パターンへの変更を検討できる。

#### L-2: `struct` タグのインデント揃えが一部不統一
**ファイル:** `src/models/child.go`, `src/models/record.go`

```go
// child.go: 一部フィールドで余分なスペースが入っている
Name          string             `bson:"name"            json:"name"`
```

`bson.ObjectID` と `string` の型名の長さが異なるため手動でインデントを揃えようとしているが、一部不均等になっている。`gofmt` では修正されないためコメントのみ。可読性への軽微な影響。

#### L-3: Authorizer が `generatePolicy` でリソース ARN を制限していない場合のフォールバックが弱い
**ファイル:** `src/functions/authorizer/main.go` (190〜196行目)

```go
if len(parts) >= 2 {
    resource = strings.Join(parts[:2], "/") + "/*/*"
} else {
    resource = request.MethodArn // ARN がパースできない場合は元の ARN をそのまま使用
}
```

`len(parts) < 2` のケースは実際のAPI Gatewayリクエストでは発生しないが、フォールバックとして元の `MethodArn`（特定の1エンドポイント）を使う設計。これで動作するが、ワイルドカードを付与しないため同一APIの他エンドポイントは `403` になる可能性がある。フォールバック時も `"arn:aws:execute-api:*:*:*"` にするかエラーを返す方が明示的。

#### L-4: `children/handler_test.go` 内の一部テストでエラーチェックを省略
**ファイル:** `src/functions/children/handler_test.go` (299行目, 421行目など)

```go
json.Unmarshal([]byte(resp.Body), &result)  // エラーを捨てている
```

`json.Unmarshal` のエラーを無視しているケースが数ヶ所ある。続くアサーションでゼロ値との比較になり実質的に問題は検出できるが、デバッグ時にパース失敗の原因が見えにくくなる。

---

## ファイル別レビュー詳細

### `src/functions/authorizer/main.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | RS256ホワイトリスト、issuer/audience検証、kidベースの鍵取得、全て適切 |
| パフォーマンス | 良 | JWKSキャッシュ(TTL 1時間, RWMutex)が機能している |
| コード品質 | 良 | 責務が明確。`buildRSAPublicKey` が独立した関数として切り出されテスタブル |
| 注意点 | L-3 参照 | ARNパース失敗時のフォールバック設計が若干曖昧 |

### `src/functions/children/main.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | 所有権チェックが全CRUDで実施済み |
| パフォーマンス | 優 | Aggregationによる残高計算でN+1を解消、全件メモリロードなし |
| コード品質 | 良 | ルーティングロジックがシンプルなswitchで可読性高い |
| 注意点 | なし | deleteChildのコメントが技術的判断を明示しており良好 |

### `src/functions/allowance-types/main.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | 削除時にrecordsのFK参照をトランザクションでnull化する設計が適切 |
| パフォーマンス | 良 | 基本的なFindOneによる所有権確認後に更新 |
| 注意点 | L-1 参照 | M0でのトランザクションの公式サポート確認が望ましい |

### `src/functions/records/main.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | allowance_type_idのユーザー所有権チェック、childの所有権チェック双方を実施 |
| ビジネスロジック | 優 | 1日1回制限のチェック（CountDocuments）が正確に実装されている |
| 注意点 | M-3 参照 | 月フィルタ必須設計は仕様上の制約として記録 |

### `src/middleware/auth.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | ENVIRONMENTの明示的オプトイン設計が優秀。デフォルト（未設定）でもフォールバック禁止 |
| テスタビリティ | 優 | ロジックが単純で全分岐がユニットテストでカバーされている |

### `src/lib/db.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| パフォーマンス | 優 | `sync.Once` による接続再利用が適切 |
| 注意点 | コメントに「同一コンテナでは回復しない」旨が明記されており、制約の認識が適切 |

### `src/lib/user.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| 安全性 | 優 | `FindOneAndUpdate + upsert` でrace conditionを防止 |
| 注意点 | H-2 参照 | `updated_at` が既存ユーザーで更新されない |

### `src/lib/response.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| 設計 | 良 | CORS_ALLOW_ORIGINのSAMシングルクォート除去処理がコメントで説明されており明快 |
| テスト | 優 | `make(chan int)` でMarshal失敗を意図的にテストしている点が良い |

### `src/lib/indexes.go`
| 項目 | 評価 | 所見 |
|------|------|------|
| 設計 | 良 | インデックス定義がCLAUDE.mdのスキーマ設計と一致している |
| 冪等性 | 優 | 複数回呼び出しても安全 |

### `src/models/`
| 項目 | 評価 | 所見 |
|------|------|------|
| 設計 | 良 | `json:"-"` による内部フィールドの隠蔽が適切 |
| 注意点 | L-2, M-5 参照 | タグインデント不統一、ChildIDのJSON露出 |

### `template.yaml`
| 項目 | 評価 | 所見 |
|------|------|------|
| セキュリティ | 優 | DefaultAuthorizer設定でCORSプリフライトのみ除外、全エンドポイント保護 |
| コスト | 優 | MemorySize: 128MB、Timeout: 30秒、スロットリング設定あり |
| 設計 | 良 | Lambda関数がリソース単位で分割されており単一責任 |
| NoEcho | 優 | MongoDBUri に `NoEcho: true` が設定されておりCloudFormationコンソールへの漏洩を防止 |

---

## 推奨アクション

| 優先度 | アクション | 対象ファイル | 工数目安 |
|--------|-----------|------------|---------|
| 高 | `TestHandler` のケース識別をフィールドベースに変更 | `authorizer/main_test.go` | 30分 |
| 高 | `go.mod` の Go バージョン修正 | `go.mod` | 5分 |
| 中 | `ResolveUser` に `$set updated_at` を追加 | `lib/user.go` + テスト | 15分 |
| 中 | `EnsureIndexes` を `sync.Once` で保護 | 各 `main.go` | 20分 |
| 低 | `json.Unmarshal` エラーチェックをテストに追加 | `children/handler_test.go` 他 | 20分 |
| 低 | `Record.ChildID` の JSON 露出要否を確認・修正 | `models/record.go` | 10分 |
| 低 | M0 でのトランザクション動作を本番で確認 | 運用確認 | 要検証 |

---

## 総評

このバックエンドコードは、家族向けの小規模アプリとしての要件を満足しつつ、セキュリティと保守性において商用品質に近い水準に達している。特に「Lambda Authorizer による全エンドポイント保護」「DB層での所有権チェック」「testcontainersによる実DB統合テスト」は設計判断として非常に優れている。

改善点のほとんどは機能・セキュリティに影響しないコード品質・慣習レベルの指摘であり、致命的な問題は発見されなかった。

**ターゲット・デリート！レビュー完了だよ！**
