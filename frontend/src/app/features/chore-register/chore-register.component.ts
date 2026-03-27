import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

/**
 * お手伝い登録画面コンポーネント（スタブ）。
 * 子どもが操作するメイン画面。本実装は別Issueで行う。
 */
@Component({
  selector: 'app-chore-register',
  standalone: true,
  imports: [RouterLink, MatButtonModule, MatIconModule],
  templateUrl: './chore-register.component.html',
  styleUrl: './chore-register.component.scss',
})
export class ChoreRegisterComponent {}
