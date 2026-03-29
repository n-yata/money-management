import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { take } from 'rxjs/operators';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog } from '@angular/material/dialog';
import { ApiService } from '../../../core/api.service';
import { ConfirmDialogComponent } from '../../../shared/confirm-dialog/confirm-dialog.component';

@Component({
  selector: 'app-allowance-type-edit',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './allowance-type-edit.component.html',
  styleUrl: './allowance-type-edit.component.scss',
})
export class AllowanceTypeEditComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);

  form!: FormGroup;
  loading = signal(false);
  private allowanceTypeId = '';

  ngOnInit(): void {
    // ルートパラメータが取得できない場合は種類一覧へリダイレクト（M-7対応）
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.router.navigate(['/allowance-types']);
      return;
    }
    this.allowanceTypeId = id;
    this.form = this.fb.group({
      name: ['', [Validators.required, Validators.maxLength(30)]],
      amount: [null, [Validators.required, Validators.min(1)]],
    });

    this.loadAllowanceType();
  }

  private loadAllowanceType(): void {
    this.loading.set(true);
    this.api.getAllowanceType(this.allowanceTypeId).subscribe({
      next: (allowanceType) => {
        this.form.patchValue({
          name: allowanceType.name,
          amount: allowanceType.amount,
        });
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('種類の情報取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  save(): void {
    if (this.form.invalid) return;

    this.loading.set(true);
    const { name, amount } = this.form.value;
    this.api.updateAllowanceType(this.allowanceTypeId, { name, amount: Number(amount) }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/allowance-types']);
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('種類の更新に失敗しました', '閉じる', { duration: 3000 });
      },
    });
  }

  cancel(): void {
    this.router.navigate(['/allowance-types']);
  }

  confirmDelete(): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '種類を削除',
        message: 'この種類を削除しますか？この操作は取り消せません。',
      },
    });

    // take(1) でダイアログが閉じた後の最初の値のみ取得し、サブスクリプションを自動完了させる
    dialogRef.afterClosed().pipe(take(1)).subscribe((confirmed: boolean) => {
      if (!confirmed) return;
      this.loading.set(true);
      this.api.deleteAllowanceType(this.allowanceTypeId).subscribe({
        next: () => {
          this.loading.set(false);
          this.router.navigate(['/allowance-types']);
        },
        error: () => {
          this.loading.set(false);
          this.snackBar.open('種類の削除に失敗しました', '閉じる', { duration: 3000 });
        },
      });
    });
  }
}
