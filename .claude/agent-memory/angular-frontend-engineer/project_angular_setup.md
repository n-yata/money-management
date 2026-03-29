---
name: Angular Frontend Setup
description: Angularプロジェクトの初期セットアップ情報（バージョン、使用ライブラリ、設定）
type: project
---

## Angularプロジェクト基本情報

- Angular CLI: 19.0.6
- Angular: 19.x
- Node.js: 22.10.0
- プロジェクトパス: `C:/develop/workspace-claude/money-management/frontend/`
- スタイル: SCSS
- Standalone Components を使用（モジュールなし）
- routing: 有効

## インストール済みライブラリ

- `@angular/pwa` 19.2.23 - PWA / Service Worker
- `@angular/material` 19.2.19 - UIコンポーネント（テーマ: indigo-pink）
- `@auth0/auth0-angular` - Auth0認証

## 環境設定

- `src/environments/environment.ts` - 開発環境（apiBaseUrl: http://localhost:3000/api/v1）
- `src/environments/environment.prod.ts` - 本番環境（apiBaseUrl: /api/v1）
- `angular.json` にfileReplacementsを設定済み（production時にenvironment.prod.tsを使用）

## Service Worker設定

- `ngsw-config.json`: APIキャッシュなし、静的アセットのみキャッシュ
- appグループ: prefetch（index.html, css, js等）
- assetsグループ: lazy（画像、フォント等）

## 認証設定（実装済み）

- `src/app/auth/auth.guard.ts`: `CanActivateFn` functional guard。未認証時は `auth.loginWithRedirect()` を呼び出す
- `src/app/auth/auth.interceptor.ts`: `HttpInterceptorFn` functional interceptor。全リクエストに `Authorization: Bearer <token>` を付与
- `app.config.ts` に `provideAuth0()` と `provideHttpClient(withInterceptors([authInterceptor]))` を設定済み

## ルーティング

- `/login`: LoginComponent（Standalone, lazy loaded）
- `/`: ChoreRegisterComponent（Standalone, lazy loaded, authGuard保護）
- `/dashboard`: DashboardComponent（authGuard保護）
- `/children/new`: ChildNewComponent（authGuard保護）
- `/children/:id`: ChildDetailComponent（authGuard保護）
- `/children/:id/edit`: ChildEditComponent（authGuard保護）
- `/children/:id/records/new`: RecordNewComponent（authGuard保護）
- `/allowance-types`: AllowanceTypeListComponent（authGuard保護）
- `/allowance-types/new`: AllowanceTypeNewComponent（authGuard保護、:id/editより先に登録）
- `/allowance-types/:id/edit`: AllowanceTypeEditComponent（authGuard保護）
- `**`: redirectTo ''

## コーディングパターン（実装済み）

- `FinancialRecord` インターフェース（JavaScript 組み込みの `Record<K,V>` との衝突を避けるため改名済み）
- `createChildForm` ファクトリ関数: `src/app/shared/forms/child-form.factory.ts`（ChildNew/ChildEdit 共通）
- ダイアログの `afterClosed()` には必ず `take(1)` を付ける
- `forkJoin` で並行取得してローディングを一元管理（ChoreRegisterComponent参照）
- `inject()` 関数スタイルに統一（コンストラクタインジェクションは使わない）
- `takeUntilDestroyed(destroyRef)` でサブスクリプション管理（LoginComponent参照）
- `authInterceptor` は `environment.apiBaseUrl` で始まるURLのみトークン付与
- `childId` 等のルートパラメータは `null` チェック後ダッシュボードへリダイレクト（M-7パターン）

## テストパターン

- ApiService は `jasmine.createSpyObj` でスタブ化（HttpClientTestingModule は使わない）
- 非同期処理は `fakeAsync` / `tick` を使用
- MatDatepicker を使うコンポーネントには `provideNativeDateAdapter()` が TestBed の providers にも必要
- コンポーネントの `providers: [provideNativeDateAdapter()]` も必要
- `forkJoin` で並行取得した場合、どちらがエラーになっても同一のエラーメッセージになる点に注意

## バンドルbudget

- `angular.json` の initial budget: maximumWarning 600kB, maximumError 1MB（Angular Material + Auth0 SDKのサイズに合わせて調整済み）
