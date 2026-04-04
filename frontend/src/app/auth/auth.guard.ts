import { inject } from '@angular/core';
import { CanActivateFn } from '@angular/router';
import { AuthService } from '@auth0/auth0-angular';
import { filter, switchMap, take, tap } from 'rxjs';

/**
 * 認証済みユーザーのみルートへのアクセスを許可するガード。
 * Auth0のローディング完了を待ってから認証状態を評価することで、
 * 初期化中に誤ってリダイレクトされるのを防ぐ。
 * 未認証の場合はAuth0のユニバーサルログイン画面にリダイレクトする。
 * ログイン後に元のURLへ戻れるよう appState.target にリダイレクト元URLを渡す。
 */
export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);

  return auth.isLoading$.pipe(
    filter(loading => !loading),
    take(1),
    switchMap(() => auth.isAuthenticated$),
    tap(isAuthenticated => {
      if (!isAuthenticated) {
        auth.loginWithRedirect({ appState: { target: state.url } });
      }
    }),
  );
};
