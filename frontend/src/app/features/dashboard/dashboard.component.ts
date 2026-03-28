import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { DecimalPipe } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { ApiService, Child } from '../../core/api.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    DecimalPipe,
  ],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent implements OnInit {
  private api = inject(ApiService);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);

  loading = signal(false);
  children = signal<Child[]>([]);

  ngOnInit(): void {
    this.loadChildren();
  }

  private loadChildren(): void {
    this.loading.set(true);
    this.api.getChildren().subscribe({
      next: (children) => {
        this.children.set(children);
        this.loading.set(false);
      },
      error: () => {
        this.snackBar.open('子ども一覧の取得に失敗しました', '閉じる', { duration: 3000 });
        this.loading.set(false);
      },
    });
  }

  goToChoreRegister(): void {
    this.router.navigate(['/']);
  }

  goToAllowanceTypes(): void {
    this.router.navigate(['/allowance-types']);
  }

  goToChildDetail(child: Child): void {
    this.router.navigate(['/children', child.id]);
  }

  goToChildNew(): void {
    this.router.navigate(['/children/new']);
  }
}
