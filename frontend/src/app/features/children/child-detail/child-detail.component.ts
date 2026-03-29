import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { DatePipe, DecimalPipe, NgClass } from '@angular/common';
import { forkJoin } from 'rxjs';
import { take } from 'rxjs/operators';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSelectModule } from '@angular/material/select';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog } from '@angular/material/dialog';
import { ApiService, Child, FinancialRecord } from '../../../core/api.service';
import { ConfirmDialogComponent } from '../../../shared/confirm-dialog/confirm-dialog.component';

export interface MonthOption {
  year: number;
  month: number;
  label: string;
}

@Component({
  selector: 'app-child-detail',
  standalone: true,
  imports: [
    DatePipe,
    DecimalPipe,
    NgClass,
    MatButtonModule,
    MatIconModule,
    MatSelectModule,
    MatListModule,
    MatProgressSpinnerModule,
    MatCheckboxModule,
  ],
  templateUrl: './child-detail.component.html',
  styleUrl: './child-detail.component.scss',
})
export class ChildDetailComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);

  loading = signal(false);
  child = signal<Child | null>(null);
  records = signal<FinancialRecord[]>([]);

  // 直近12ヶ月の選択肢
  monthOptions = signal<MonthOption[]>(this.buildMonthOptions());
  selectedMonth = signal<MonthOption>(this.monthOptions()[0]);

  // 支払いモード
  paymentMode = signal(false);
  selectedRecordIds = signal<Set<string>>(new Set());

  // 未払いと判定するincomeレコードIDのSet
  // child.balance（全期間残高）を基準に、新しい順に残高分だけ未払い扱いにする
  unpaidIncomeIds = computed(() => {
    const balance = this.child()?.balance ?? 0;
    const incomes = this.records()
      .filter(r => r.type === 'income')
      .sort((a, b) => b.created_at.localeCompare(a.created_at)); // 新しい順
    const unpaid = new Set<string>();
    let covered = 0;
    for (const r of incomes) {
      if (covered >= balance) break;
      unpaid.add(r.id);
      covered += r.amount;
    }
    return unpaid;
  });

  selectedTotal = computed(() =>
    this.records()
      .filter(r => r.type === 'income' && this.selectedRecordIds().has(r.id))
      .reduce((sum, r) => sum + r.amount, 0)
  );
  isOverBalance = computed(() => this.selectedTotal() > (this.child()?.balance ?? 0));

  private childId = '';

  ngOnInit(): void {
    // ルートパラメータが取得できない場合はダッシュボードへリダイレクト（M-7対応）
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.router.navigate(['/dashboard']);
      return;
    }
    this.childId = id;
    this.loadData();
  }

  private buildMonthOptions(): MonthOption[] {
    const now = new Date();
    const options: MonthOption[] = [];
    for (let i = 0; i < 12; i++) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
      options.push({
        year: d.getFullYear(),
        month: d.getMonth() + 1,
        label: `${d.getFullYear()}年${d.getMonth() + 1}月`,
      });
    }
    return options;
  }

  private loadData(): void {
    this.loading.set(true);
    const { year, month } = this.selectedMonth();
    forkJoin({
      child: this.api.getChild(this.childId),
      records: this.api.getRecords(this.childId, year, month),
    }).subscribe({
      next: ({ child, records }) => {
        this.child.set(child);
        // 日付新しい順にソート
        this.records.set([...records].sort((a, b) => b.created_at.localeCompare(a.created_at)));
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('データの取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  onMonthChange(option: MonthOption): void {
    this.selectedMonth.set(option);
    this.loadRecords();
  }

  private loadRecords(): void {
    this.loading.set(true);
    const { year, month } = this.selectedMonth();
    this.api.getRecords(this.childId, year, month).subscribe({
      next: (records) => {
        this.records.set([...records].sort((a, b) => b.created_at.localeCompare(a.created_at)));
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('収支記録の取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  deleteRecord(record: FinancialRecord): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '収支記録の削除',
        message: 'この収支記録を削除しますか？',
      },
    });

    // take(1) でダイアログが閉じた後の最初の値のみ取得し、サブスクリプションを自動完了させる
    dialogRef.afterClosed().pipe(take(1)).subscribe((confirmed: boolean) => {
      if (!confirmed) return;
      this.api.deleteRecord(this.childId, record.id).subscribe({
        next: () => {
          this.loadRecords();
        },
        error: () => {
          this.snackBar.open('収支記録の削除に失敗しました', '閉じる', { duration: 3000 });
        },
      });
    });
  }

  togglePaymentMode(): void {
    this.paymentMode.set(!this.paymentMode());
    this.selectedRecordIds.set(new Set());
  }

  toggleRecord(id: string): void {
    const next = new Set(this.selectedRecordIds());
    next.has(id) ? next.delete(id) : next.add(id);
    this.selectedRecordIds.set(next);
  }

  paySelected(): void {
    const total = this.selectedTotal();
    const count = this.selectedRecordIds().size;
    if (total === 0) return;

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'おこづかいを渡す',
        message: `¥${total.toLocaleString()} を渡しますか？（${count}件分）`,
      },
    });

    dialogRef.afterClosed().pipe(take(1)).subscribe((confirmed: boolean) => {
      if (!confirmed) return;
      const d = new Date();
      const today = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
      this.api.createRecord(this.childId, {
        type: 'expense',
        amount: total,
        description: `おこづかい支払い（${count}件分）`,
        date: today,
      }).subscribe({
        next: () => {
          this.paymentMode.set(false);
          this.selectedRecordIds.set(new Set());
          this.loadData();
          this.snackBar.open('おこづかいを渡しました', '閉じる', { duration: 2000 });
        },
        error: () => {
          this.snackBar.open('支払いの記録に失敗しました', '閉じる', { duration: 3000 });
        },
      });
    });
  }

  goToDashboard(): void {
    this.router.navigate(['/dashboard']);
  }

  goToEdit(): void {
    this.router.navigate(['/children', this.childId, 'edit']);
  }

  goToRecordNew(): void {
    this.router.navigate(['/children', this.childId, 'records', 'new']);
  }
}
