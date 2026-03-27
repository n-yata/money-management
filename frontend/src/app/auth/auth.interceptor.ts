import { inject } from '@angular/core';
import { HttpInterceptorFn } from '@angular/common/http';
import { AuthService } from '@auth0/auth0-angular';
import { switchMap } from 'rxjs';

/**
 * すべてのHTTPリクエストにAuth0のアクセストークンを付与するインターセプター。
 * トークン取得に失敗した場合はエラーをスローし、ログインが必要であることを示す。
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);

  return auth.getAccessTokenSilently().pipe(
    switchMap(token => {
      const authReq = req.clone({
        setHeaders: { Authorization: `Bearer ${token}` },
      });
      return next(authReq);
    })
  );
};
