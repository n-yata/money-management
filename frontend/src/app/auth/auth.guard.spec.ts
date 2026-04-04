import { TestBed, fakeAsync, tick } from '@angular/core/testing';
import { Router } from '@angular/router';
import { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { Observable, ReplaySubject, of } from 'rxjs';
import { AuthService } from '@auth0/auth0-angular';
import { authGuard } from './auth.guard';

describe('authGuard', () => {
  let authServiceSpy: jasmine.SpyObj<AuthService>;

  const dummyRoute = {} as ActivatedRouteSnapshot;
  const dummyState = { url: '/dashboard' } as RouterStateSnapshot;

  /**
   * authGuard は CanActivateFn のため TestBed.runInInjectionContext で inject() を解決する。
   * 戻り値は常に Observable<boolean> なので安全にキャストする。
   */
  const runGuard = () =>
    TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState)) as Observable<boolean>;

  beforeEach(() => {
    authServiceSpy = jasmine.createSpyObj<AuthService>('AuthService', ['loginWithRedirect'], {
      isAuthenticated$: of(false), // デフォルトは未認証
      isLoading$: of(false),       // デフォルトはロード完了
    });

    TestBed.configureTestingModule({
      providers: [
        { provide: AuthService, useValue: authServiceSpy },
        { provide: Router, useValue: jasmine.createSpyObj('Router', ['navigate']) },
      ],
    });
  });

  // ── 認証済み ──────────────────────────────────────────────────

  describe('認証済みユーザー', () => {
    beforeEach(() => {
      Object.defineProperty(authServiceSpy, 'isAuthenticated$', { value: of(true) });
    });

    it('true を返す', (done) => {
      runGuard().subscribe((result: boolean) => {
        expect(result).toBeTrue();
        done();
      });
    });

    it('loginWithRedirect を呼ばない', (done) => {
      runGuard().subscribe(() => {
        expect(authServiceSpy.loginWithRedirect).not.toHaveBeenCalled();
        done();
      });
    });
  });

  // ── 未認証 ────────────────────────────────────────────────────

  describe('未認証ユーザー', () => {
    it('false を返す', (done) => {
      runGuard().subscribe((result: boolean) => {
        expect(result).toBeFalse();
        done();
      });
    });

    it('loginWithRedirect を呼んでAuth0ログイン画面へリダイレクトする', (done) => {
      runGuard().subscribe(() => {
        expect(authServiceSpy.loginWithRedirect).toHaveBeenCalledTimes(1);
        done();
      });
    });

    it('loginWithRedirect に appState.target としてリダイレクト元URL が渡されること', (done) => {
      runGuard().subscribe(() => {
        expect(authServiceSpy.loginWithRedirect).toHaveBeenCalledWith(
          jasmine.objectContaining({
            appState: jasmine.objectContaining({ target: dummyState.url }),
          })
        );
        done();
      });
    });
  });

  // ── isLoading$ の遷移 ──────────────────────────────────────────

  describe('isAuthenticated$ が遅延して値を発行する場合（isLoading$ 遷移シミュレーション）', () => {
    it('認証済みのとき true を返すこと', fakeAsync(() => {
      const auth$ = new ReplaySubject<boolean>(1);

      const authSpy = jasmine.createSpyObj<AuthService>('AuthService', ['loginWithRedirect'], {
        isAuthenticated$: auth$.asObservable(),
        isLoading$: of(false),
      });

      TestBed.overrideProvider(AuthService, { useValue: authSpy });

      let result: boolean | undefined;
      (TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState)) as Observable<boolean>)
        .subscribe((v: boolean) => (result = v));

      // ロード完了後に認証済みの値を発行
      auth$.next(true);
      tick();

      expect(result).toBeTrue();
      expect(authSpy.loginWithRedirect).not.toHaveBeenCalled();
    }));

    it('未認証のとき false を返し loginWithRedirect が呼ばれること', fakeAsync(() => {
      const auth$ = new ReplaySubject<boolean>(1);

      const authSpy = jasmine.createSpyObj<AuthService>('AuthService', ['loginWithRedirect'], {
        isAuthenticated$: auth$.asObservable(),
        isLoading$: of(false),
      });

      TestBed.overrideProvider(AuthService, { useValue: authSpy });

      let result: boolean | undefined;
      (TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState)) as Observable<boolean>)
        .subscribe((v: boolean) => (result = v));

      // ロード完了後に未認証の値を発行
      auth$.next(false);
      tick();

      expect(result).toBeFalse();
      expect(authSpy.loginWithRedirect).toHaveBeenCalledTimes(1);
    }));
  });
});
