import { Component, OnInit, OnDestroy, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import {
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  Validators,
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatSelectModule } from '@angular/material/select';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatNativeDateModule, provideNativeDateAdapter } from '@angular/material/core';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Subject, takeUntil } from 'rxjs';
import { ApiService, AllowanceType } from '../../../core/api.service';

@Component({
  selector: 'app-record-new',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatSelectModule,
    MatDatepickerModule,
    MatNativeDateModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  providers: [provideNativeDateAdapter()],
  templateUrl: './record-new.component.html',
  styleUrl: './record-new.component.scss',
})
export class RecordNewComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);
  private destroy$ = new Subject<void>();

  form!: FormGroup;
  loading = signal(false);
  allowanceTypes = signal<AllowanceType[]>([]);

  /** ルートパラメータから取得した子どもID */
  childId = '';

  ngOnInit(): void {
    this.childId = this.route.snapshot.paramMap.get('id') ?? '';

    this.form = this.fb.group({
      type: ['income', Validators.required],
      allowance_type_id: [null],
      amount: [null, [Validators.required, Validators.min(1)]],
      description: ['', [Validators.required, Validators.maxLength(50)]],
      date: [new Date(), Validators.required],
    });

    // おこづかいの種類一覧を取得
    this.api.getAllowanceTypes().subscribe({
      next: (types) => this.allowanceTypes.set(types),
      error: () =>
        this.snackBar.open(
          'おこづかいの種類の取得に失敗しました',
          '閉じる',
          { duration: 3000 }
        ),
    });

    // 収支種別の切り替えを監視：支出に切り替えたら種類選択をリセット・非表示
    this.form
      .get('type')!
      .valueChanges.pipe(takeUntil(this.destroy$))
      .subscribe((type: string) => {
        if (type === 'expense') {
          this.form.get('allowance_type_id')!.setValue(null);
        }
      });

    // おこづかいの種類選択を監視：選択時に金額・説明を自動入力
    this.form
      .get('allowance_type_id')!
      .valueChanges.pipe(takeUntil(this.destroy$))
      .subscribe((typeId: string | null) => {
        if (!typeId) return;
        const selected = this.allowanceTypes().find((t) => t.id === typeId);
        if (selected) {
          this.form.get('amount')!.setValue(selected.amount);
          this.form.get('description')!.setValue(selected.name);
        }
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  /** 収入が選択されているかどうか（種類セレクト表示の制御） */
  get isIncome(): boolean {
    return this.form.get('type')?.value === 'income';
  }

  save(): void {
    if (this.form.invalid) return;

    this.loading.set(true);
    const { type, allowance_type_id, amount, description, date } =
      this.form.value;

    // Date オブジェクトを YYYY-MM-DD 文字列に変換
    const dateStr = this.formatDate(date as Date);

    const body: {
      type: 'income' | 'expense';
      amount: number;
      description: string;
      date: string;
      allowance_type_id?: string;
    } = {
      type,
      amount: Number(amount),
      description,
      date: dateStr,
    };

    // 種類が選択されている場合のみ allowance_type_id を付与
    if (allowance_type_id) {
      body.allowance_type_id = allowance_type_id;
    }

    this.api.createRecord(this.childId, body).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/children', this.childId]);
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('収支記録の追加に失敗しました', '閉じる', {
          duration: 3000,
        });
      },
    });
  }

  cancel(): void {
    this.router.navigate(['/children', this.childId]);
  }

  /** Date オブジェクトを YYYY-MM-DD 形式の文字列に変換 */
  private formatDate(date: Date): string {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
  }
}
