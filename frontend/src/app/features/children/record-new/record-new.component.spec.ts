import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { provideNativeDateAdapter } from '@angular/material/core';
import { of, throwError, Subject } from 'rxjs';
import { RecordNewComponent } from './record-new.component';
import { ApiService, AllowanceType, Record as ApiRecord } from '../../../core/api.service';

describe('RecordNewComponent', () => {
  let component: RecordNewComponent;
  let fixture: ComponentFixture<RecordNewComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;

  const mockChildId = 'child-123';

  const mockAllowanceTypes: AllowanceType[] = [
    { id: 'type-1', name: 'お皿洗い', amount: 50 },
    { id: 'type-2', name: 'ゴミ出し', amount: 100 },
  ];

  const mockRecord: ApiRecord = {
    id: 'record-1',
    type: 'income',
    amount: 50,
    description: 'お皿洗い',
    date: '2026-03-28',
  };

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', [
      'getAllowanceTypes',
      'createRecord',
    ]);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    // デフォルトは成功レスポンス
    apiService.getAllowanceTypes.and.returnValue(of(mockAllowanceTypes));

    await TestBed.configureTestingModule({
      imports: [RecordNewComponent, ReactiveFormsModule, NoopAnimationsModule],
      providers: [
        provideNativeDateAdapter(),
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: { get: (_: string) => mockChildId } },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RecordNewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  describe('初期化', () => {
    it('ngOnInit で getAllowanceTypes が呼ばれること', () => {
      expect(apiService.getAllowanceTypes).toHaveBeenCalledTimes(1);
    });

    it('getAllowanceTypes の結果が allowanceTypes シグナルに格納されること', () => {
      expect(component.allowanceTypes()).toEqual(mockAllowanceTypes);
    });

    it('フォームのデフォルト値が設定されること（type: income、日付: 今日）', () => {
      expect(component.form.get('type')!.value).toBe('income');
      expect(component.form.get('allowance_type_id')!.value).toBeNull();
      expect(component.form.get('amount')!.value).toBeNull();
      expect(component.form.get('description')!.value).toBe('');
      // date は今日の Date オブジェクト
      const dateValue = component.form.get('date')!.value as Date;
      const today = new Date();
      expect(dateValue.getFullYear()).toBe(today.getFullYear());
      expect(dateValue.getMonth()).toBe(today.getMonth());
      expect(dateValue.getDate()).toBe(today.getDate());
    });
  });

  describe('種類セレクトの表示制御', () => {
    it('type が income のとき isIncome が true になること', () => {
      component.form.get('type')!.setValue('income');
      expect(component.isIncome).toBeTrue();
    });

    it('type が expense のとき isIncome が false になること', () => {
      component.form.get('type')!.setValue('expense');
      expect(component.isIncome).toBeFalse();
    });

    it('type を expense に切り替えると allowance_type_id がリセットされること', fakeAsync(() => {
      // まず収入にして種類を選択
      component.form.get('type')!.setValue('income');
      component.form.get('allowance_type_id')!.setValue('type-1');
      tick();

      // 支出に切り替える
      component.form.get('type')!.setValue('expense');
      tick();

      expect(component.form.get('allowance_type_id')!.value).toBeNull();
    }));
  });

  describe('種類選択による自動入力', () => {
    it('種類を選択すると金額・説明が自動入力されること', fakeAsync(() => {
      component.form.get('type')!.setValue('income');
      component.form.get('allowance_type_id')!.setValue('type-1');
      tick();

      expect(component.form.get('amount')!.value).toBe(50);
      expect(component.form.get('description')!.value).toBe('お皿洗い');
    }));

    it('別の種類を選択すると金額・説明が上書きされること', fakeAsync(() => {
      component.form.get('type')!.setValue('income');
      component.form.get('allowance_type_id')!.setValue('type-1');
      tick();

      component.form.get('allowance_type_id')!.setValue('type-2');
      tick();

      expect(component.form.get('amount')!.value).toBe(100);
      expect(component.form.get('description')!.value).toBe('ゴミ出し');
    }));

    it('自動入力後に手動で金額を変更できること', fakeAsync(() => {
      component.form.get('type')!.setValue('income');
      component.form.get('allowance_type_id')!.setValue('type-1');
      tick();

      // 手動で変更
      component.form.get('amount')!.setValue(200);
      expect(component.form.get('amount')!.value).toBe(200);
    }));
  });

  describe('フォームバリデーション', () => {
    it('amount が null のとき invalid になること', () => {
      component.form.get('amount')!.setValue(null);
      component.form.get('description')!.setValue('テスト');
      component.form.get('date')!.setValue(new Date());
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('amount')!.hasError('required')).toBeTrue();
    });

    it('amount が 0（min 未満）のとき invalid になること', () => {
      component.form.get('amount')!.setValue(0);
      component.form.get('description')!.setValue('テスト');
      component.form.get('date')!.setValue(new Date());
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('amount')!.hasError('min')).toBeTrue();
    });

    it('amount が 1 のとき valid になること（境界値）', () => {
      component.form.get('amount')!.setValue(1);
      component.form.get('description')!.setValue('テスト');
      component.form.get('date')!.setValue(new Date());
      expect(component.form.valid).toBeTrue();
    });

    it('description が空のとき invalid になること', () => {
      component.form.get('amount')!.setValue(100);
      component.form.get('description')!.setValue('');
      component.form.get('date')!.setValue(new Date());
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('description')!.hasError('required')).toBeTrue();
    });

    it('description が 51 文字のとき invalid になること', () => {
      component.form.get('amount')!.setValue(100);
      component.form.get('description')!.setValue('a'.repeat(51));
      component.form.get('date')!.setValue(new Date());
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('description')!.hasError('maxlength')).toBeTrue();
    });

    it('すべて有効値のとき valid になること', () => {
      component.form.get('type')!.setValue('income');
      component.form.get('amount')!.setValue(100);
      component.form.get('description')!.setValue('テスト');
      component.form.get('date')!.setValue(new Date());
      expect(component.form.valid).toBeTrue();
    });
  });

  describe('保存処理', () => {
    const testDate = new Date(2026, 2, 28); // 2026-03-28

    beforeEach(() => {
      component.form.get('type')!.setValue('income');
      component.form.get('allowance_type_id')!.setValue(null);
      component.form.get('amount')!.setValue(100);
      component.form.get('description')!.setValue('テスト記録');
      component.form.get('date')!.setValue(testDate);
    });

    it('種類未選択時に createRecord が allowance_type_id なしで呼ばれること', fakeAsync(() => {
      apiService.createRecord.and.returnValue(of(mockRecord));

      component.save();
      tick();

      expect(apiService.createRecord).toHaveBeenCalledWith(mockChildId, {
        type: 'income',
        amount: 100,
        description: 'テスト記録',
        date: '2026-03-28',
      });
    }));

    it('種類選択時に createRecord が allowance_type_id ありで呼ばれること', fakeAsync(() => {
      component.form.get('allowance_type_id')!.setValue('type-1');
      tick(); // valueChanges による自動入力を待つ

      // 手動で値を上書きして期待値を固定
      component.form.get('amount')!.setValue(100);
      component.form.get('description')!.setValue('テスト記録');

      apiService.createRecord.and.returnValue(of(mockRecord));

      component.save();
      tick();

      expect(apiService.createRecord).toHaveBeenCalledWith(mockChildId, {
        type: 'income',
        amount: 100,
        description: 'テスト記録',
        date: '2026-03-28',
        allowance_type_id: 'type-1',
      });
    }));

    it('date が YYYY-MM-DD 形式の文字列で送られること', fakeAsync(() => {
      apiService.createRecord.and.returnValue(of(mockRecord));

      component.save();
      tick();

      const callArgs = apiService.createRecord.calls.mostRecent().args;
      const body = callArgs[1] as { date: string };
      expect(body.date).toBe('2026-03-28');
      expect(body.date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    }));

    it('保存成功後に /children/:id へ遷移すること', fakeAsync(() => {
      apiService.createRecord.and.returnValue(of(mockRecord));

      component.save();
      tick();

      expect(router.navigate).toHaveBeenCalledWith(['/children', mockChildId]);
    }));

    it('保存失敗時に SnackBar が表示されること', fakeAsync(() => {
      apiService.createRecord.and.returnValue(throwError(() => new Error('API error')));

      component.save();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith(
        '収支記録の追加に失敗しました',
        '閉じる',
        { duration: 3000 }
      );
    }));

    it('API通信中は loading が true になること', fakeAsync(() => {
      const subject = new Subject<ApiRecord>();
      apiService.createRecord.and.returnValue(subject.asObservable());

      component.save();

      // save() 呼び出し直後（レスポンス前）は loading が true
      expect(component.loading()).toBeTrue();

      subject.next(mockRecord);
      subject.complete();
      tick();

      expect(component.loading()).toBeFalse();
    }));

    it('フォームが invalid の場合は createRecord が呼ばれないこと', () => {
      component.form.get('amount')!.setValue(null);

      component.save();

      expect(apiService.createRecord).not.toHaveBeenCalled();
    });
  });

  describe('キャンセル処理', () => {
    it('キャンセルで /children/:id へ遷移すること', () => {
      component.cancel();
      expect(router.navigate).toHaveBeenCalledWith(['/children', mockChildId]);
    });
  });
});
