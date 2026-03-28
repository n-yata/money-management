import { Routes } from '@angular/router';
import { authGuard } from './auth/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./features/chore-register/login/login.component').then(
        m => m.LoginComponent
      ),
  },
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/chore-register/chore-register.component').then(
        m => m.ChoreRegisterComponent
      ),
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/dashboard/dashboard.component').then(
        m => m.DashboardComponent
      ),
  },
  // children/new は children/:id より先に登録する（Angularのルーティング優先順位）
  {
    path: 'children/new',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/children/child-new/child-new.component').then(
        m => m.ChildNewComponent
      ),
  },
  {
    path: 'children/:id',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/children/child-detail/child-detail.component').then(
        m => m.ChildDetailComponent
      ),
  },
  {
    path: 'children/:id/edit',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/children/child-edit/child-edit.component').then(
        m => m.ChildEditComponent
      ),
  },
  // children/:id/records/new は children/:id より先に登録する（Angularのルーティング優先順位）
  {
    path: 'children/:id/records/new',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/children/record-new/record-new.component').then(
        m => m.RecordNewComponent
      ),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
