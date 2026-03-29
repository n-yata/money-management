import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface Child {
  id: string;
  name: string;
  age: number;
  base_allowance: number;
  balance: number;
}

export interface AllowanceType {
  id: string;
  name: string;
  amount: number;
}

// JavaScript 組み込みの Record 型との名前衝突を避けるため FinancialRecord に変更（L-1対応）
export interface FinancialRecord {
  id: string;
  type: 'income' | 'expense';
  amount: number;
  description: string;
  date: string;
  created_at: string;
  allowance_type_id?: string;
}

export interface CreateChildRequest {
  name: string;
  age: number;
  base_allowance: number;
}

export interface UpdateChildRequest {
  name: string;
  age: number;
  base_allowance: number;
}

export interface CreateRecordRequest {
  type: 'income' | 'expense';
  amount: number;
  description: string;
  date: string;
  allowance_type_id?: string;
}

export interface CreateAllowanceTypeRequest {
  name: string;
  amount: number;
}

export interface UpdateAllowanceTypeRequest {
  name: string;
  amount: number;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private http = inject(HttpClient);
  private base = environment.apiBaseUrl;

  // 子ども一覧取得（残高を含む）
  getChildren(): Observable<Child[]> {
    return this.http.get<{ data: Child[] }>(`${this.base}/children`).pipe(map(res => res.data));
  }

  // 子ども詳細取得
  getChild(id: string): Observable<Child> {
    return this.http.get<{ data: Child }>(`${this.base}/children/${id}`).pipe(map(res => res.data));
  }

  // 子ども追加
  createChild(body: CreateChildRequest): Observable<Child> {
    return this.http.post<{ data: Child }>(`${this.base}/children`, body).pipe(map(res => res.data));
  }

  // 子ども情報更新
  updateChild(id: string, body: UpdateChildRequest): Observable<Child> {
    return this.http.put<{ data: Child }>(`${this.base}/children/${id}`, body).pipe(map(res => res.data));
  }

  // 子ども削除（関連recordsも削除）
  deleteChild(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/children/${id}`);
  }

  // おこづかいの種類一覧取得
  getAllowanceTypes(): Observable<AllowanceType[]> {
    return this.http.get<{ data: AllowanceType[] }>(`${this.base}/allowance-types`).pipe(map(res => res.data));
  }

  // おこづかいの種類詳細取得
  getAllowanceType(id: string): Observable<AllowanceType> {
    return this.http.get<{ data: AllowanceType }>(`${this.base}/allowance-types/${id}`).pipe(map(res => res.data));
  }

  // おこづかいの種類追加
  createAllowanceType(body: CreateAllowanceTypeRequest): Observable<AllowanceType> {
    return this.http.post<{ data: AllowanceType }>(`${this.base}/allowance-types`, body).pipe(map(res => res.data));
  }

  // おこづかいの種類更新
  updateAllowanceType(id: string, body: UpdateAllowanceTypeRequest): Observable<AllowanceType> {
    return this.http.put<{ data: AllowanceType }>(`${this.base}/allowance-types/${id}`, body).pipe(map(res => res.data));
  }

  // おこづかいの種類削除
  deleteAllowanceType(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/allowance-types/${id}`);
  }

  // 収支記録一覧取得（年月フィルタ付き）
  getRecords(childId: string, year: number, month: number): Observable<FinancialRecord[]> {
    const params = new HttpParams()
      .set('year', year.toString())
      .set('month', month.toString());
    return this.http.get<{ data: FinancialRecord[] }>(`${this.base}/children/${childId}/records`, { params }).pipe(map(res => res.data));
  }

  // 収支記録追加
  createRecord(childId: string, body: CreateRecordRequest): Observable<FinancialRecord> {
    return this.http.post<{ data: FinancialRecord }>(`${this.base}/children/${childId}/records`, body).pipe(map(res => res.data));
  }

  // 収支記録削除
  deleteRecord(childId: string, recordId: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/children/${childId}/records/${recordId}`);
  }
}
