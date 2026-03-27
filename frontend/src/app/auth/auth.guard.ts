import { inject } from '@angular/core';
import { CanActivateFn } from '@angular/router';
import { AuthService } from '@auth0/auth0-angular';
import { tap } from 'rxjs';

/**
 * 認証済みユーザーのみルートへのアクセスを許可するガード。
 * 未認証の場合はAuth0のユニバーサルログイン画面にリダイレクトする。
 */
export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);

  return auth.isAuthenticated$.pipe(
    tap(isAuthenticated => {
      if (!isAuthenticated) {
        auth.loginWithRedirect();
      }
    })
  );
};
