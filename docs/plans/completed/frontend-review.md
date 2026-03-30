# フロントエンドソースレビュー

レビュー日: 2026-03-28
対象: `frontend/src/` 配下の全ソースファイル
Angular バージョン: 19.x（Standalone Components + Signals）

---

## サマリー

全体的にAngularの最新プラクティス（Standalone Components、Signals、functional guard/interceptor）を適切に採用している。
コードの構造は統一されており、各コンポーネントの責務も明確に分離されている。
テストは意味のあるアサーションで書かれており、境界値・異常系のカバレッジも十分。

ただし、以下の点で修正・改善が必要な問題が確認された。

**主要な問題点（優先度順）:**

1. `LoginComponent` の `isAuthenticated$` サブスクリプションが未管理（メモリリーク）
2. `auth.interceptor.ts` がすべてのHTTPリクエスト（外部URLを含む）にトークンを付与するリスク
3. `ChildDetailComponent` のネイティブ `<select>` 使用が不整合（他コンポーネントは `mat-select`）
4. `ChoreRegisterComponent` の並行APIロードで `loading` フラグが正確に制御されていない
5. `DialogRef` の `afterClosed` サブスクリプションが未管理（複数コンポーネントで共通問題）

---

## 重要度別の指摘事項

### 🔴 High（要対応）

---

#### H-1: LoginComponent の isAuthenticated$ サブスクリプション未管理

**ファイル:** `src/app/features/chore-register/login/login.component.ts` L28

**問題の説明:**

`ngOnInit` 内で `isAuthenticated$` を subscribe しているが、`takeUntil` や `AsyncPipe` を使った unsubscribe 処理が存在しない。
`LoginComponent` が破棄された後もサブスクリプションが生き続け、`router.navigate` が意図せず呼ばれる可能性がある。

```typescript
// 問題のあるコード
ngOnInit(): void {
  this.auth.isAuthenticated$.subscribe(isAuthenticated => {
    if (isAuthenticated) {
      this.router.navigate(['/']);
    }
  });
}
```

**改善案:**

```typescript
// 方法1: takeUntilDestroyed（Angular 16以降の推奨）
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

export class LoginComponent implements OnInit {
  private destroyRef = inject(DestroyRef);

  ngOnInit(): void {
    this.auth.isAuthenticated$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(isAuthenticated => {
        if (isAuthenticated) {
          this.router.navigate(['/']);
        }
      });
  }
}

// 方法2: AsyncPipeとテンプレート側で制御（よりAngularらしい実装）
// auth.isAuthenticated$ | async を使ってテンプレート側でリダイレクトを制御
```

---

#### H-2: auth.interceptor.ts がすべてのリクエストにトークンを付与する

**ファイル:** `src/app/auth/auth.interceptor.ts` L10-21

**問題の説明:**

現時点ではAPIリクエストは自社バックエンドのみだが、将来的にサードパーティAPIへのリクエスト（例: CDN、外部ログサービス、Auth0の管理API等）が追加された場合、アクセストークンが不要な宛先に送信されるリスクがある。
また、`getAccessTokenSilently` がエラーをスローした場合（セッション切れ等）のエラーハンドリングが実装されていない。エラーが呼び出し元に伝播するため、ユーザーはエラーメッセージを見るだけでログイン画面に戻れない。

```typescript
// 現状: すべてのリクエストにトークン付与
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  return auth.getAccessTokenSilently().pipe(
    switchMap(token => { ... })
  );
};
```

**改善案:**

```typescript
import { catchError, switchMap } from 'rxjs/operators';
import { throwError } from 'rxjs';
import { Router } from '@angular/router';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // 自社バックエンドへのリクエストのみトークンを付与する
  if (!req.url.startsWith(environment.apiBaseUrl)) {
    return next(req);
  }

  return auth.getAccessTokenSilently().pipe(
    switchMap(token => {
      const authReq = req.clone({
        setHeaders: { Authorization: `Bearer ${token}` },
      });
      return next(authReq);
    }),
    catchError(err => {
      // トークン取得失敗時はログイン画面にリダイレクト
      auth.loginWithRedirect();
      return throwError(() => err);
    })
  );
};
```

---

#### H-3: DialogRef.afterClosed() サブスクリプションが未管理

**ファイル:**
- `src/app/features/children/child-detail/child-detail.component.ts` L120
- `src/app/features/children/child-edit/child-edit.component.ts` L98
- `src/app/features/allowance-types/allowance-type-edit/allowance-type-edit.component.ts` L96

**問題の説明:**

`dialogRef.afterClosed()` のサブスクリプションが明示的にアンサブスクライブされていない。
`afterClosed()` は通常1回しか emit しないため実害は小さいが、コンポーネント破棄後に予期せぬ副作用が起きるリスクがゼロではない。
また、3コンポーネントで同一パターンが繰り返されており、一箇所でも問題が顕在化すると他も同様になる。

```typescript
// 3コンポーネントで同じパターン
dialogRef.afterClosed().subscribe((confirmed: boolean) => {
  if (!confirmed) return;
  // ... API呼び出し
});
```

**改善案:**

```typescript
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

// DestroyRefを使って明示的に管理する
dialogRef.afterClosed()
  .pipe(take(1)) // afterClosed は1回限り emit するが明示する
  .subscribe((confirmed: boolean) => {
    if (!confirmed) return;
    // ... API呼び出し
  });
```

---

### 🟡 Medium（推奨対応）

---

#### M-1: ChoreRegisterComponent の loading フラグが並行ロードで正確に制御されない

**ファイル:** `src/app/features/chore-register/chore-register.component.ts` L53-82

**問題の説明:**

`loadChildren()` と `loadAllowanceTypes()` を並行して呼び出しているが、`loading` フラグは `loadChildren()` のみで制御されている。
`loadAllowanceTypes()` のエラー時には `loading` が `true` のまま残るケース（`loadChildren` 成功後に `loadAllowanceTypes` がエラーになった場合 `loading` はすでに `false` になっているため、この特定ケースは発生しないが）、設計として脆弱。
また、`loadAllowanceTypes()` の通信中は `loading` が `false` になっている可能性があり、スピナーが早期に消えることがある。

```typescript
private loadChildren(): void {
  this.loading.set(true);  // ← loadChildren だけが loading を制御
  // ...
}

private loadAllowanceTypes(): void {
  // loading の制御なし
  this.api.getAllowanceTypes().subscribe({ ... });
}
```

**改善案:**

`forkJoin` を使って2つのAPIを並行取得し、`loading` を一元管理する。

```typescript
ngOnInit(): void {
  this.loading.set(true);
  forkJoin({
    children: this.api.getChildren(),
    allowanceTypes: this.api.getAllowanceTypes(),
  }).subscribe({
    next: ({ children, allowanceTypes }) => {
      this.children.set(children);
      this.allowanceTypes.set(allowanceTypes);
      this.loading.set(false);
    },
    error: (err) => {
      this.snackBar.open('データの取得に失敗しました', '閉じる', { duration: 3000 });
      this.loading.set(false);
    },
  });
}
```

---

#### M-2: child-detail.component.html でネイティブ `<select>` を使用している

**ファイル:** `src/app/features/children/child-detail/child-detail.component.html` L25-29

**問題の説明:**

月フィルタのセレクトボックスにネイティブの `<select>` 要素を使用しており、他のフォームコンポーネントが使用する Angular Material の `mat-select` と不整合がある。
また、`$any($event.target).selectedIndex` を使ったインデックスベースの値参照はタイプセーフでなく、表示順序が変わった場合にバグを引き起こす。

```html
<!-- 現状: ネイティブ select + $any キャスト -->
<select class="month-select" (change)="onMonthChange(monthOptions()[$any($event.target).selectedIndex])">
```

**改善案:**

`mat-select` + `[(ngModel)]` または `formControl` を使用することで、タイプセーフかつMaterialデザインと整合した実装になる。

```html
<mat-select [value]="selectedMonth()" (selectionChange)="onMonthChange($event.value)">
  @for (option of monthOptions(); track option.label) {
    <mat-option [value]="option">{{ option.label }}</mat-option>
  }
</mat-select>
```

コンポーネント側の `imports` に `MatSelectModule` がすでに含まれているため、テンプレートの変更のみで対応できる。

---

#### M-3: ChildDetailComponent の computed 未使用（残高の表示はAPIから取得済みの値を使用）

**ファイル:** `src/app/features/children/child-detail/child-detail.component.ts` L1

**問題の説明:**

`computed` を import しているが実際には使用していない（未使用インポート）。
現在の月のレコードから残高を計算する `computed` シグナルを使う設計も検討できるが、バックエンドが子ども全期間の残高をAPIで返す設計のため、現状のデータフローは正しい。
ただし未使用インポートはビルド時の警告、コードの可読性低下につながる。

```typescript
// 使用されていない computed インポート
import { Component, OnInit, inject, signal, computed } from '@angular/core';
```

**改善案:**

```typescript
import { Component, OnInit, inject, signal } from '@angular/core';
```

---

#### M-4: ChildNewComponent と ChildEditComponent のフォーム定義が重複

**ファイル:**
- `src/app/features/children/child-new/child-new.component.ts` L37-41
- `src/app/features/children/child-edit/child-edit.component.ts` L42-46

**問題の説明:**

子ども情報フォームのバリデーション定義（name、age、base_allowance のルール）が2つのコンポーネントにまったく同一内容で存在する。
バリデーションルールを変更する場合に2箇所を修正する必要があり、片方の修正を忘れるリスクがある。

```typescript
// child-new と child-edit で同じコード
this.form = this.fb.group({
  name: ['', [Validators.required, Validators.maxLength(20)]],
  age: [null, [Validators.required, Validators.min(1), Validators.max(18)]],
  base_allowance: [null, [Validators.required, Validators.min(0)]],
});
```

**改善案:**

フォーム定義をファクトリ関数として共通化する。

```typescript
// shared/forms/child-form.factory.ts
export function createChildForm(fb: FormBuilder): FormGroup {
  return fb.group({
    name: ['', [Validators.required, Validators.maxLength(20)]],
    age: [null, [Validators.required, Validators.min(1), Validators.max(18)]],
    base_allowance: [null, [Validators.required, Validators.min(0)]],
  });
}
```

---

#### M-5: LoginComponent がコンストラクタインジェクションを使用している（他コンポーネントと非一貫）

**ファイル:** `src/app/features/chore-register/login/login.component.ts` L22-25

**問題の説明:**

プロジェクト内の他のすべてのコンポーネントは Angular 14以降推奨の `inject()` 関数を使用しているが、`LoginComponent` のみコンストラクタインジェクションを使用している。
コードスタイルの一貫性が損なわれる。

```typescript
// 現状: コンストラクタインジェクション
constructor(
  public auth: AuthService,
  private router: Router
) {}

// プロジェクト標準: inject() 関数
private auth = inject(AuthService);
private router = inject(Router);
```

**改善案:**

`inject()` 関数を使用するよう統一する。なお `auth` を `public` にしているのはテンプレートから参照するためだが、`inject()` に変更する場合は `auth` を `protected` または `public` な Signal/Observable プロパティとして公開することを検討する。

---

#### M-6: AllowanceTypeListComponent で金額に number パイプが未使用

**ファイル:** `src/app/features/allowance-types/allowance-type-list/allowance-type-list.component.html` L26

**問題の説明:**

他の画面では `| number` パイプで金額を `1,200` のようにフォーマットしているが、種類管理画面のみ `¥{{ allowanceType.amount }}` とそのまま表示している。表示の一貫性が欠ける。

```html
<!-- 現状: number パイプなし -->
<span class="item-amount">¥{{ allowanceType.amount }}</span>

<!-- 他の画面（例: dashboard.component.html）: number パイプあり -->
<div class="child-balance">¥{{ child.balance | number }}</div>
```

**改善案:**

```html
<span class="item-amount">¥{{ allowanceType.amount | number }}</span>
```

`imports` に `DecimalPipe` を追加する必要がある。

---

#### M-7: ChildDetailComponent の childId が空文字列のままAPIを呼び出すリスク

**ファイル:** `src/app/features/children/child-detail/child-detail.component.ts` L53-55

**問題の説明:**

`this.childId = this.route.snapshot.paramMap.get('id') ?? ''` としており、ルートパラメータが取得できなかった場合（URLが不正な場合）に空文字列でAPIを呼び出す。
`/children//records` のようなリクエストを送信することになり、バックエンドで意図しない動作を引き起こす可能性がある。
同様の問題が `ChildEditComponent` と `RecordNewComponent` にも存在する。

**改善案:**

```typescript
ngOnInit(): void {
  const id = this.route.snapshot.paramMap.get('id');
  if (!id) {
    this.router.navigate(['/dashboard']);
    return;
  }
  this.childId = id;
  this.loadData();
}
```

---

#### M-8: ConfirmDialogComponent が constructor インジェクションを使用

**ファイル:** `src/app/shared/confirm-dialog/confirm-dialog.component.ts` L26-30

**問題の説明:**

`MAT_DIALOG_DATA` は `@Inject` デコレーターが必要なため `inject()` 関数が直接使えないが（Angular 14以降は `inject(MAT_DIALOG_DATA)` で可能）、コンストラクタで `public` として注入しているため `data` がテンプレートから直接アクセス可能になっている点は問題ない。ただし `dialogRef` も `public` であるため外部から直接 `dialogRef.close()` を呼べる状態になっており、カプセル化が弱い。

**改善案:**

Angular 14以降では `inject` 関数で MAT_DIALOG_DATA を注入できる。

```typescript
export class ConfirmDialogComponent {
  protected data = inject<ConfirmDialogData>(MAT_DIALOG_DATA);
  private dialogRef = inject(MatDialogRef<ConfirmDialogComponent>);

  onCancel(): void {
    this.dialogRef.close(false);
  }

  onConfirm(): void {
    this.dialogRef.close(true);
  }
}
```

---

### 🟢 Low（任意対応）

---

#### L-1: `Record` 型名がグローバルな `Record` ユーティリティ型と衝突する

**ファイル:** `src/app/core/api.service.ts` L20

**問題の説明:**

`export interface Record` という名前はTypeScriptのグローバルユーティリティ型 `Record<K, V>` と同名である。
`api.service.spec.ts` でも `Record as ApiRecord` としてエイリアスが必要になっており（L9）、混乱を招く。

```typescript
// api.service.ts
export interface Record { ... } // TypeScript 組み込みの Record 型と同名

// spec.ts での回避策
import { Record as ApiRecord } from '../../../core/api.service';
```

**改善案:**

インターフェース名を `AllowanceRecord` や `FinancialRecord` など、ドメインを明示した名前に変更する。

---

#### L-2: 各コンポーネントで `MatSnackBar` の設定値が重複

**ファイル:** 全コンポーネント（8コンポーネント以上）

**問題の説明:**

`{ duration: 3000 }` というSnackBarのオプションが全コンポーネントにハードコードされている。
変更が必要になった場合に全ファイルを修正する必要がある。

**改善案:**

定数ファイルや共通サービスにSnackBarの設定を集約する。

```typescript
// src/app/core/snack-bar.service.ts
@Injectable({ providedIn: 'root' })
export class AppSnackBarService {
  private snackBar = inject(MatSnackBar);
  private readonly duration = 3000;

  showError(message: string): void {
    this.snackBar.open(message, '閉じる', { duration: this.duration });
  }

  showSuccess(message: string): void {
    this.snackBar.open(message, '閉じる', { duration: this.duration });
  }
}
```

---

#### L-3: child-detail.component.html のヘッダーボタンに aria-label が未設定

**ファイル:** `src/app/features/children/child-detail/child-detail.component.html` L3, L11

**問題の説明:**

「戻る」ボタンと「編集」ボタンに `aria-label` が設定されていない。他のコンポーネント（`dashboard.component.html` など）では適切に設定されており、一貫性が欠ける。

```html
<!-- 現状: aria-label なし -->
<button mat-icon-button (click)="goToDashboard()">
  <mat-icon>arrow_back</mat-icon>
</button>
<button mat-icon-button (click)="goToEdit()">
  <mat-icon>edit</mat-icon>
</button>
```

**改善案:**

```html
<button mat-icon-button (click)="goToDashboard()" aria-label="ダッシュボードへ戻る">
<button mat-icon-button (click)="goToEdit()" aria-label="子ども情報を編集">
```

---

#### L-4: child-new.component.html のローディングオーバーレイが二重

**ファイル:** `src/app/features/children/child-new/child-new.component.html` L50-55, L60-64

**問題の説明:**

保存ボタン内部にも `mat-spinner` があり（インラインスピナー）、かつ全画面の `loading-overlay` も表示される。
両方が同時に表示されるため、スピナーが二重に表示される。同様の問題が `child-edit.component.html` にも存在する。

```html
<!-- ボタン内スピナー -->
@if (loading()) {
  <mat-spinner diameter="20"></mat-spinner>
} @else {
  保存
}

<!-- 全画面オーバーレイスピナー（同時に表示される） -->
@if (loading()) {
  <div class="loading-overlay">
    <mat-spinner></mat-spinner>
  </div>
}
```

**改善案:**

どちらか一方に統一する。フォーム画面では保存ボタン内のインラインスピナーのみ残し、オーバーレイを削除する方が UX として自然。

---

#### L-5: テストファイルに `as any` キャストが複数存在する

**ファイル:**
- `child-detail.component.spec.ts` L128
- `child-edit.component.spec.ts` L136, L150, L159
- `allowance-type-edit.component.spec.ts` L137, L150, L159

**問題の説明:**

`MatDialogRef` のスタブを `as any` でキャストしているため、スタブの型チェックが効かない。
Jasmineの `SpyObj` や適切な型定義を使えば型安全にできる。

```typescript
// 現状
spyOn(dialog, 'open').and.returnValue({
  afterClosed: () => of(true),
} as any);
```

**改善案:**

```typescript
import { MatDialogRef } from '@angular/material/dialog';
spyOn(dialog, 'open').and.returnValue({
  afterClosed: () => of(true),
} as unknown as MatDialogRef<unknown>);
// または、テスト用の MatDialogRef スタブを定義する
```

---

#### L-6: ApiService のエラーハンドリングが各コンポーネントに分散

**ファイル:** 全コンポーネント

**問題の説明:**

APIエラー時の処理（SnackBar表示、loading解除）が各コンポーネントに繰り返し実装されている。
現状は問題ないが、スケールすると保守が困難になる。

**改善案（将来的な検討）:**

`ApiService` レイヤーで `catchError` オペレーターを使ったグローバルエラーハンドリング、またはHTTPインターセプターでの一元処理を検討する。ただし、エラーメッセージはコンテキスト依存（「子ども一覧の取得に失敗」等）であるため、コンポーネント側での個別ハンドリング自体は適切でもある。

---

## テスト品質の確認

### 良い点

- 全コンポーネントにスペックファイルが存在し、`expect(true).toBe(true)` のような無意味なアサーションは一切ない
- 境界値テスト（age: 0, 19, base_allowance: -1, 0; amount: 0, 1）が適切にカバーされている
- `fakeAsync`/`tick` を使った非同期テストが正しく実装されている
- APIエラー時のSnackBar表示と `loading` フラグのリセットが検証されている
- `Subject` を使った API通信中の `loading` 状態テストが実装されている（child-new, allowance-type-new, record-new）

### テストカバレッジの抜け

- **`LoginComponent` のテストファイルが存在しない:** ログイン画面のテストが未作成。`isAuthenticated$` が `true` の場合のリダイレクト、`login()` メソッドの呼び出しのテストが必要。
- **`ConfirmDialogComponent` のテストファイルが存在しない:** 確認/キャンセル時の `dialogRef.close` 呼び出しのテストが必要。
- **`auth.interceptor.ts` のテストファイルが存在しない:** `getAccessTokenSilently` の成功・失敗ケース、Authorizationヘッダーの付与確認が必要。インターセプターのテストは `HttpClientTestingModule` を使って実装できる。
- **`ChildDetailComponent` の月フィルタエラーケーステストが不足:** `onMonthChange` 後のAPIエラー時の SnackBar 表示テストがない。
- **`DashboardComponent` のテンプレートテスト:** `goToChildNew` ボタンクリックのテンプレートイベントテストがない（メソッドの直接呼び出しテストはある）。

---

## 良い点

### 設計・アーキテクチャ

- **Standalone Components の一貫した使用:** すべてのコンポーネントが `standalone: true` で実装されており、NgModule不使用のAngular 19らしいコードになっている。
- **Signals の適切な活用:** `loading`、`children`、`records` 等の状態管理にAngular Signalsを使用しており、`computed` や `effect` も必要な場面で適切に採用されている（`RecordNewComponent` の `isIncome` getter は適切）。
- **Lazy Loading の徹底:** すべてのルートで `loadComponent` による遅延ロードが実装されており、初期バンドルサイズが最小化されている。
- **`forkJoin` による並行API呼び出し:** `ChildDetailComponent` で child 情報と records を `forkJoin` で並行取得しており効率的。

### セキュリティ

- **XSSリスクなし:** テンプレートはすべて `{{ }}` バインディング（テキストノード）または `[属性]` バインディングを使用しており、`innerHTML` や `[outerHTML]` のような危険なバインディングは存在しない。`bypassSecurityTrust*` も使用していない。
- **環境変数の適切な管理:** Auth0のドメイン・クライアントIDが `environment.ts` で管理されており、コードへのハードコードはない。
- **AuthGuard の適切な実装:** `CanActivateFn` の functional guard で実装されており、未認証時の `loginWithRedirect()` も正しく機能する。

### コード品質

- **`RecordNewComponent` の `takeUntil` / `destroy$` パターン:** `valueChanges` のサブスクリプション管理が適切に実装されており、メモリリークを防いでいる。他のコンポーネントでも同様のパターンが必要な場合の良いリファレンスになっている。
- **型安全性:** `any` 型の使用が最小限（テストの `MatDialogRef` スタブのみ）であり、全体的に型安全なコードになっている。
- **ConfirmDialog の再利用:** 削除確認ダイアログが `ConfirmDialogComponent` として共通化されており、適切に再利用されている。
- **日付フォーマット処理の明示:** `RecordNewComponent` の `formatDate` メソッドがタイムゾーン問題を避けるためにローカル日付（`getFullYear`, `getMonth`, `getDate`）を使用しており、`toISOString()` によるUTCオフセット問題を回避している。
- **ルーティングの優先順位コメント:** `children/new` と `children/:id`、`allowance-types/new` と `allowance-types/:id/edit` の順序にコメントが付いており、将来の誤修正を防いでいる。
