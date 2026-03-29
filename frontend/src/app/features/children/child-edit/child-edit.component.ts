import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
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
import { createChildForm } from '../../../shared/forms/child-form.factory';

@Component({
  selector: 'app-child-edit',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './child-edit.component.html',
  styleUrl: './child-edit.component.scss',
})
export class ChildEditComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);

  form!: FormGroup;
  loading = signal(false);
  private childId = '';

  ngOnInit(): void {
    // ルートパラメータが取得できない場合はダッシュボードへリダイレクト（M-7対応）
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.router.navigate(['/dashboard']);
      return;
    }
    this.childId = id;
    // 共通ファクトリ関数でフォームを生成（M-4対応）
    this.form = createChildForm(this.fb);

    this.loadChild();
  }

  private loadChild(): void {
    this.loading.set(true);
    this.api.getChild(this.childId).subscribe({
      next: (child) => {
        this.form.patchValue({
          name: child.name,
          age: child.age,
          base_allowance: child.base_allowance,
        });
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('子どもの情報取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  save(): void {
    if (this.form.invalid) return;

    this.loading.set(true);
    const { name, age, base_allowance } = this.form.value;
    this.api.updateChild(this.childId, { name, age: Number(age), base_allowance: Number(base_allowance) }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/children', this.childId]);
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('子どもの情報更新に失敗しました', '閉じる', { duration: 3000 });
      },
    });
  }

  cancel(): void {
    this.router.navigate(['/children', this.childId]);
  }

  confirmDelete(): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '子どもの削除',
        message: '本当に削除しますか？この操作は取り消せません。',
      },
    });

    // take(1) でダイアログが閉じた後の最初の値のみ取得し、サブスクリプションを自動完了させる
    dialogRef.afterClosed().pipe(take(1)).subscribe((confirmed: boolean) => {
      if (!confirmed) return;
      this.loading.set(true);
      this.api.deleteChild(this.childId).subscribe({
        next: () => {
          this.loading.set(false);
          this.router.navigate(['/dashboard']);
        },
        error: () => {
          this.loading.set(false);
          this.snackBar.open('子どもの削除に失敗しました', '閉じる', { duration: 3000 });
        },
      });
    });
  }
}
