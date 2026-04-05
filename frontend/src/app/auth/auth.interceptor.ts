import { inject } from '@angular/core';
import { HttpInterceptorFn } from '@angular/common/http';
import { AuthService } from '@auth0/auth0-angular';
import { switchMap, catchError } from 'rxjs/operators';
import { throwError } from 'rxjs';
import { environment } from '../../environments/environment';

/**
 * 自社バックエンドへのHTTPリクエストにAuth0のアクセストークンを付与するインターセプター。
 * 自社API（environment.apiBaseUrl）以外のリクエストにはトークンを付与しない。
 * トークン取得に失敗した場合は loginWithRedirect でログイン画面へ誘導する。
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);

  // 自社バックエンド以外のリクエスト（Auth0 JWKS等）にはトークンを付与しない
  if (!req.url.startsWith(environment.apiBaseUrl)) {
    return next(req);
  }

  return auth.getAccessTokenSilently().pipe(
    catchError(err => {
      // トークン取得失敗時のみログイン画面へリダイレクト（APIエラーは対象外）
      auth.loginWithRedirect();
      return throwError(() => err);
    }),
    switchMap(token => {
      const authReq = req.clone({
        setHeaders: { Authorization: `Bearer ${token}` },
      });
      return next(authReq);
    }),
  );
};
