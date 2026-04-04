import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { Subject } from 'rxjs';
import { AuthService } from '@auth0/auth0-angular';
import { LoginComponent } from './login.component';

describe('LoginComponent', () => {
  let component: LoginComponent;
  let fixture: ComponentFixture<LoginComponent>;
  let authServiceStub: {
    isAuthenticated$: Subject<boolean>;
    loginWithRedirect: jasmine.Spy;
  };
  let router: jasmine.SpyObj<Router>;

  beforeEach(async () => {
    // AuthService の isAuthenticated$ を Subject で制御し、SDK内部に依存しない
    authServiceStub = {
      isAuthenticated$: new Subject<boolean>(),
      loginWithRedirect: jasmine.createSpy('loginWithRedirect'),
    };
    router = jasmine.createSpyObj('Router', ['navigate']);

    await TestBed.configureTestingModule({
      imports: [LoginComponent, NoopAnimationsModule],
      providers: [
        { provide: AuthService, useValue: authServiceStub },
        { provide: Router, useValue: router },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
  });

  describe('認証済みの場合のリダイレクト', () => {
    it('isAuthenticated$ が true を流したとき / へリダイレクトすること', fakeAsync(() => {
      fixture.detectChanges(); // ngOnInit 実行（takeUntilDestroyed 登録）

      authServiceStub.isAuthenticated$.next(true);
      tick();

      expect(router.navigate).toHaveBeenCalledWith(['/']);
    }));

    it('isAuthenticated$ が false を流したときはリダイレクトしないこと', fakeAsync(() => {
      fixture.detectChanges();

      authServiceStub.isAuthenticated$.next(false);
      tick();

      expect(router.navigate).not.toHaveBeenCalled();
    }));
  });

  describe('ログインボタン', () => {
    it('login() 呼び出し時に loginWithRedirect が appState: { target: "/" } で呼ばれること', () => {
      fixture.detectChanges();

      component.login();

      expect(authServiceStub.loginWithRedirect).toHaveBeenCalledOnceWith({
        appState: { target: '/' },
      });
    });
  });

  describe('コンポーネント破棄時のサブスクリプション解除', () => {
    it('コンポーネント破棄後は isAuthenticated$ の新しい値でリダイレクトが発生しないこと', fakeAsync(() => {
      fixture.detectChanges();

      // 破棄前に認証されていない値を流す（リダイレクトなし）
      authServiceStub.isAuthenticated$.next(false);
      tick();
      expect(router.navigate).not.toHaveBeenCalled();

      // コンポーネントを破棄する（takeUntilDestroyed が発動）
      fixture.destroy();

      // 破棄後に true を流してもサブスクリプションは解除済みなのでリダイレクトされない
      authServiceStub.isAuthenticated$.next(true);
      tick();

      expect(router.navigate).not.toHaveBeenCalled();
    }));
  });
});
