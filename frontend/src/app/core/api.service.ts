import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
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

export interface Record {
  id: string;
  type: 'income' | 'expense';
  amount: number;
  description: string;
  date: string;
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
    return this.http.get<Child[]>(`${this.base}/children`);
  }

  // 子ども詳細取得
  getChild(id: string): Observable<Child> {
    return this.http.get<Child>(`${this.base}/children/${id}`);
  }

  // 子ども追加
  createChild(body: CreateChildRequest): Observable<Child> {
    return this.http.post<Child>(`${this.base}/children`, body);
  }

  // 子ども情報更新
  updateChild(id: string, body: UpdateChildRequest): Observable<Child> {
    return this.http.put<Child>(`${this.base}/children/${id}`, body);
  }

  // 子ども削除（関連recordsも削除）
  deleteChild(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/children/${id}`);
  }

  // おこづかいの種類一覧取得
  getAllowanceTypes(): Observable<AllowanceType[]> {
    return this.http.get<AllowanceType[]>(`${this.base}/allowance-types`);
  }

  // おこづかいの種類詳細取得
  getAllowanceType(id: string): Observable<AllowanceType> {
    return this.http.get<AllowanceType>(`${this.base}/allowance-types/${id}`);
  }

  // おこづかいの種類追加
  createAllowanceType(body: CreateAllowanceTypeRequest): Observable<AllowanceType> {
    return this.http.post<AllowanceType>(`${this.base}/allowance-types`, body);
  }

  // おこづかいの種類更新
  updateAllowanceType(id: string, body: UpdateAllowanceTypeRequest): Observable<AllowanceType> {
    return this.http.put<AllowanceType>(`${this.base}/allowance-types/${id}`, body);
  }

  // おこづかいの種類削除
  deleteAllowanceType(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/allowance-types/${id}`);
  }

  // 収支記録一覧取得（年月フィルタ付き）
  getRecords(childId: string, year: number, month: number): Observable<Record[]> {
    const params = new HttpParams()
      .set('year', year.toString())
      .set('month', month.toString());
    return this.http.get<Record[]>(`${this.base}/children/${childId}/records`, { params });
  }

  // 収支記録追加
  createRecord(childId: string, body: CreateRecordRequest): Observable<Record> {
    return this.http.post<Record>(`${this.base}/children/${childId}/records`, body);
  }

  // 収支記録削除
  deleteRecord(childId: string, recordId: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/children/${childId}/records/${recordId}`);
  }
}
