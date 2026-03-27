import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from '@auth0/auth0-angular';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

/**
 * ログイン画面コンポーネント。
 * 認証済みの場合はお手伝い登録画面（/）にリダイレクトする。
 * 未認証の場合はAuth0のユニバーサルログインへのボタンを表示する。
 */
@Component({
  selector: 'app-login',
  standalone: true,
  imports: [MatCardModule, MatButtonModule, MatIconModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent implements OnInit {
  constructor(
    public auth: AuthService,
    private router: Router
  ) {}

  ngOnInit(): void {
    // 認証済みの場合はお手伝い登録画面にリダイレクト
    this.auth.isAuthenticated$.subscribe(isAuthenticated => {
      if (isAuthenticated) {
        this.router.navigate(['/']);
      }
    });
  }

  /**
   * Auth0のユニバーサルログイン画面にリダイレクトする。
   * ログイン成功後はお手伝い登録画面（/）に戻る。
   */
  login(): void {
    this.auth.loginWithRedirect({ appState: { target: '/' } });
  }
}
