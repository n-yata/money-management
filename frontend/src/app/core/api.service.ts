import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
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

export interface CreateRecordRequest {
  type: 'income' | 'expense';
  amount: number;
  description: string;
  date: string;
  allowance_type_id?: string;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private http = inject(HttpClient);
  private base = environment.apiBaseUrl;

  getChildren(): Observable<Child[]> {
    return this.http.get<Child[]>(`${this.base}/children`);
  }

  getAllowanceTypes(): Observable<AllowanceType[]> {
    return this.http.get<AllowanceType[]>(`${this.base}/allowance-types`);
  }

  createRecord(childId: string, body: CreateRecordRequest): Observable<Record> {
    return this.http.post<Record>(`${this.base}/children/${childId}/records`, body);
  }
}
