# バックエンドソースレビュー結果

レビュー日: 2026-03-28
対象: backend/src/ 配下

---

## サマリー

| 重要度 | 件数 |
|--------|------|
| 🔴 Critical（必須対応） | 5件 |
| 🟡 Warning（推奨対応） | 9件 |
| 🔵 Info（改善提案） | 7件 |

---

## 指摘事項

### 🔴 Critical（必須対応）

---

#### C-1: resolveUser の競合状態（Race Condition）による重複ユーザー作成

- **ファイル**: `functions/children/main.go:66-88`, `functions/allowance-types/main.go:61-82`, `functions/records/main.go:71-92`（3箇所に同じコードが重複）
- **問題**: `FindOne` でユーザーが存在しないことを確認してから `InsertOne` するまでの間に、同一 auth0_sub で別リクエストが割り込むと、2つの `User` ドキュメントが作成される可能性がある。auth0_sub には unique インデックスが設定されているため2件目は失敗するが、エラーを呼び出し元に返してしまい、ユーザーの最初のリクエストが 500 エラーになる。
- **影響**: 初回ログイン直後に同時リクエストが来た場合（PWA でのページロード時など）、正規ユーザーが 500 エラーを受け取る。
- **改善案**: MongoDB の `FindOneAndUpdate` + `upsert: true` でアトミックに処理する。

```go
opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
now := time.Now()
var user models.User
err := col.FindOneAndUpdate(ctx,
    bson.M{"auth0_sub": auth0Sub},
    bson.M{
        "$setOnInsert": bson.M{
            "_id":        bson.NewObjectID(),
            "auth0_sub":  auth0Sub,
            "created_at": now,
            "updated_at": now,
        },
    },
    opts,
).Decode(&user)
```

---

#### C-2: deleteChild のカスケード削除が非アトミック

- **ファイル**: `functions/children/main.go:313-322`
- **問題**: `records.DeleteMany` が成功した後、`children.DeleteOne` が失敗した場合、records だけが消えて子どもドキュメントは残る。結果として「records が存在しない子ども」という不整合状態になり、その後の残高計算は 0 を返し続ける。
- **影響**: データ整合性の破壊。削除リトライしても records は既に消えているため整合性回復不可能。
- **改善案**: MongoDB の `Session` + `WithTransaction` でトランザクションを使用する（Atlas M0 はレプリカセットでありトランザクション対応）。

```go
session, err := client.StartSession()
if err != nil { ... }
defer session.EndSession(ctx)
_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    if _, err := recordsCol.DeleteMany(sc, bson.M{"child_id": child.ID}); err != nil {
        return nil, err
    }
    return childrenCol.DeleteOne(sc, bson.M{"_id": child.ID})
})
```

---

#### C-3: deleteAllowanceType の records 更新も非アトミック

- **ファイル**: `functions/allowance-types/main.go:220-233`
- **問題**: `records.UpdateMany`（allowance_type_id を $unset）が成功した後、`allowance_types.DeleteOne` が失敗した場合、種類ドキュメントは残るが紐づく records の allowance_type_id は消えた状態になる。
- **影響**: 既存の収支記録から種類情報が失われる不整合。C-2 と同様、リトライしても回復不能。
- **改善案**: C-2 と同様にトランザクションを使用する。

---

#### C-4: Lambda Authorizer の IAM ポリシーが全リソースを許可

- **ファイル**: `functions/authorizer/main.go:177`
- **問題**: 認証成功時に `arn:aws:execute-api:*:*:*` でワイルドカードポリシーを発行している。これは認証済みユーザーであれば、すべての API Gateway リソース（他のステージ、他の API）に対しても実行権限が与えられることを意味する。
- **影響**: 将来的に同じ AWS アカウントで別の API Gateway を立てた場合、そちらへのアクセスも許可されてしまうリスクがある。また API Gateway のキャッシュ（ResultTtlInSeconds: 300）と組み合わせると、キャッシュされたポリシーが意図しないリソースへのアクセスを許可し続ける。
- **改善案**: `request.MethodArn` を使って発行リクエスト対象のリソースのみ許可するか、同一 API 内の全リソースに限定したARNを使用する。

```go
// MethodArnから API/ステージ部分だけ抽出して /* を付ける
// 例: arn:aws:execute-api:ap-northeast-1:123456789:abc123def/prod/*/*
parts := strings.Split(request.MethodArn, "/")
resource := strings.Join(parts[:2], "/") + "/*/*"
return generatePolicy(sub, "Allow", resource), nil
```

---

#### C-5: JWKS 取得時の HTTP タイムアウト未設定

- **ファイル**: `functions/authorizer/main.go:61`
- **問題**: `http.Get(...)` でデフォルトの `http.DefaultClient` を使用しており、タイムアウトが設定されていない。Auth0 の JWKS エンドポイントが応答しない場合、Lambda がタイムアウト（30秒）まで待ち続ける。
- **影響**: Lambda が 30 秒間ブロックされ、全リクエストが詰まる。コールドスタートと重なるとさらに悪化する。
- **改善案**:

```go
var httpClient = &http.Client{Timeout: 5 * time.Second}
// http.Get の代わりに httpClient.Get を使用
resp, err := httpClient.Get(fmt.Sprintf("https://%s/.well-known/jwks.json", domain))
```

---

### 🟡 Warning（推奨対応）

---

#### W-1: resolveUser が全 Lambda に重複コピーされている

- **ファイル**: `functions/children/main.go:66-88`, `functions/allowance-types/main.go:61-82`, `functions/records/main.go:71-92`
- **問題**: 完全に同一の `resolveUser` 関数が 3 つの Lambda パッケージにコピーされている。C-1 の修正を 3 箇所同時に行う必要があり、修正漏れが発生しやすい。
- **影響**: 保守性の低下。1箇所で修正してもバグが残る。
- **改善案**: `lib` パッケージに `ResolveUser(ctx, db, auth0Sub)` として切り出す。同様に `jsonResponse` / `errorResponse` / `findOwnedChild` も共通化できる。

---

#### W-2: jsonResponse での json.Marshal エラー無視

- **ファイル**: `functions/children/main.go:23-29`（他 Lambda も同様）
- **問題**: `b, _ := json.Marshal(body)` でエラーを無視している。通常は問題ないが、マーシャル不可能な値（チャネル型、関数型など）が含まれると空レスポンスが返る。
- **影響**: デバッグが困難になる。エラー時にフロントエンドが空ボディを受け取って予期しない動作をする可能性がある。
- **改善案**: エラー時は 500 を返すか、`log.Printf` でエラーを記録する。

---

#### W-3: listChildren の残高計算が N+1 問題になっている

- **ファイル**: `functions/children/main.go:161-175`
- **問題**: 子ども一覧取得時に、N 人の子どもに対して N 回 `calcBalanceForChild`（= MongoDB クエリ）を実行している。子どもが増えるほどレイテンシが線形に増加する。
- **影響**: 子ども 10 人で MongoDB クエリが 11 回、全 records を取得するため転送データ量も多い。
- **改善案**: MongoDB の Aggregation Pipeline を使い、1クエリで残高を集計する。

```go
pipeline := mongo.Pipeline{
    {{Key: "$match", Value: bson.M{"user_id": user.ID}}},
    {{Key: "$lookup", Value: bson.M{
        "from":         models.CollectionRecords,
        "localField":   "_id",
        "foreignField": "child_id",
        "as":           "records",
    }}},
    {{Key: "$addFields", Value: bson.M{
        "balance": bson.M{"$subtract": []any{
            bson.M{"$sum": bson.M{"$map": bson.M{
                "input": bson.M{"$filter": bson.M{
                    "input": "$records",
                    "cond":  bson.M{"$eq": []any{"$$this.type", "income"}},
                }},
                "in": "$$this.amount",
            }}},
            bson.M{"$sum": bson.M{"$map": bson.M{
                "input": bson.M{"$filter": bson.M{
                    "input": "$records",
                    "cond":  bson.M{"$eq": []any{"$$this.type", "expense"}},
                }},
                "in": "$$this.amount",
            }}},
        }},
    }}},
}
```

---

#### W-4: calcBalanceForChild が全 records を取得している（getChild/updateChild）

- **ファイル**: `functions/children/main.go:93-106`
- **問題**: 残高計算のために全期間の records を全件メモリに読み込んでいる。records が多くなるほど MongoDB からの転送データ量とメモリ使用量が増大する。
- **影響**: 長期利用で records が数百件になるとレイテンシとメモリコストが増大する。Lambda の 128MB メモリ制限に対するリスクもある。
- **改善案**: MongoDB の `$group` + `$sum` による Aggregation で集計値だけを取得する。

---

#### W-5: EnsureIndexes が Lambda ハンドラーから呼ばれていない

- **ファイル**: `lib/indexes.go`
- **問題**: `EnsureIndexes` 関数が定義されているが、どの Lambda の `main` 関数からも呼ばれていない。インデックスが存在しない状態でデプロイした場合、クエリがコレクションスキャンになる。特に `users.auth0_sub` の unique インデックスがないと C-1 の競合状態による重複が検知されない。
- **影響**: インデックス未作成時のクエリ性能低下、unique 制約が機能しない。
- **改善案**: 各 Lambda の `init()` 関数または `main()` で `lib.EnsureIndexes` を呼び出す。（冪等なので毎回呼んでも安全）

---

#### W-6: User モデルの auth0_sub と email が API レスポンスに含まれる可能性

- **ファイル**: `models/user.go:10-16`
- **問題**: `User` 構造体に `json:"auth0_sub"` と `json:"email"` タグが付いており、誤って `User` 型をそのままレスポンスに含めた場合に内部識別子が漏れる。現時点ではレスポンスに含めていないが、将来的なリスクがある。
- **影響**: Auth0 の sub ID は内部識別子であり、クライアントに公開すべきでない。
- **改善案**: `json:"-"` タグを付けてシリアライズから除外するか、レスポンス専用の `UserResponse` 型を用意する。

---

#### W-7: CORS 設定が AllowOrigin: '*'（ワイルドカード）

- **ファイル**: `backend/template.yaml:42`
- **問題**: `AllowOrigin: "'*'"` で全オリジンからのアクセスを許可している。GitHub Pages のフロントエンドのみを想定しているため、本番環境では特定のオリジンに制限すべき。
- **影響**: 悪意のある第三者サイトからも API にリクエストを送ることができる（認証は必要だがフィッシング攻撃のリスクが上がる）。
- **改善案**: `AllowOrigin: "'https://your-account.github.io'"` のように実際のフロントエンド URL に制限する。SAM の Parameter として管理すると環境ごとに変更しやすい。

---

#### W-8: Lambda Authorizer のキャッシュ TTL が 300 秒

- **ファイル**: `backend/template.yaml:51`
- **問題**: `ResultTtlInSeconds: 300` で認証結果を 5 分間キャッシュしている。ユーザーを Auth0 コンソールで無効化した場合でも 5 分間はアクセスが継続できる。
- **影響**: アカウント停止・パスワード変更後も最大 5 分間 API アクセスが可能。
- **改善案**: セキュリティ要件に応じてキャッシュを 0 に設定するか短縮する。個人・家族向けアプリであれば許容範囲だが、設計の意図として明示的にコメントを残すことを推奨。

---

#### W-9: hasID のチェックが req.PathParameters の存在確認になっていない

- **ファイル**: `functions/children/main.go:127`, `functions/allowance-types/main.go:103`, `functions/records/main.go:112-113`
- **問題**: `childID, hasID := req.PathParameters["id"]` で `hasID` を判定しているが、Go の map アクセスでは値が空文字でも `hasID = true` になる（キーが存在するが値が空の場合）。API Gateway はパスパラメータをマップに入れる際に空文字を入れることは通常ないが、テスト時に意図せずマップにキーだけ入れた場合に意図しない挙動になりうる。
- **影響**: ユニットテストで誤った挙動を検証するリスク。
- **改善案**: `hasID = childID != ""` も合わせて確認するか、コメントで意図を明示する。

---

### 🔵 Info（改善提案）

---

#### I-1: getChild / updateChild で findOwnedChild を 2 回呼ぶ可能性（updateChild）

- **ファイル**: `functions/children/main.go:248-299`
- **問題**: `updateChild` は `findOwnedChild` で子どもを取得してから、`col.UpdateOne` で更新し、さらに `calcBalanceForChild` でクエリを発行する。合計 3 回の MongoDB クエリが発生する。
- **改善案**: `FindOneAndUpdate` を使うと取得と更新を 1 クエリにまとめられる（パフォーマンスの軽微な改善）。

---

#### I-2: lib/db.go の sync.Once エラーが永続化する

- **ファイル**: `lib/db.go:23-32`
- **問題**: `clientOnce.Do` 内で `mongoErr` が設定されると、以降の `GetClient` 呼び出しでは `sync.Once` のため再試行されず、永久にエラーが返り続ける。Lambda が再起動するまで回復しない。
- **影響**: 一時的な接続エラーでも Lambda の再起動が必要になる（実際には Lambda は一定時間後にコンテナを破棄するため致命的ではないが、同一コンテナ内では回復しない）。
- **改善案**: エラー時に `sync.Once` をリセットするか、接続失敗を検知して再接続する仕組みを検討する。または MongoDB Atlas のフェイルオーバーに任せて許容する（コメントで意図を明示）。

---

#### I-3: records の listRecords で year が上限なしに受け付けられる

- **ファイル**: `functions/records/main.go:145-146`
- **問題**: `year < 1` のバリデーションのみで、上限がない。`year=9999` のような値でも通ってしまう。
- **改善案**: 現実的な範囲（例: 2000 〜 2100）でバリデーションする。

---

#### I-4: User 構造体に email フィールドがあるが、どこでも設定されていない

- **ファイル**: `models/user.go:13`, `functions/children/main.go:78-83`（他 Lambda も同様）
- **問題**: `resolveUser` で新規ユーザーを作成する際に `Email` フィールドを設定していない。スキーマ上は email を管理することになっているが、常に空文字で保存される。
- **影響**: 将来的に email を使う機能を追加したとき、既存ユーザーのデータが空のまま残る。
- **改善案**: Auth0 の JWT クレームに email が含まれている場合はそこから取得して保存する。または email フィールドを削除してスキーマをシンプルにする。

---

#### I-5: CalcBalance で int64 オーバーフローの考慮なし

- **ファイル**: `lib/balance.go:6-17`
- **問題**: 収入・支出を int64 で加減算しているが、理論上は `math.MaxInt64`（約 9.2 × 10^18 円）を超えるとオーバーフローする。おこずかいアプリの実用上は問題ないが、バリデーションと合わせて明示的に上限を設けると安全。
- **改善案**: 各 record の `Amount` に上限バリデーション（例: 10,000,000 円以下）を追加する。

---

#### I-6: AllowanceType の UserID が JSON レスポンスに含まれる

- **ファイル**: `models/allowance_type.go:11-17`
- **問題**: `AllowanceType.UserID` が `json:"user_id"` でシリアライズされ、listAllowanceTypes や createAllowanceType のレスポンスに含まれる。UserID は内部管理用フィールドであり、クライアントが必要とする情報ではない。Child モデルも同様（`json:"user_id"`）。
- **改善案**: `json:"-"` タグで除外するか、レスポンス専用の `AllowanceTypeResponse` 型を用意する。

---

#### I-7: template.yaml の Authorizer が TOKEN タイプではなく REQUEST タイプ

- **ファイル**: `backend/template.yaml:46-50`
- **問題**: `Identity.Headers: [Authorization]` を使った REQUEST タイプの Authorizer として設定されている。Authorization ヘッダーのみを識別子としており、他のリクエスト属性（IP、クエリパラメータなど）はキャッシュキーに含まれない。`ResultTtlInSeconds: 300` のキャッシュと組み合わせると、同じトークンを持つ異なるリソースへのリクエストがキャッシュされた結果を再利用する。現状は問題ないが、将来的にリソースベースのアクセス制御を追加したい場合に注意が必要。
- **改善案**: 現状の用途では適切。キャッシュ動作の意図をコメントで明示することを推奨。

---

## 総評

全体的に **基本的なセキュリティ設計（所有権チェック、JWT 検証）は適切に実装されている**。特に `findOwnedChild` / `findOwnedAllowanceType` での `user_id` 結合クエリによる所有権確認、Lambda Authorizer の分離と全エンドポイントへの適用は正しく設計されている。

一方で、以下の 3 点が最優先で対応が必要。

1. **C-1 の resolveUser 競合状態**: 初回ログイン時の 500 エラーにつながる。`upsert` でアトミックに修正する。
2. **C-2/C-3 のカスケード削除の非アトミック性**: MongoDB Atlas M0 でもトランザクション対応しているため、sessions/transactions の導入を検討する。
3. **C-5 の HTTP タイムアウト未設定**: 認証 Lambda のブロッキングにつながるため即時修正すべき。

W-1（resolveUser の重複）は C-1 の修正と合わせて `lib` パッケージへの集約を行うと、保守性と一貫性が大きく向上する。

W-3/W-4（N+1 問題）については現状の小規模ユースケース（子ども数人・records 数百件）では許容範囲だが、Aggregation Pipeline への切り替えは将来の技術的負債を減らす投資として検討に値する。
