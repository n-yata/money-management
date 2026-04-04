import { TestBed } from '@angular/core/testing';
import {
  HttpClient,
  provideHttpClient,
  withInterceptors,
} from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { of, throwError } from 'rxjs';
import { AuthService } from '@auth0/auth0-angular';
import { authInterceptor } from './auth.interceptor';
import { environment } from '../../environments/environment';

describe('authInterceptor', () => {
  let httpClient: HttpClient;
  let httpMock: HttpTestingController;
  let authServiceStub: {
    getAccessTokenSilently: jasmine.Spy;
    loginWithRedirect: jasmine.Spy;
  };

  const apiUrl = `${environment.apiBaseUrl}/children`;
  const externalUrl = 'https://external-auth-provider.com/.well-known/jwks.json';

  beforeEach(() => {
    authServiceStub = {
      getAccessTokenSilently: jasmine.createSpy('getAccessTokenSilently'),
      loginWithRedirect: jasmine.createSpy('loginWithRedirect'),
    };

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: authServiceStub },
      ],
    });

    httpClient = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  describe('自社APIへのリクエスト', () => {
    it('Authorization: Bearer <token> ヘッダーが付与されること', () => {
      const fakeToken = 'test-access-token-xyz';
      authServiceStub.getAccessTokenSilently.and.returnValue(of(fakeToken));

      httpClient.get(apiUrl).subscribe();

      const req = httpMock.expectOne(apiUrl);
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${fakeToken}`);
      req.flush([]);
    });

    it('トークンを取得して正しいエンドポイントへリクエストが到達すること', () => {
      const fakeToken = 'another-token';
      authServiceStub.getAccessTokenSilently.and.returnValue(of(fakeToken));

      httpClient.get(apiUrl).subscribe();

      const req = httpMock.expectOne(apiUrl);
      expect(req.request.url).toBe(apiUrl);
      req.flush({});
    });
  });

  describe('外部URLへのリクエスト', () => {
    it('Authorization ヘッダーが付与されないこと', () => {
      httpClient.get(externalUrl).subscribe();

      const req = httpMock.expectOne(externalUrl);
      expect(req.request.headers.has('Authorization')).toBeFalse();
      req.flush({});
    });

    it('外部URLへのリクエストでは getAccessTokenSilently が呼ばれないこと', () => {
      httpClient.get(externalUrl).subscribe();

      httpMock.expectOne(externalUrl).flush({});
      expect(authServiceStub.getAccessTokenSilently).not.toHaveBeenCalled();
    });
  });

  describe('トークン取得失敗時', () => {
    it('loginWithRedirect が呼ばれること', () => {
      authServiceStub.getAccessTokenSilently.and.returnValue(
        throwError(() => new Error('Login required'))
      );

      // エラーが伝播するためエラーハンドラを設定する
      httpClient.get(apiUrl).subscribe({
        error: () => { /* エラーは期待通り */ },
      });

      // トークン取得失敗時にはHTTPリクエストは送出されないため verify() で確認済み
      expect(authServiceStub.loginWithRedirect).toHaveBeenCalledTimes(1);
    });

    it('トークン取得失敗時にエラーが呼び出し元に伝播すること', (done) => {
      const tokenError = new Error('Login required');
      authServiceStub.getAccessTokenSilently.and.returnValue(
        throwError(() => tokenError)
      );

      httpClient.get(apiUrl).subscribe({
        error: (err: Error) => {
          expect(err.message).toBe('Login required');
          done();
        },
      });
    });
  });
});
