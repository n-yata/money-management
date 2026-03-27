import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import { ApiService, Child, AllowanceType, Record, CreateRecordRequest } from './api.service';
import { environment } from '../../environments/environment';

describe('ApiService', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;
  const base = environment.apiBaseUrl;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
    });
    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    // 未処理のリクエストが残っていないことを保証する
    httpMock.verify();
  });

  // ── getChildren ─────────────────────────────────────────────

  describe('getChildren', () => {
    it('GET /children を呼び出して子ども一覧を返す', () => {
      const mockChildren: Child[] = [
        { id: 'c1', name: 'たろう', age: 8, base_allowance: 1000, balance: 1200 },
        { id: 'c2', name: 'はなこ', age: 6, base_allowance: 800, balance: 500 },
      ];

      let result: Child[] | undefined;
      service.getChildren().subscribe((data) => (result = data));

      const req = httpMock.expectOne(`${base}/children`);
      expect(req.request.method).toBe('GET');
      req.flush(mockChildren);

      expect(result).toEqual(mockChildren);
    });

    it('子どもが0件の場合は空配列を返す', () => {
      let result: Child[] | undefined;
      service.getChildren().subscribe((data) => (result = data));

      const req = httpMock.expectOne(`${base}/children`);
      req.flush([]);

      expect(result).toEqual([]);
    });
  });

  // ── getAllowanceTypes ────────────────────────────────────────

  describe('getAllowanceTypes', () => {
    it('GET /allowance-types を呼び出して種類一覧を返す', () => {
      const mockTypes: AllowanceType[] = [
        { id: 't1', name: 'お皿洗い', amount: 50 },
        { id: 't2', name: '掃除機かけ', amount: 80 },
      ];

      let result: AllowanceType[] | undefined;
      service.getAllowanceTypes().subscribe((data) => (result = data));

      const req = httpMock.expectOne(`${base}/allowance-types`);
      expect(req.request.method).toBe('GET');
      req.flush(mockTypes);

      expect(result).toEqual(mockTypes);
    });
  });

  // ── createRecord ────────────────────────────────────────────

  describe('createRecord', () => {
    const childId = 'c1';
    const reqBody: CreateRecordRequest = {
      type: 'income',
      amount: 50,
      description: 'お皿洗い',
      date: '2026-03-27',
      allowance_type_id: 't1',
    };
    const mockRecord: Record = {
      id: 'r1',
      type: 'income',
      amount: 50,
      description: 'お皿洗い',
      date: '2026-03-27',
      allowance_type_id: 't1',
    };

    it('POST /children/:id/records を正しいボディで呼び出して登録結果を返す', () => {
      let result: Record | undefined;
      service.createRecord(childId, reqBody).subscribe((data) => (result = data));

      const req = httpMock.expectOne(`${base}/children/${childId}/records`);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(reqBody);
      req.flush(mockRecord);

      expect(result).toEqual(mockRecord);
    });

    it('allowance_type_id なしでもリクエストを送信できる', () => {
      const bodyWithoutType: CreateRecordRequest = {
        type: 'income',
        amount: 100,
        description: 'テスト',
        date: '2026-03-27',
      };

      let result: Record | undefined;
      service.createRecord(childId, bodyWithoutType).subscribe((data) => (result = data));

      const req = httpMock.expectOne(`${base}/children/${childId}/records`);
      expect(req.request.body).toEqual(bodyWithoutType);
      req.flush({ ...mockRecord, allowance_type_id: undefined });

      expect(result).toBeDefined();
    });
  });
});
