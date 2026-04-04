# テスト品質レビュー
レビュー日: 2026-04-04

---

## サマリー

バックエンド（Go）・フロントエンド（Angular）ともに、CLAUDE.mdで定めたテスト方針に対して高い水準で実装されている。テーブル駆動テスト・testcontainers統合テスト・`fakeAsync/tick`の適切な使用など、方針通りの実装が確認できた。

一方で、いくつかの改善点が存在する。特に以下2点が中優先度以上の課題となっている。

1. `lib/child.go`（`FindOwnedChild`）と `lib/indexes.go`（`EnsureIndexes`）に対するユニットテストが存在しない
2. `shared/utils/date.ts`（`formatDate`）と `shared/forms/child-form.factory.ts`（`createChildForm`）のユニットテストが存在しない
3. `authGuard` のテストケースが4件と少なく、目標90%に対して不確実（エッジケース未カバー）
4. `records/handler_test.go` の `TestMain` がレプリカセットなしで起動しており、children/handler_test.go との非対称性がある

---

## 良い点

### バックエンド

- **テーブル駆動テストの徹底**: `TestCalcBalance`（8ケース）、`TestGetAuthSub`（6ケース）、`TestChildInputValidate`（12ケース）、`TestRecordInputValidate`（13ケース）、`TestAllowanceTypeInputValidate`（8ケース）など、すべてのバリデーションロジックがテーブル駆動で記述されている。

- **境界値テストの網羅**: 名前20文字OK・21文字NG、年齢1・18境界、金額1円・0円・マイナス・1,000万円超過など、境界値が確実に押さえられている。

- **認証（JWT Authorizer）の高品質テスト**: 本物の RSA 鍵ペアを動的生成し、有効トークン・期限切れ・署名不正・subなし・ヘッダーなし・小文字ヘッダーと幅広いケースをテストしている。JWKS エンドポイントを `httptest.NewTLSServer` でモックし、実際のHTTPクライアントを差し替える手法は品質が高い。

- **統合テストの独立性確保**: `newTestDB` がテストごとに一意なDB名（UnixNano）を使い、テスト間の状態汚染を完全に防いでいる。

- **所有権チェックの徹底**: すべてのCRUDハンドラで「他ユーザーのリソースへのアクセスは404を返す」ことを統合テストで検証している。

- **カスケード削除のテスト**: `deleteChild` 時に関連する `records` が削除されること、`deleteAllowanceType` 時に関連 `records` の `allowance_type_id` が null になることを実際のDBクエリで確認している。

- **ビジネスルールのテスト**: 「同じ日・同じ種類の重複登録は409 Conflict」「別日なら同じ種類でも登録可能」「種類なしの手動記録は同日複数可能」という業務ルールが統合テストで担保されている。

- **`middleware/auth.go` の環境変数依存テスト**: `t.Setenv` によりテスト後自動リセットされる形で `ENVIRONMENT=local` バイパスの安全性を検証している。

### フロントエンド

- **`fakeAsync/tick` の適切な使用**: すべての非同期処理（Observable購読・RxJSオペレーター）に `fakeAsync/tick` が使われており、実時間待ちのないテストになっている。

- **`HttpClientTestingModule` による完全なモック**: `api.service.spec.ts` は全エンドポイント（GET/POST/PUT/DELETE × 全リソース）をカバーし、リクエストのメソッド・URL・ボディ・クエリパラメータをすべて検証している。

- **`authInterceptor` の重要パスの検証**: 自社APIへのBearerヘッダー付与、外部URLへのヘッダー非付与、トークン取得失敗時の `loginWithRedirect` 呼び出しとエラー伝播を確認している。

- **ローディング状態の検証**: `Subject` を使って非同期レスポンスを制御し、API通信中の `loading() === true`、完了後の `false` を正確にテストしている（`ChildNewComponent`, `AllowanceTypeNewComponent`, `RecordNewComponent`）。

- **テンプレートの実質的な検証**: 残高の数値表示（`1,200`）・空メッセージ・子どもカード枚数など、UIの出力を検証している。ただし Angular Material コンポーネント自体のテストは避けている（方針通り）。

- **コンポーネント破棄後のサブスクリプション解除テスト**: `LoginComponent` で `takeUntilDestroyed` が正しく動作することを `fixture.destroy()` 後のシグナル発火で確認している。

- **`ChoreRegisterComponent` の業務フロー検証**: ステップ遷移（子ども選択→種類選択→完了）・当日登録済み種類の重複ガード・`todayDoneTypeIds` の正確な設定など、コアビジネスロジックをカバーしている。

- **`ChildDetailComponent` の支払いモードテスト**: `unpaidIncomeIds`（残高内に収まるincomeの算出）・`isOverBalance`（選択合計と残高の比較）・`paySelected`（ダイアログ確認後のexpense記録作成）を詳細にテストしている。

---

## 改善が必要な点

### 高優先度

#### H-1: `lib/child.go` の `FindOwnedChild` にユニットテストがない

`FindOwnedChild` はすべての子ども関連ハンドラで呼ばれる所有権チェック関数であり、認証バイパスリスクに直結する。統合テスト（`handler_test.go`）で間接的にはカバーされているが、単体テストが存在しない。

テストすべきケース:
- 空文字IDは `(_, false, nil)` を返す
- 不正なObjectID形式は `(_, false, nil)` を返す
- 他ユーザーの子どもIDは `(_, false, nil)` を返す（所有権NG）
- 自ユーザーの子どもIDは `(child, true, nil)` を返す（正常）

#### H-2: `records/handler_test.go` の `TestMain` がレプリカセットなしで起動

`children/handler_test.go` と `allowance-types/handler_test.go` は `mongodb.WithReplicaSet("rs0")` を使ってシングルノードレプリカセットで起動しているが、`records/handler_test.go` の `TestMain` はそれを使っていない。

`records` のハンドラがMongoDBトランザクションを使用する場合、この設定の非対称性がCI環境での誤検知を引き起こすリスクがある。実装コードを確認して統一するか、意図的な省略であればコメントで理由を説明すべきである。

### 中優先度

#### M-1: `authGuard` のテストが4件のみ（目標90%に対して不十分な可能性）

現在のテストケース:
- 認証済み → true を返す
- 認証済み → loginWithRedirect を呼ばない
- 未認証 → false を返す
- 未認証 → loginWithRedirect を呼ぶ

不足しているケース:
- `loginWithRedirect` に渡される `appState` オプションの検証（リダイレクト先URLが正しく渡されるか）
- Observable がエラーを流した場合の動作

#### M-2: `shared/utils/date.ts` の `formatDate` にテストがない

`formatDate` は日付フォーマット変換を担う共通ユーティリティで、`RecordNewComponent`・`ChoreRegisterComponent` などで使用されている。シンプルな関数だが、月・日のゼロパディングのバグはAPIバリデーションエラーに直結するため、ユニットテストで担保すべきである。

テストすべきケース:
- 1桁の月・日（例: 2026-01-05）がゼロパディングされること
- 12月31日が正しくフォーマットされること（境界値）

#### M-3: `shared/forms/child-form.factory.ts` の `createChildForm` にテストがない

`ChildNewComponent` と `ChildEditComponent` で共通使用するフォームファクトリ関数。各コンポーネントのテストでバリデーションは間接的にカバーされているが、ファクトリ関数自体のユニットテストがない。リファクタリング時の退行検知として、ファクトリのテストがあると安全性が高まる。

#### M-4: `lib/indexes.go` の `EnsureIndexes` にテストがない

インデックス設定は一度しか呼ばれないが、インデックスが欠けると本番での検索性能・unique制約が失われる。統合テスト環境で実際にインデックスが作成されることを確認するテストがあると望ましい。

#### M-5: `ChildEditComponent` のフォームバリデーションテストが不足

`child-edit.component.spec.ts` には「保存処理」「キャンセル処理」「削除処理」のテストはあるが、`child-new.component.spec.ts` にある「フォームバリデーション」のテストブロックが存在しない。`createChildForm` ファクトリを共用しているため同じバリデーションが適用されているはずだが、編集画面に固有の「フォームが invalid のとき save() を呼んでも updateChild が呼ばれないこと」のテストも不足している。

### 低優先度

#### L-1: `app.component.spec.ts` のテストが最小限

現在は「生成できる」「router-outletが存在する」の2件のみ。`AppComponent` がルーティング設定以外のロジックを持たないため現状で妥当だが、将来機能が追加された際にテストを忘れるリスクがある。

#### L-2: `lib/response_test.go` で `JSONResponse` のCORSヘッダー以外のヘッダー検証が欠落

`Access-Control-Allow-Methods` や `Access-Control-Allow-Headers` がレスポンスに含まれる場合、それらの検証がない。現在の実装範囲では問題ないが、CORSポリシー変更時に退行検知できるテストがあると安全。

#### L-3: `getPublicKey` のキャッシュ有効期限テストがない

`authorizer/main_test.go` で `getPublicKey` のキャッシュ機能は間接的に使われているが、キャッシュの有効期限切れ後に再フェッチが行われるかどうかをテストするケースがない。セキュリティ上、古いJWKSキャッシュが使われ続けるシナリオは確認しておく価値がある。

---

## カバレッジ評価（目標との対比）

| 対象 | 目標 | 評価 | 根拠 |
|------|------|------|------|
| ビジネスロジック（残高計算・バリデーション） | 90%以上 | **達成見込み** | `CalcBalance`は8ケース（空・収入のみ・支出のみ・組合せ・不明typeなど）で網羅。各バリデーション関数も境界値含む全ケースをカバー |
| バックエンド全体（Go） | 70%以上 | **概ね達成、要確認** | `lib/child.go`（`FindOwnedChild`）・`lib/indexes.go`・`lib/db.go` は直接テストなし。統合テストで間接カバーされているが、測定結果の確認が必要 |
| JWT検証ミドルウェア（AuthGuard含む） | 90%以上 | **バックエンド達成、フロントエンド要補強** | `authorizer/main_test.go` は RSA 鍵・JWKS・handler の全パスをカバー。`middleware/auth_test.go` も環境変数分岐を含む全ケースをカバー。フロントエンドの `authGuard` は4ケースのみで `appState` 検証が不足 |
| フロントエンド Service（ApiService） | 80%以上 | **達成** | 全メソッド（11メソッド）・空配列ケース・allowance_type_idなしなど、実質100%のパスをカバー |
| フロントエンド Component | 60%以上 | **達成** | 全9コンポーネントにテストファイルが存在。初期化・バリデーション・API成功/失敗・画面遷移・ローディング状態を網羅。`ChildDetailComponent` は支払いモードの複雑なcomputed値まで検証 |

---

## 推奨アクション

### 即対応（高優先度）

1. **`lib/child.go` のユニットテスト追加**
   - ファイル: `backend/src/lib/child_test.go`（新規作成）
   - テスト: 空ID・不正ID・他ユーザー所有・正常取得の4ケース
   - testcontainersを使った統合テストとして実装する

2. **`records/handler_test.go` の `TestMain` をレプリカセット起動に統一するか、意図を明確化**
   - `children` と `allowance-types` の `TestMain` は `WithReplicaSet("rs0")` を使用しているのに `records` は使っていない非対称性を解消する

### 早期対応（中優先度）

3. **`shared/utils/date.ts` の `formatDate` ユニットテスト追加**
   - ファイル: `frontend/src/app/shared/utils/date.spec.ts`（新規作成）
   - テスト: 1桁月日のゼロパディング、境界値（12月31日、1月1日）

4. **`authGuard` に `appState` 検証テスト追加**
   - `loginWithRedirect` に渡される引数（`{ appState: { target: '/dashboard' } }` 等）を検証するケースを追加する

5. **`ChildEditComponent` のフォームが invalid 時の `save()` 無効化テスト追加**
   - `child-edit.component.spec.ts` に「フォームが invalid のとき save() を呼んでも updateChild が呼ばれないこと」のケースを追加する

### 機会があれば対応（低優先度）

6. **`shared/forms/child-form.factory.ts` のユニットテスト追加**
7. **`getPublicKey` のキャッシュ有効期限切れ再フェッチテスト**
8. **`lib/indexes.go` の統合テスト**（インデックスが実際に作成されることの確認）
