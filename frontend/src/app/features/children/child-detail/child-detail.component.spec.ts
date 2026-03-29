import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { of, throwError } from 'rxjs';
import { ChildDetailComponent } from './child-detail.component';
import { ApiService, Child, FinancialRecord } from '../../../core/api.service';

describe('ChildDetailComponent', () => {
  let component: ChildDetailComponent;
  let fixture: ComponentFixture<ChildDetailComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;
  let dialog: MatDialog;

  const mockChild: Child = {
    id: 'child-1',
    name: 'たろう',
    age: 8,
    base_allowance: 1000,
    balance: 1200,
  };

  const mockRecords: FinancialRecord[] = [
    { id: 'r1', type: 'income', amount: 500, description: 'お手伝い', date: '2026-03-20', created_at: '2026-03-20T10:00:00Z' },
    { id: 'r2', type: 'expense', amount: 300, description: 'おやつ', date: '2026-03-15', created_at: '2026-03-15T10:00:00Z' },
  ];

  beforeEach(async () => {
    apiService = jasmine.createSpyObj<ApiService>('ApiService', ['getChild', 'getRecords', 'deleteRecord']);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    // デフォルトの成功レスポンス
    apiService.getChild.and.returnValue(of(mockChild));
    apiService.getRecords.and.returnValue(of(mockRecords));

    await TestBed.configureTestingModule({
      imports: [ChildDetailComponent, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? 'child-1' : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ChildDetailComponent);
    component = fixture.componentInstance;
    // 実際の MatDialog インスタンスを取得してスパイする
    dialog = TestBed.inject(MatDialog);
  });

  describe('初期表示', () => {
    it('ngOnInit で getChild と getRecords（当月）が呼ばれること', fakeAsync(() => {
      const now = new Date();
      const currentYear = now.getFullYear();
      const currentMonth = now.getMonth() + 1;

      fixture.detectChanges();
      tick();

      expect(apiService.getChild).toHaveBeenCalledWith('child-1');
      expect(apiService.getRecords).toHaveBeenCalledWith('child-1', currentYear, currentMonth);
    }));

    it('取得した child と records がシグナルに反映されること', fakeAsync(() => {
      fixture.detectChanges();
      tick();

      expect(component.child()).toEqual(mockChild);
      // 日付降順でソートされていることを確認
      expect(component.records()[0].id).toBe('r1');
      expect(component.records()[1].id).toBe('r2');
    }));

    it('getChild エラー時に SnackBar が表示されること', fakeAsync(() => {
      apiService.getChild.and.returnValue(throwError(() => new Error('API error')));

      fixture.detectChanges();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith(
        'データの取得に失敗しました',
        '閉じる',
        { duration: 3000 }
      );
    }));
  });

  describe('月フィルタ', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
      // 初期呼び出しのカウントをリセット
      apiService.getRecords.calls.reset();
    }));

    it('月フィルタ変更で getRecords が新しい year/month で呼ばれること', fakeAsync(() => {
      const oldOption = { year: 2026, month: 1, label: '2026年1月' };
      component.onMonthChange(oldOption);
      tick();

      expect(apiService.getRecords).toHaveBeenCalledWith('child-1', 2026, 1);
    }));
  });

  describe('レコード削除', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('確認ダイアログを開き、確認後 deleteRecord が正しい引数で呼ばれること', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(true),
      } as unknown as MatDialogRef<unknown>);
      apiService.deleteRecord.and.returnValue(of(void 0));

      component.deleteRecord(mockRecords[0]);
      tick();

      expect(dialog.open).toHaveBeenCalled();
      expect(apiService.deleteRecord).toHaveBeenCalledWith('child-1', 'r1');
    }));

    it('ダイアログでキャンセルした場合 deleteRecord が呼ばれないこと', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(false),
      } as unknown as MatDialogRef<unknown>);

      component.deleteRecord(mockRecords[0]);
      tick();

      expect(apiService.deleteRecord).not.toHaveBeenCalled();
    }));
  });

  describe('画面遷移', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('goToDashboard で /dashboard へ遷移すること', () => {
      component.goToDashboard();
      expect(router.navigate).toHaveBeenCalledWith(['/dashboard']);
    });

    it('goToEdit で /children/:id/edit へ遷移すること', () => {
      component.goToEdit();
      expect(router.navigate).toHaveBeenCalledWith(['/children', 'child-1', 'edit']);
    });

    it('goToRecordNew で /children/:id/records/new へ遷移すること', () => {
      component.goToRecordNew();
      expect(router.navigate).toHaveBeenCalledWith(['/children', 'child-1', 'records', 'new']);
    });
  });
});
