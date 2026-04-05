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
  /** 今日すでに登録済みのお手伝い種類IDセット */
  todayDoneTypeIds = signal<Set<string>>(new Set());

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

  /** 子どもを選択してステップ2へ進む（今日の記録を取得して登録済み種類を判定） */
  selectChild(child: Child): void {
    this.selectedChild.set(child);
    this.loading.set(true);

    const d = new Date();
    this.api.getRecords(child.id, d.getFullYear(), d.getMonth() + 1).subscribe({
      next: (records) => {
        const todayStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
        const doneIds = new Set(
          records
            .filter(r => r.allowance_type_id != null && r.date.startsWith(todayStr))
            .map(r => r.allowance_type_id!)
        );
        this.todayDoneTypeIds.set(doneIds);
        this.loading.set(false);
        this.step.set('select-type');
      },
      error: () => {
        this.todayDoneTypeIds.set(new Set());
        this.loading.set(false);
        this.step.set('select-type');
      },
    });
  }

  /** お手伝い種類を選択してAPIに登録する */
  selectType(type: AllowanceType): void {
    const child = this.selectedChild();
    if (!child) return;

    // 今日すでに登録済みの種類はSnackBarで通知してブロック
    if (this.todayDoneTypeIds().has(type.id)) {
      this.snackBar.open('このお手伝いは今日すでに登録されています', '閉じる', { duration: 3000 });
      return;
    }

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
      error: (err) => {
        // バックエンドが 409 Conflict で返す重複メッセージも適切に表示する
        const msg = err?.error?.message === 'このお手伝いは今日すでに登録されています'
          ? 'このお手伝いは今日すでに登録されています'
          : '登録に失敗しました';
        this.snackBar.open(msg, '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  /** ステップ2からステップ1へ戻る */
  goBack(): void {
    this.step.set('select-child');
    this.selectedChild.set(null);
    this.todayDoneTypeIds.set(new Set());
  }

  /** 完了後にステップ1へリセットして続けて登録できるようにする */
  reset(): void {
    this.step.set('select-child');
    this.selectedChild.set(null);
    this.selectedType.set(null);
    this.todayDoneTypeIds.set(new Set());
  }

  /** 管理画面（ダッシュボード）へ遷移する */
  goToDashboard(): void {
    this.router.navigate(['/dashboard']);
  }

  // 子どもカードのパステル背景色（循環）
  private readonly cardBgColors = [
    '#e3f2fd', '#e8f5e9', '#f3e5f5', '#fff3e0', '#e0f7fa',
  ];
  private readonly cardBorderColors = [
    '#1565c0', '#2e7d32', '#6a1b9a', '#e65100', '#00695c',
  ];

  getCardBg(index: number): string {
    return this.cardBgColors[index % this.cardBgColors.length];
  }

  getCardBorder(index: number): string {
    return this.cardBorderColors[index % this.cardBorderColors.length];
  }
}
