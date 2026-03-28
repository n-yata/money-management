import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { of, throwError, Subject } from 'rxjs';
import { AllowanceTypeNewComponent } from './allowance-type-new.component';
import { ApiService, AllowanceType } from '../../../core/api.service';

describe('AllowanceTypeNewComponent', () => {
  let component: AllowanceTypeNewComponent;
  let fixture: ComponentFixture<AllowanceTypeNewComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;

  const mockAllowanceType: AllowanceType = {
    id: 'at-1',
    name: 'お皿洗い',
    amount: 50,
  };

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', ['createAllowanceType']);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    await TestBed.configureTestingModule({
      imports: [AllowanceTypeNewComponent, ReactiveFormsModule, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AllowanceTypeNewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  describe('フォームバリデーション', () => {
    it('フォームが空の状態で invalid であること', () => {
      expect(component.form.invalid).toBeTrue();
    });

    it('name の maxLength(30) 超過で invalid になること', () => {
      component.form.get('name')!.setValue('a'.repeat(31));
      component.form.get('amount')!.setValue(50);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('name')!.hasError('maxlength')).toBeTrue();
    });

    it('amount が 0 のとき invalid になること', () => {
      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(0);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('amount')!.hasError('min')).toBeTrue();
    });

    it('amount が負の値のとき invalid になること', () => {
      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(-1);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('amount')!.hasError('min')).toBeTrue();
    });

    it('すべて有効値のとき valid であること', () => {
      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(50);
      expect(component.form.valid).toBeTrue();
    });

    it('amount が 1 のとき valid であること（境界値）', () => {
      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(1);
      expect(component.form.valid).toBeTrue();
    });

    it('name が 30 文字のとき valid であること（境界値）', () => {
      component.form.get('name')!.setValue('a'.repeat(30));
      component.form.get('amount')!.setValue(50);
      expect(component.form.valid).toBeTrue();
    });
  });

  describe('保存処理', () => {
    beforeEach(() => {
      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(50);
    });

    it('保存成功時に createAllowanceType が正しい引数で呼ばれ /allowance-types へ遷移すること', fakeAsync(() => {
      apiService.createAllowanceType.and.returnValue(of(mockAllowanceType));

      component.save();
      tick();

      expect(apiService.createAllowanceType).toHaveBeenCalledWith({
        name: 'お皿洗い',
        amount: 50,
      });
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types']);
    }));

    it('保存失敗時に SnackBar が表示されること', fakeAsync(() => {
      apiService.createAllowanceType.and.returnValue(throwError(() => new Error('API error')));

      component.save();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('種類の追加に失敗しました', '閉じる', { duration: 3000 });
    }));

    it('API通信中は loading が true になること', fakeAsync(() => {
      const subject = new Subject<AllowanceType>();
      apiService.createAllowanceType.and.returnValue(subject.asObservable());

      component.save();

      // save() 呼び出し直後（レスポンス前）は loading が true
      expect(component.loading()).toBeTrue();

      subject.next(mockAllowanceType);
      subject.complete();
      tick();

      expect(component.loading()).toBeFalse();
    }));

    it('フォームが invalid のとき save() を呼んでも createAllowanceType が呼ばれないこと', () => {
      component.form.get('name')!.setValue('');

      component.save();

      expect(apiService.createAllowanceType).not.toHaveBeenCalled();
    });
  });

  describe('キャンセル処理', () => {
    it('キャンセルで /allowance-types へ遷移すること', () => {
      component.cancel();
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types']);
    });
  });
});
