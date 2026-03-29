import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ApiService } from '../../../core/api.service';
import { createChildForm } from '../../../shared/forms/child-form.factory';

@Component({
  selector: 'app-child-new',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './child-new.component.html',
  styleUrl: './child-new.component.scss',
})
export class ChildNewComponent implements OnInit {
  private api = inject(ApiService);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);

  form!: FormGroup;
  loading = signal(false);

  ngOnInit(): void {
    // 共通ファクトリ関数でフォームを生成（M-4対応）
    this.form = createChildForm(this.fb);
  }

  save(): void {
    if (this.form.invalid) return;

    this.loading.set(true);
    const { name, age, base_allowance } = this.form.value;
    this.api.createChild({ name, age: Number(age), base_allowance: Number(base_allowance) }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/dashboard']);
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('子どもの追加に失敗しました', '閉じる', { duration: 3000 });
      },
    });
  }

  cancel(): void {
    this.router.navigate(['/dashboard']);
  }
}
