import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { DecimalPipe } from '@angular/common';
import { forkJoin } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { ApiService, Child, AllowanceType } from '../../core/api.service';

type Step = 'select-child' | 'select-type' | 'done';

/**
 * お手伝い登録画面コンポーネント。
 * 子どもがタップ操作だけでお手伝いを登録できる2ステップUI。
 *
 * フロー:
 *   ステップ1（子ども選択）→ ステップ2（種類選択）→ API登録 → 完了表示
 */
@Component({
  selector: 'app-chore-register',
  standalone: true,
  imports: [
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    DecimalPipe,
  ],
  templateUrl: './chore-register.component.html',
  styleUrl: './chore-register.component.scss',
})
export class ChoreRegisterComponent implements OnInit {
  private api = inject(ApiService);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);

  /** 現在のステップ */
  step = signal<Step>('select-child');
  /** ローディング中フラグ */
  loading = signal(false);

  /** 子ども一覧 */
  children = signal<Child[]>([]);
  /** お手伝い種類一覧 */
  allowanceTypes = signal<AllowanceType[]>([]);
  /** 選択中の子ども */
  selectedChild = signal<Child | null>(null);
  /** 選択中のお手伝い種類 */
  selectedType = signal<AllowanceType | null>(null);

  ngOnInit(): void {
    // forkJoin で子ども一覧とお手伝い種類を並行取得し、loading を一元管理する
    this.loading.set(true);
    forkJoin({
      children: this.api.getChildren(),
      allowanceTypes: this.api.getAllowanceTypes(),
    }).subscribe({
      next: ({ children, allowanceTypes }) => {
        this.children.set(children);
        this.allowanceTypes.set(allowanceTypes);
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('データの取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  /** 子どもを選択してステップ2へ進む */
  selectChild(child: Child): void {
    this.selectedChild.set(child);
    this.step.set('select-type');
  }

  /** お手伝い種類を選択してAPIに登録する */
  selectType(type: AllowanceType): void {
    const child = this.selectedChild();
    if (!child) return;

    this.loading.set(true);
    // 今日の日付を YYYY-MM-DD 形式で取得
    const d = new Date();
    const today = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

    this.api.createRecord(child.id, {
      type: 'income',
      amount: type.amount,
      description: type.name,
      date: today,
      allowance_type_id: type.id,
    }).subscribe({
      next: () => {
        this.selectedType.set(type);
        this.step.set('done');
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('登録に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  /** ステップ2からステップ1へ戻る */
  goBack(): void {
    this.step.set('select-child');
    this.selectedChild.set(null);
  }

  /** 完了後にステップ1へリセットして続けて登録できるようにする */
  reset(): void {
    this.step.set('select-child');
    this.selectedChild.set(null);
    this.selectedType.set(null);
  }

  /** 管理画面（ダッシュボード）へ遷移する */
  goToDashboard(): void {
    this.router.navigate(['/dashboard']);
  }
}
