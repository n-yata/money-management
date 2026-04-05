import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import {
  ApiService,
  Child,
  AllowanceType,
  FinancialRecord,
  CreateChildRequest,
  UpdateChildRequest,
  CreateRecordRequest,
  CreateAllowanceTypeRequest,
  UpdateAllowanceTypeRequest,
} from './api.service';
import { environment } from '../../environments/environment';

const base = environment.apiBaseUrl;

const mockChild: Child = { id: 'c1', name: 'たろう', age: 8, base_allowance: 1000, balance: 1200 };
const mockChildren: Child[] = [
  mockChild,
  { id: 'c2', name: 'はなこ', age: 6, base_allowance: 800, balance: 500 },
];
const mockType: AllowanceType = { id: 't1', name: 'お皿洗い', amount: 50 };
const mockTypes: AllowanceType[] = [mockType, { id: 't2', name: '掃除機かけ', amount: 80 }];
const mockRecord: FinancialRecord = { id: 'r1', type: 'income', amount: 50, description: 'お皿洗い', date: '2026-03-27', created_at: '2026-03-27T10:00:00Z' };

describe('ApiService', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
    });
    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  // ── 子ども一覧 ───────────────────────────────────────────────

  describe('getChildren', () => {
    it('GET /children を呼び出して子ども一覧を返す', () => {
      let result: Child[] | undefined;
      service.getChildren().subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/children`);
      expect(req.request.method).toBe('GET');
      req.flush({ data: mockChildren });

      expect(result).toEqual(mockChildren);
    });

    it('子どもが0件の場合は空配列を返す', () => {
      let result: Child[] | undefined;
      service.getChildren().subscribe(d => (result = d));
      httpMock.expectOne(`${base}/children`).flush({ data: [] });
      expect(result).toEqual([]);
    });
  });

  // ── 子ども詳細 ───────────────────────────────────────────────

  describe('getChild', () => {
    it('GET /children/:id を呼び出して子ども詳細を返す', () => {
      let result: Child | undefined;
      service.getChild('c1').subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/children/c1`);
      expect(req.request.method).toBe('GET');
      req.flush({ data: mockChild });

      expect(result).toEqual(mockChild);
    });
  });

  // ── 子ども追加 ───────────────────────────────────────────────

  describe('createChild', () => {
    it('POST /children を正しいボディで呼び出して作成結果を返す', () => {
      const body: CreateChildRequest = { name: 'たろう', age: 8, base_allowance: 1000 };
      let result: Child | undefined;
      service.createChild(body).subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/children`);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(body);
      req.flush({ data: mockChild });

      expect(result).toEqual(mockChild);
    });
  });

  // ── 子ども更新 ───────────────────────────────────────────────

  describe('updateChild', () => {
    it('PUT /children/:id を正しいボディで呼び出して更新結果を返す', () => {
      const body: UpdateChildRequest = { name: 'たろう改', age: 9, base_allowance: 1200 };
      let result: Child | undefined;
      service.updateChild('c1', body).subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/children/c1`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(body);
      req.flush({ data: { ...mockChild, ...body } });

      expect(result?.name).toBe('たろう改');
    });
  });

  // ── 子ども削除 ───────────────────────────────────────────────

  describe('deleteChild', () => {
    it('DELETE /children/:id を呼び出す', () => {
      let completed = false;
      service.deleteChild('c1').subscribe(() => (completed = true));

      const req = httpMock.expectOne(`${base}/children/c1`);
      expect(req.request.method).toBe('DELETE');
      req.flush(null);

      expect(completed).toBeTrue();
    });
  });

  // ── おこづかい種類一覧 ────────────────────────────────────────

  describe('getAllowanceTypes', () => {
    it('GET /allowance-types を呼び出して種類一覧を返す', () => {
      let result: AllowanceType[] | undefined;
      service.getAllowanceTypes().subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/allowance-types`);
      expect(req.request.method).toBe('GET');
      req.flush({ data: mockTypes });

      expect(result).toEqual(mockTypes);
    });
  });

  // ── おこづかい種類詳細 ────────────────────────────────────────

  describe('getAllowanceType', () => {
    it('GET /allowance-types/:id を呼び出して種類詳細を返す', () => {
      let result: AllowanceType | undefined;
      service.getAllowanceType('t1').subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/allowance-types/t1`);
      expect(req.request.method).toBe('GET');
      req.flush({ data: mockType });

      expect(result).toEqual(mockType);
    });
  });

  // ── おこづかい種類追加 ────────────────────────────────────────

  describe('createAllowanceType', () => {
    it('POST /allowance-types を正しいボディで呼び出して作成結果を返す', () => {
      const body: CreateAllowanceTypeRequest = { name: 'お皿洗い', amount: 50 };
      let result: AllowanceType | undefined;
      service.createAllowanceType(body).subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/allowance-types`);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(body);
      req.flush({ data: mockType });

      expect(result).toEqual(mockType);
    });
  });

  // ── おこづかい種類更新 ────────────────────────────────────────

  describe('updateAllowanceType', () => {
    it('PUT /allowance-types/:id を正しいボディで呼び出して更新結果を返す', () => {
      const body: UpdateAllowanceTypeRequest = { name: 'お皿洗い改', amount: 60 };
      let result: AllowanceType | undefined;
      service.updateAllowanceType('t1', body).subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/allowance-types/t1`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(body);
      req.flush({ data: { ...mockType, ...body } });

      expect(result?.amount).toBe(60);
    });
  });

  // ── おこづかい種類削除 ────────────────────────────────────────

  describe('deleteAllowanceType', () => {
    it('DELETE /allowance-types/:id を呼び出す', () => {
      let completed = false;
      service.deleteAllowanceType('t1').subscribe(() => (completed = true));

      const req = httpMock.expectOne(`${base}/allowance-types/t1`);
      expect(req.request.method).toBe('DELETE');
      req.flush(null);

      expect(completed).toBeTrue();
    });
  });

  // ── 収支記録一覧（月フィルタ） ────────────────────────────────

  describe('getRecords', () => {
    it('GET /children/:id/records?year=&month= を正しいクエリパラメータで呼び出す', () => {
      let result: FinancialRecord[] | undefined;
      service.getRecords('c1', 2026, 3).subscribe(d => (result = d));

      const req = httpMock.expectOne(r =>
        r.url === `${base}/children/c1/records` &&
        r.params.get('year') === '2026' &&
        r.params.get('month') === '3'
      );
      expect(req.request.method).toBe('GET');
      req.flush({ data: [mockRecord] });

      expect(result).toEqual([mockRecord]);
    });

    it('収支記録が0件の場合は空配列を返す', () => {
      let result: FinancialRecord[] | undefined;
      service.getRecords('c1', 2026, 3).subscribe(d => (result = d));

      httpMock.expectOne(r => r.url === `${base}/children/c1/records`).flush({ data: [] });

      expect(result).toEqual([]);
    });
  });

  // ── 収支記録追加 ──────────────────────────────────────────────

  describe('createRecord', () => {
    it('POST /children/:id/records を正しいボディで呼び出して登録結果を返す', () => {
      const body: CreateRecordRequest = {
        type: 'income',
        amount: 50,
        description: 'お皿洗い',
        date: '2026-03-27',
        allowance_type_id: 't1',
      };
      let result: FinancialRecord | undefined;
      service.createRecord('c1', body).subscribe(d => (result = d));

      const req = httpMock.expectOne(`${base}/children/c1/records`);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(body);
      req.flush({ data: mockRecord });

      expect(result).toEqual(mockRecord);
    });

    it('allowance_type_id なしでもリクエストを送信できる', () => {
      const body: CreateRecordRequest = { type: 'expense', amount: 200, description: 'おかし', date: '2026-03-27' };
      service.createRecord('c1', body).subscribe();

      const req = httpMock.expectOne(`${base}/children/c1/records`);
      expect(req.request.body).toEqual(body);
      req.flush({ data: { ...mockRecord, type: 'expense', amount: 200 } });
    });
  });

  // ── 収支記録削除 ──────────────────────────────────────────────

  describe('deleteRecord', () => {
    it('DELETE /children/:id/records/:recordId を呼び出す', () => {
      let completed = false;
      service.deleteRecord('c1', 'r1').subscribe(() => (completed = true));

      const req = httpMock.expectOne(`${base}/children/c1/records/r1`);
      expect(req.request.method).toBe('DELETE');
      req.flush(null);

      expect(completed).toBeTrue();
    });
  });
});
