import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { DecimalPipe, NgClass } from '@angular/common';
import { forkJoin } from 'rxjs';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSelectModule } from '@angular/material/select';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog } from '@angular/material/dialog';
import { ApiService, Child, Record } from '../../../core/api.service';
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
    DecimalPipe,
    NgClass,
    MatButtonModule,
    MatIconModule,
    MatSelectModule,
    MatListModule,
    MatProgressSpinnerModule,
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
  records = signal<Record[]>([]);

  // 直近12ヶ月の選択肢
  monthOptions = signal<MonthOption[]>(this.buildMonthOptions());
  selectedMonth = signal<MonthOption>(this.monthOptions()[0]);

  private childId = '';

  ngOnInit(): void {
    this.childId = this.route.snapshot.paramMap.get('id') ?? '';
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
        this.records.set([...records].sort((a, b) => b.date.localeCompare(a.date)));
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
        this.records.set([...records].sort((a, b) => b.date.localeCompare(a.date)));
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('収支記録の取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  deleteRecord(record: Record): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '収支記録の削除',
        message: 'この収支記録を削除しますか？',
      },
    });

    dialogRef.afterClosed().subscribe((confirmed: boolean) => {
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
