import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { Observable, of } from 'rxjs';
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
  });
});
