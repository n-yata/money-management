# 開発ナレッジ・反省点

このプロジェクトの開発を通じて得た反省点と、今後気をつけるべき知見をまとめる。

---

## 1. APIレスポンス形式の不一致

### 問題
設計書では `GET /api/v1/children` のレスポンスを直接配列 `[...]` と定義していたが、
バックエンドの実装は `{"data": [...]}` のラッパー形式で返していた。
フロントエンドが直接配列を期待して実装されていたため、`newCollection[Symbol.iterator] is not a function` エラーが発生した。

### 対応
フロントエンドの `ApiService` 全メソッドに `.pipe(map(res => res.data))` を追加して展開するよう修正した。

### 今後の対策
- **バックエンド実装開始前に、フロントエンドとバックエンドのレスポンス形式を明示的に合意する**
- ラッパー形式（`{"data": ...}`）か直接形式かを設計書に明記し、両者で統一する
- 統合テストまたはAPIコントラクトテストを導入して、形式のズレを早期検出する

---

## 2. 日時のタイムゾーン（UTC vs JST）

### 問題
日付登録時に `new Date().toISOString().split('T')[0]` を使用していたため、UTC基準の日付が使われていた。
JST（UTC+9）では0:00〜9:00の間に登録すると、日付が1日ずれて登録された。

### 対応
ローカル時刻を使うよう修正した：
```typescript
const d = new Date();
const today = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
```

### 今後の対策
- **日付の取得には `toISOString()` を使わない。** UTCとローカル時刻の差を常に意識する
- 日本向けアプリでは日付処理はローカル時刻（JST）を基準にする
- バックエンドが `date` フィールドをISO 8601 datetimeで返す場合、フロントエンドは `DatePipe` でタイムゾーンを考慮してフォーマットする

---

## 3. SAM local start-api の Windows 起動問題

### 問題
`samconfig.toml` に `disable_authorizer = true` を設定していたが、`sam local start-api` を実行すると
Lambda Authorizer が無効化されず 401 が返された。また Lambda コンテナが起動しない場合は 502 が返された。

### 原因
- `samconfig.toml` の `disable_authorizer = true` が環境によって適用されないことがあった
- Windows環境でのDocker networking（Lambda コンテナ → SAM ホスト間通信）が不安定

### 対応
`samconfig.toml` に頼らず、CLIオプションを明示的に指定して起動する：
```bash
cd backend
sam.cmd local start-api \
  --disable-authorizer \
  --port 3000 \
  --env-vars env.json \
  --parameter-overrides "LocalAuth0Sub=local-dev-user ..."
```

### 今後の対策
- **起動コマンドを `Makefile` または `scripts/` に明示的に記載する**（`samconfig.toml` だけに依存しない）
- Windows環境での `sam local` は Docker Desktop の設定やバージョンによって挙動が変わるため、起動確認手順をREADMEに記載する
- ローカル開発用の起動スクリプトを用意する

---

## 4. Auth0 Interceptor の catchError 配置

### 問題
`auth.interceptor.ts` の `catchError` を `switchMap` の後（HTTPリクエスト全体）に適用していたため、
APIがエラーを返すたびに `loginWithRedirect()` が呼ばれ、無限リダイレクトループが発生した。

### 対応
`catchError` を `getAccessTokenSilently()` のみに限定した：
```typescript
return auth.getAccessTokenSilently().pipe(
  catchError(err => {
    auth.loginWithRedirect();  // トークン取得失敗時のみリダイレクト
    return throwError(() => err);
  }),
  switchMap(token => next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }))),
);
```

### 今後の対策
- **RxJS の `catchError` は適用範囲を最小限にする。** パイプの全体に適用するのではなく、エラーハンドリングが必要な操作のみに適用する
- インターセプターのエラーハンドリングは特に副作用（リダイレクト等）を伴うため、慎重に設計する

---

## 5. テストモックの型整合性

### 問題
インターフェースに新しいフィールド（`created_at`）を追加した際、テストのモックオブジェクトが古いままだったため
TypeScriptのコンパイルエラーが発生し、テストが実行できない状態になった。

### 対応
テストの `mockRecord` などのモックオブジェクトに `created_at` フィールドを追加した。

### 今後の対策
- **インターフェースを変更したら、関連するテストファイルも同時に更新する**
- TypeScriptの厳格モード（`strict: true`）を有効にして、型の不整合をコンパイル時に検出できるようにする
- `Partial<T>` を使ったモック生成ヘルパーを用意すると保守しやすくなる

---

## 6. ローカル開発環境と本番の認証バイパス

### 設計
ローカル開発では `--disable-authorizer` でLambda Authorizerをスキップする。
その代わり、Lambda関数内で `LOCAL_AUTH0_SUB` 環境変数から認証ユーザーを識別する。

```go
func GetAuthSub(request events.APIGatewayProxyRequest) (string, bool) {
    sub, ok := request.RequestContext.Authorizer["sub"].(string)
    if !ok || sub == "" {
        if localSub := os.Getenv("LOCAL_AUTH0_SUB"); localSub != "" {
            return localSub, true
        }
        return "", false
    }
    return sub, true
}
```

### 注意点
- `LOCAL_AUTH0_SUB` は必ず SAM テンプレートのパラメータ経由で渡す（`template.yaml` に定義済み）
- `LOCAL_AUTH0_SUB` は本番環境では空文字列または未設定にすること
- `env.json` や `samconfig.toml` は `.gitignore` に追加してリポジトリにコミットしない

---

## 7. 支払い済みレコードの判定ロジック

### 課題
収支記録にどのincomeレコードが支払われたかを示す `paid` フラグがないため、
「支払い済みか否か」を正確に判定できない。

### 現在の実装（暫定）
`child.balance`（全期間残高 = 未払い総額）を基準に、新しい順に残高分だけのincomeレコードを「未払い」とみなす：
```typescript
unpaidIncomeIds = computed(() => {
  const balance = this.child()?.balance ?? 0;
  const incomes = this.records()
    .filter(r => r.type === 'income')
    .sort((a, b) => b.date.localeCompare(a.date));
  const unpaid = new Set<string>();
  let covered = 0;
  for (const r of incomes) {
    if (covered >= balance) break;
    unpaid.add(r.id);
    covered += r.amount;
  }
  return unpaid;
});
```

### 制限
- 表示中の月のレコードのみ参照するため、他の月の未払い残高がある場合は不正確になる可能性がある
- incomeレコードとexpenseレコードを明示的に紐づけていないため、「どのお手伝いに対して払ったか」が追跡できない

### 今後の改善案
- recordsコレクションに `paid: bool` フィールドを追加し、支払い時に更新する
- または支払いレコードに紐づくincomeレコードIDリストを保持する

---

## 8. 設計書と実装の乖離防止

### 問題
設計書（`api-design.md`）のDELETEレスポンスが `204 No Content` と定義されていたが、
実際の実装は `200 OK` + `{"data": null}` だった。

### 今後の対策
- **実装後に設計書を必ず更新する**（または実装前に設計書を確定させる）
- 設計書はコードと同じリポジトリに置き、実装変更と同じPRでレビューする
- 将来的にはOpenAPI（Swagger）定義を導入し、設計書と実装の自動検証を行う

---

## まとめ

| カテゴリ | 重要度 | 対策 |
|---------|--------|------|
| APIレスポンス形式の合意 | ★★★ | 設計段階でフロント・バック間で明示的に確認 |
| 日時タイムゾーン | ★★★ | `toISOString()` は使わない、ローカル時刻を使う |
| SAM local 起動コマンド | ★★ | Makefile/スクリプトに明記 |
| Auth Interceptor の catchError | ★★★ | 適用範囲を最小限に |
| テストモックの型整合 | ★★ | インターフェース変更時は同時にテストも更新 |
| 設計書と実装の同期 | ★★ | 実装変更と同じタイミングで設計書を更新 |
