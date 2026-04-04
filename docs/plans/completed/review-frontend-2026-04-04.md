# フロントエンド コードレビュー
レビュー日: 2026-04-03

## サマリー

全体として**非常に高品質なコードベース**であると評価する。Angular の最新プラクティス（Standalone Components、Signals、`inject()` 関数スタイル）が一貫して採用されており、Auth0 との統合も適切に実装されている。テストカバレッジも広く、意味のあるアサーションが書かれている。

主な発見事項は以下の通り。

**良好な点（継続すること）:**
- 全コンポーネントが Standalone Components で実装されている
- `signal()` / `computed()` の積極的な活用
- `take(1)` / `takeUntilDestroyed` によるサブスクリプション管理
- HttpClientTestingModule を使ったサービステスト
- 境界値テストを含む充実したフォームバリデーションテスト

**改善が必要な点（高優先度）:**
- `environment.ts` に Auth0 の `clientId` とドメインが平文でコミットされている（セキュリティリスク）
- `child-detail.component.ts` でネストしたサブスクリプション（unsubscribe 漏れの可能性）がある
- `LoginComponent` にテストが存在しない
- `authInterceptor` のテストが存在しない

---

## 良い点

### 1. Angular ベストプラクティスの遵守

- **全コンポーネントが `standalone: true`** を宣言している
- `inject()` 関数スタイルが一貫して使用されており、コンストラクタインジェクションと混在していない
- `signal()` / `computed()` が状態管理の中核として活用されており、Angular 17+ の推奨スタイルに沿っている
- Lazy loading（`loadComponent`）が全ルートに適用されており、初期バンドルサイズが最小化されている

### 2. ルーティング設計

`app.routes.ts` において `children/new` を `children/:id` より前に登録するといったルート優先順位への配慮がコメント付きで実装されており、保守性が高い。

### 3. 認証フロー

- `authInterceptor` が自社 API（`environment.apiBaseUrl`）のみにトークンを付与する設計になっており、Auth0 JWKS 等の外部エンドポイントに不必要なトークンが送出されない
- `authGuard` が未認証時に `loginWithRedirect()` を呼ぶのは Auth0 の推奨パターンに沿っている
- トークン取得失敗時のみログイン画面へリダイレクトする（API エラーは対象外）という設計判断がコメントで明示されており適切

### 4. RxJS サブスクリプション管理

- `LoginComponent` が `takeUntilDestroyed(this.destroyRef)` を使用してコンポーネント破棄時のサブスクリプションを自動解除している
- `RecordNewComponent` が `OnDestroy` + `Subject<void>` + `takeUntil(this.destroy$)` パターンで `valueChanges` を管理している
- ダイアログの `afterClosed()` に `take(1)` を使用して完結させている

### 5. 共通化・再利用性

- `child-form.factory.ts` でフォーム生成ロジックを `ChildNewComponent` と `ChildEditComponent` 間で共有しており、DRY 原則に従っている
- `ConfirmDialogComponent` が汎用的な確認ダイアログとして `shared/` に配置されている

### 6. テスト品質

- `ApiService` テストが全メソッドをカバーしており、境界値（空配列、オプショナルパラメータなし）も検証している
- `authGuard` テストが `TestBed.runInInjectionContext` を使って `CanActivateFn` を正しくテストしている
- `ChoreRegisterComponent` テストが `fakeAsync` / `tick` で非同期フローを適切に検証している
- フォームバリデーションテストで境界値（age: 0、19、base_allowance: 0、amount: 1）を確認している
- テストケース名が日本語で「何をテストするか」を明確に記述している

### 7. エラーハンドリングの一貫性

- 全コンポーネントでエラー時に `MatSnackBar` で日本語のメッセージを表示している
- `loading` シグナルがエラー時も `false` に戻る設計になっている

### 8. セキュリティ（XSS 対策）

- テンプレートで `{{ }}` バインディング（テキスト補間）のみを使用しており、`[innerHTML]` などの危険なバインディングは使用していない
- `innerHTML` や `bypassSecurityTrustHtml` の使用は一切ない

---

## 改善が必要な点

### 高優先度

#### H-1: Auth0 認証情報が `environment.ts` に平文でコミットされている

**ファイル:** `frontend/src/environments/environment.ts`

```typescript
// 現状（リスクあり）
auth0: {
  domain: 'dev-r25g73f2tlbnmbtt.us.auth0.com',
  clientId: 'oK3CRIKQX4xnVF0ZGOFd4zkLIuUqv091',
  audience: 'https://api.money-management.com',
},
```

Auth0 の `clientId` はフロントエンドでは公開されること自体は Auth0 の設計上許容されているが、`domain` と `clientId` がソースコードに直書きでコミットされているため、GitHub リポジトリが public になった場合や git 履歴が流出した場合にフィッシングやブルートフォース攻撃の標的になりうる。

`environment.prod.ts` は `__API_BASE_URL__` 等のプレースホルダーで CI/CD で置換する設計になっているが、`environment.ts`（開発用）はそうなっていない。

**推奨対応:**
- `environment.ts` も `.gitignore` に追加し、`environment.ts.example` をテンプレートとしてコミットする
- または開発用の値は public でない `dev-*` テナントであることを確認し、許容リスクとして文書化する

#### H-2: `ChildDetailComponent` のネストしたサブスクリプション

**ファイル:** `frontend/src/app/features/children/child-detail/child-detail.component.ts` (L159-169, L195-215)

```typescript
// 現状（ネストしたサブスクリプション）
dialogRef.afterClosed().pipe(take(1)).subscribe((confirmed: boolean) => {
  if (!confirmed) return;
  this.api.deleteRecord(this.childId, record.id).subscribe({  // 内側のサブスクリプション
    next: () => { this.loadRecords(); },
    error: () => { this.snackBar.open(...); },
  });
});
```

`take(1)` により外側の `afterClosed()` サブスクリプションは自動完了するが、内側の `deleteRecord().subscribe()` はコンポーネント破棄後も存続する可能性がある。同様のパターンが `paySelected()` メソッドにも存在する。

**推奨対応:** `switchMap` を使ってストリームを平坦化する。

```typescript
dialogRef.afterClosed().pipe(
  take(1),
  filter((confirmed): confirmed is true => confirmed === true),
  switchMap(() => this.api.deleteRecord(this.childId, record.id)),
).subscribe({
  next: () => { this.loadRecords(); },
  error: () => { this.snackBar.open('収支記録の削除に失敗しました', '閉じる', { duration: 3000 }); },
});
```

同様のリファクタリングが `paySelected()` にも必要。

#### H-3: `LoginComponent` のテストが存在しない

**ファイル:** `frontend/src/app/features/chore-register/login/login.component.ts`

`LoginComponent` は認証フローの入口として重要なコンポーネントだが、テストファイルが存在しない。特に以下の動作が未検証：
- 認証済みの場合に `/` へリダイレクトする
- ログインボタンクリック時に `loginWithRedirect` が呼ばれる
- `takeUntilDestroyed` によるサブスクリプション解除

#### H-4: `authInterceptor` のテストが存在しない

**ファイル:** `frontend/src/app/auth/auth.interceptor.ts`

インターセプターはすべての API リクエストのセキュリティに関わるが、テストが存在しない。テストが必要なケース：
- 自社 API URL へのリクエストに Bearer トークンが付与されること
- 外部 URL（Auth0 JWKS 等）へのリクエストにはトークンが付与されないこと
- トークン取得失敗時に `loginWithRedirect` が呼ばれること

---

### 中優先度

#### M-1: `ChoreRegisterComponent` の日付文字列生成ロジックが重複している

**ファイル:** `frontend/src/app/features/chore-register/chore-register.component.ts` (L83, L115)

`YYYY-MM-DD` 形式の日付文字列生成が `selectChild()` と `selectType()` の両方に実装されており、かつ `RecordNewComponent` にも `formatDate()` プライベートメソッドとして別実装がある。

`RecordNewComponent.formatDate()` のような共通ユーティリティ関数を `shared/` に切り出すことでコードの重複を排除できる。

#### M-2: `child-detail.component.ts` の `unpaidIncomeIds` computed の計算ロジックの懸念

**ファイル:** `frontend/src/app/features/children/child-detail/child-detail.component.ts` (L61-74)

```typescript
unpaidIncomeIds = computed(() => {
  const balance = this.child()?.balance ?? 0;
  const incomes = this.records()
    .filter(r => r.type === 'income')
    .sort((a, b) => b.created_at.localeCompare(a.created_at)); // 新しい順
  // ...
});
```

`records()` は「選択中の月」のレコードのみを保持しているが、`child()?.balance` は全期間の残高である。月を切り替えると `records()` が変わるため、未払いの判定が月をまたいで不整合になる可能性がある（例：先月の収入が今月の残高の文脈で未払い判定される）。

また `created_at` を文字列比較（`localeCompare`）でソートしているが、ISO 8601 形式であれば正常動作するものの、タイムゾーンの違いにより意図しない順序になるリスクがある。

**推奨対応:** この仕様が意図的なものであるかを確認し、意図的であればコメントで明示する。

#### M-3: `AllowanceTypeEditComponent` のフォームバリデーションテストが不足している

**ファイル:** `frontend/src/app/features/allowance-types/allowance-type-edit/allowance-type-edit.component.spec.ts`

`AllowanceTypeNewComponent` テストには境界値を含む詳細なバリデーションテストがあるが、`AllowanceTypeEditComponent` には同等のバリデーションテストがない。編集時も同じバリデーションルールが適用されるため、テストを追加すべきである。

#### M-4: `ChildDetailComponent` のテストで `paySelected` と `toggleRecord` が未テスト

**ファイル:** `frontend/src/app/features/children/child-detail/child-detail.component.spec.ts`

支払いモードに関する `paymentMode`、`toggleRecord`、`paySelected`、`isOverBalance`、`unpaidIncomeIds` のロジックが未テスト。これらは残高計算を伴う重要な機能であり、CLAUDE.md のカバレッジ目標（ビジネスロジック 90% 以上）に照らしても不足している。

#### M-5: `app.component.spec.ts` のテストが形式的である

**ファイル:** `frontend/src/app/app.component.spec.ts`

`AppComponent` はルーティングのシェルとしてシンプルなコンポーネントなので現状のテストで概ね問題はないが、`should render router-outlet` の検証は `<router-outlet>` タグの存在確認のみであり、テストとして最低限の意味しかない。AppComponent がほぼ空実装であることを考えると、現状のテストは許容範囲内。

#### M-6: `environment.ts` の開発環境 Auth0 設定について

**ファイル:** `frontend/src/environments/environment.ts`

開発用テナント（`dev-*`）の Auth0 設定が `environment.prod.ts` と異なり、プレースホルダー置換ではなく直書きになっている。これ自体は開発効率のためにある程度許容されるが、H-1 で述べた通りセキュリティリスクを孕む。

---

### 低優先度

#### L-1: `CreateChildRequest` と `UpdateChildRequest` の構造が同一

**ファイル:** `frontend/src/app/core/api.service.ts` (L32-42)

```typescript
export interface CreateChildRequest {
  name: string;
  age: number;
  base_allowance: number;
}

export interface UpdateChildRequest {
  name: string;
  age: number;
  base_allowance: number;
}
```

フィールドが完全に同一。将来的に diverge する可能性があるのであれば現状維持でよいが、そうでなければ `type ChildMutationRequest = CreateChildRequest` の alias か、1つの interface に統合してもよい。同様に `CreateAllowanceTypeRequest` と `UpdateAllowanceTypeRequest` も同一構造。

#### L-2: `ChoreRegisterComponent` の `selectChild` でのエラー処理が情報を隠蔽している

**ファイル:** `frontend/src/app/features/chore-register/chore-register.component.ts` (L93-97)

```typescript
error: () => {
  this.todayDoneTypeIds.set(new Set());
  this.loading.set(false);
  this.step.set('select-type');  // エラーでも次のステップへ進む
},
```

`getRecords` のエラー時にユーザーへの通知なしにステップ2へ進む設計になっている。今日の記録が取得できなかった場合、すでに登録済みのお手伝いを重複して登録できてしまう可能性がある。SnackBar で「通信エラーが発生しました。重複登録に注意してください」等の警告を表示する方が親切。

#### L-3: `OnPush` 変更検知戦略が未採用

全コンポーネントでデフォルトの変更検知戦略（`Default`）が使用されている。`signal()` を多用しているため `OnPush` への移行は比較的容易であり、パフォーマンス改善が期待できる。ただし、このプロジェクトの規模では実影響は小さいため低優先度とする。

#### L-4: `ChildDetailComponent` の `loading` オーバーレイの位置

**ファイル:** `frontend/src/app/features/children/child-detail/child-detail.component.html` (L96-100)

```html
@if (loading()) {
  <div class="loading-overlay">
    <mat-spinner></mat-spinner>
  </div>
}
```

他のコンポーネント（`DashboardComponent`、`ChoreRegisterComponent`）ではローディングオーバーレイをテンプレートの**先頭**に配置しているのに対し、`ChildDetailComponent` のみ**末尾**に配置している。機能的な差異はないが、一貫性の観点で先頭に移動すると可読性が向上する。

#### L-5: `record-new.component.ts` の `childId` プロパティが `public` になっている

**ファイル:** `frontend/src/app/features/children/record-new/record-new.component.ts` (L54)

```typescript
/** ルートパラメータから取得した子どもID */
childId = '';  // private でよいはず
```

テストから `component.childId` でアクセスされているため意図的かもしれないが、テンプレートで使用されていない場合は `private` にする方がカプセル化の観点で望ましい。ただしテストでの検証用途が主であれば、テスト側を修正して `private` にすることを推奨。

---

## ファイル別レビュー詳細

| ファイル | 評価 | 主な所見 |
|---------|------|----------|
| `app.config.ts` | 良好 | Auth0 設定、Service Worker 設定が適切。`redirect_uri` の本番/開発分岐も正確 |
| `app.routes.ts` | 良好 | Lazy loading 全適用、ルート優先順位コメント付き |
| `auth/auth.guard.ts` | 良好 | Auth0 推奨パターンに準拠 |
| `auth/auth.guard.spec.ts` | 良好 | `TestBed.runInInjectionContext` の適切な使用、認証/未認証の両ケースをカバー |
| `auth/auth.interceptor.ts` | 良好（要テスト追加） | 自社 API のみトークン付与、エラーハンドリング適切。テスト不在が高優先度課題 |
| `core/api.service.ts` | 良好 | 全エンドポイント実装、型定義が整理されている。`CreateChildRequest` と `UpdateChildRequest` の重複は低優先度 |
| `core/api.service.spec.ts` | 非常に良好 | 全メソッドテスト済み、境界値（空配列、オプショナルなし）も確認。`httpMock.verify()` でリーク検出 |
| `shared/confirm-dialog/confirm-dialog.component.ts` | 良好 | 汎用ダイアログ、`inject()` スタイル統一 |
| `shared/forms/child-form.factory.ts` | 良好 | フォームロジックの共通化、バリデーションルールが1箇所に集約 |
| `features/chore-register/login/login.component.ts` | 良好（要テスト追加） | `takeUntilDestroyed` 使用。テストファイルが存在しない |
| `features/chore-register/chore-register.component.ts` | 良好 | `forkJoin` の適切な使用、ステップ管理のシグナル活用。日付文字列生成の重複あり |
| `features/chore-register/chore-register.component.spec.ts` | 非常に良好 | `fakeAsync`/`tick` 使用、全フロー検証、エラーハンドリングも確認 |
| `features/dashboard/dashboard.component.ts` | 良好 | シンプルで読みやすい。`getAvatarColor`/`getInitial` はヘルパー関数として適切 |
| `features/dashboard/dashboard.component.spec.ts` | 良好 | 通常/エラー/空状態の3ケースをカバー |
| `features/children/child-detail/child-detail.component.ts` | 要改善 | ネストしたサブスクリプション（H-2）、`unpaidIncomeIds` の月またぎ計算懸念（M-2） |
| `features/children/child-detail/child-detail.component.spec.ts` | 中程度 | 初期化・月フィルタ・削除・遷移をカバーするが支払い機能が未テスト（M-4） |
| `features/children/child-new/child-new.component.ts` | 良好 | シンプルで読みやすい |
| `features/children/child-new/child-new.component.spec.ts` | 非常に良好 | バリデーション境界値テスト充実、`Subject` で通信中 loading を検証 |
| `features/children/child-edit/child-edit.component.ts` | 良好 | `take(1)` でダイアログを管理 |
| `features/children/child-edit/child-edit.component.spec.ts` | 良好 | 削除ダイアログのキャンセル/確認を両方テスト |
| `features/children/record-new/record-new.component.ts` | 良好 | `takeUntil` で `valueChanges` を管理。`childId` の可視性が低優先度課題 |
| `features/children/record-new/record-new.component.spec.ts` | 非常に良好 | 自動入力・日付形式・allowance_type_id の有無など細かいケースを網羅 |
| `features/allowance-types/allowance-type-list/` | 良好 | テスト含め一貫した実装 |
| `features/allowance-types/allowance-type-new/` | 良好 | バリデーションテスト充実 |
| `features/allowance-types/allowance-type-edit/` | 良好（要改善） | バリデーションテストが不足（M-3） |
| `environments/environment.ts` | 要改善 | Auth0 認証情報の平文コミット（H-1） |
| `environments/environment.prod.ts` | 良好 | プレースホルダー置換方式で CI/CD と適切に連携 |

---

## 推奨アクション

優先度順に記載する。

### 即時対応（高優先度）

1. **`environment.ts` の Auth0 認証情報を git 履歴から保護する（H-1）**
   - `environment.ts` を `.gitignore` に追加し、`environment.ts.example` を作成する
   - または、現状のリスクを文書化してチーム内で合意する

2. **`authInterceptor` のテストを追加する（H-4）**
   - 自社 API URL へのリクエストにトークンが付与されること
   - 外部 URL（Auth0 JWKS 等）にはトークンが付与されないこと
   - トークン取得失敗時の `loginWithRedirect` 呼び出し

3. **`LoginComponent` のテストを追加する（H-3）**
   - 認証済み時の `/` へのリダイレクト
   - ログインボタンクリック時の `loginWithRedirect` 呼び出し

4. **`ChildDetailComponent` のネストしたサブスクリプションを `switchMap` でリファクタリングする（H-2）**
   - `deleteRecord()` 内
   - `paySelected()` 内

### 次スプリント（中優先度）

5. **`ChildDetailComponent` の支払い機能テストを追加する（M-4）**
   - `toggleRecord`、`paySelected`、`isOverBalance`、`unpaidIncomeIds` の computed ロジック

6. **`AllowanceTypeEditComponent` にバリデーションテストを追加する（M-3）**
   - `AllowanceTypeNewComponent` と同等のテストを追加

7. **日付文字列生成ロジックを `shared/` に集約する（M-1）**
   - `ChoreRegisterComponent` と `RecordNewComponent` で重複しているロジックを `shared/utils/date.ts` 等に切り出す

### 余裕があれば（低優先度）

8. **`CreateChildRequest` と `UpdateChildRequest` の統合を検討する（L-1）**

9. **`ChoreRegisterComponent.selectChild` のエラー時にユーザー通知を追加する（L-2）**

10. **`OnPush` 変更検知戦略への移行を検討する（L-3）**

11. **`record-new.component.ts` の `childId` を `private` にする（L-5）**

12. **`ChildDetailComponent` のローディングオーバーレイ位置をテンプレート先頭に移動する（L-4）**
