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
    path: 'allowance-types',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/allowance-types/allowance-type-list/allowance-type-list.component').then(
        m => m.AllowanceTypeListComponent
      ),
  },
  // allowance-types/new は allowance-types/:id/edit より先に登録する（Angularのルーティング優先順位）
  {
    path: 'allowance-types/new',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/allowance-types/allowance-type-new/allowance-type-new.component').then(
        m => m.AllowanceTypeNewComponent
      ),
  },
  {
    path: 'allowance-types/:id/edit',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/allowance-types/allowance-type-edit/allowance-type-edit.component').then(
        m => m.AllowanceTypeEditComponent
      ),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
