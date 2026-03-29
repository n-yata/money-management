import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { DecimalPipe } from '@angular/common';
import { MatListModule } from '@angular/material/list';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ApiService, AllowanceType } from '../../../core/api.service';

@Component({
  selector: 'app-allowance-type-list',
  standalone: true,
  imports: [
    DecimalPipe,
    MatListModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './allowance-type-list.component.html',
  styleUrl: './allowance-type-list.component.scss',
})
export class AllowanceTypeListComponent implements OnInit {
  private api = inject(ApiService);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);

  loading = signal(false);
  allowanceTypes = signal<AllowanceType[]>([]);

  ngOnInit(): void {
    this.loadAllowanceTypes();
  }

  private loadAllowanceTypes(): void {
    this.loading.set(true);
    this.api.getAllowanceTypes().subscribe({
      next: (types) => {
        this.allowanceTypes.set(types);
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('種類一覧の取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  goToDashboard(): void {
    this.router.navigate(['/dashboard']);
  }

  goToNew(): void {
    this.router.navigate(['/allowance-types/new']);
  }

  goToEdit(allowanceType: AllowanceType): void {
    this.router.navigate(['/allowance-types', allowanceType.id, 'edit']);
  }
}
