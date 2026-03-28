import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ApiService } from '../../../core/api.service';

@Component({
  selector: 'app-allowance-type-new',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './allowance-type-new.component.html',
  styleUrl: './allowance-type-new.component.scss',
})
export class AllowanceTypeNewComponent implements OnInit {
  private api = inject(ApiService);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);

  form!: FormGroup;
  loading = signal(false);

  ngOnInit(): void {
    this.form = this.fb.group({
      name: ['', [Validators.required, Validators.maxLength(30)]],
      amount: [null, [Validators.required, Validators.min(1)]],
    });
  }

  save(): void {
    if (this.form.invalid) return;

    this.loading.set(true);
    const { name, amount } = this.form.value;
    this.api.createAllowanceType({ name, amount: Number(amount) }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/allowance-types']);
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('種類の追加に失敗しました', '閉じる', { duration: 3000 });
      },
    });
  }

  cancel(): void {
    this.router.navigate(['/allowance-types']);
  }
}
