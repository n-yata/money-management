---
name: Angular Material モックの注意点
description: Standalone ComponentのimportsにMatSnackBarModule/MatDialogModuleを追加するとTestBedのプロバイダーモックが上書きされる
type: feedback
---

## 問題

Standalone Component の `imports` 配列に `MatSnackBarModule` や `MatDialogModule` を追加すると、TestBed の `providers` で設定した `MatSnackBar` や `MatDialog` のスパイ/モックが上書きされ、テストで `snackBar.open` や `dialog.open` が呼ばれないように見える問題が発生する。

## 原因

`MatSnackBarModule` や `MatDialogModule` を Standalone Component の `imports` に追加すると、Angular のコンパイラがそのモジュールのプロバイダーをコンポーネントスコープで登録し、`TestBed.configureTestingModule` の `providers` で設定したモックを無視することがある。

## 解決策

- コンポーネントの `imports` から `MatSnackBarModule` と `MatDialogModule` を削除する
- `MatSnackBar` と `MatDialog` はどちらも `providedIn: 'root'` なので、`imports` なしでも `inject()` で利用可能
- `ConfirmDialogComponent` 自体は `MatDialogModule` を imports に持つが、呼び出し側コンポーネントは不要
- テストでは `MatDialog` の open をスパイする場合、`TestBed.inject(MatDialog)` で実際のインスタンスを取得し `spyOn(dialog, 'open')` を使う（`jasmine.createSpyObj` で MatDialog をモック全体で置き換えると `openDialogs` が undefined になりエラー）

## テストでのベストプラクティス

```typescript
// NG: MatDialog をjasmine.createSpyObjで完全に置き換える
dialog = jasmine.createSpyObj('MatDialog', ['open']); // openDialogsがundefinedになる

// OK: TestBed.inject で実際のインスタンスを取得してspyOnする
dialog = TestBed.inject(MatDialog);
spyOn(dialog, 'open').and.returnValue({ afterClosed: () => of(true) } as any);
```
