import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { of, throwError, Subject } from 'rxjs';
import { ChildNewComponent } from './child-new.component';
import { ApiService, Child } from '../../../core/api.service';

describe('ChildNewComponent', () => {
  let component: ChildNewComponent;
  let fixture: ComponentFixture<ChildNewComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;

  const mockChild: Child = {
    id: 'child-1',
    name: 'たろう',
    age: 8,
    base_allowance: 1000,
    balance: 0,
  };

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', ['createChild']);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    await TestBed.configureTestingModule({
      imports: [ChildNewComponent, ReactiveFormsModule, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ChildNewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  describe('フォームバリデーション', () => {
    it('フォームが空の状態で invalid であること', () => {
      expect(component.form.invalid).toBeTrue();
    });

    it('name の maxLength(20) 超過で invalid になること', () => {
      component.form.get('name')!.setValue('a'.repeat(21));
      component.form.get('age')!.setValue(8);
      component.form.get('base_allowance')!.setValue(1000);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('name')!.hasError('maxlength')).toBeTrue();
    });

    it('age が 0 のとき invalid になること', () => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(0);
      component.form.get('base_allowance')!.setValue(1000);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('age')!.hasError('min')).toBeTrue();
    });

    it('age が 19 のとき invalid になること', () => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(19);
      component.form.get('base_allowance')!.setValue(1000);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('age')!.hasError('max')).toBeTrue();
    });

    it('base_allowance が -1 のとき invalid になること', () => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(8);
      component.form.get('base_allowance')!.setValue(-1);
      expect(component.form.invalid).toBeTrue();
      expect(component.form.get('base_allowance')!.hasError('min')).toBeTrue();
    });

    it('すべて有効値のとき valid であること', () => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(8);
      component.form.get('base_allowance')!.setValue(1000);
      expect(component.form.valid).toBeTrue();
    });

    it('base_allowance が 0 のとき valid であること（境界値）', () => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(1);
      component.form.get('base_allowance')!.setValue(0);
      expect(component.form.valid).toBeTrue();
    });
  });

  describe('保存処理', () => {
    beforeEach(() => {
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(8);
      component.form.get('base_allowance')!.setValue(1000);
    });

    it('保存成功時に createChild が正しい引数で呼ばれ /dashboard へ遷移すること', fakeAsync(() => {
      apiService.createChild.and.returnValue(of(mockChild));

      component.save();
      tick();

      expect(apiService.createChild).toHaveBeenCalledWith({
        name: 'たろう',
        age: 8,
        base_allowance: 1000,
      });
      expect(router.navigate).toHaveBeenCalledWith(['/dashboard']);
    }));

    it('保存失敗時に SnackBar が表示されること', fakeAsync(() => {
      apiService.createChild.and.returnValue(throwError(() => new Error('API error')));

      component.save();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('子どもの追加に失敗しました', '閉じる', { duration: 3000 });
    }));

    it('API通信中は loading が true になること', fakeAsync(() => {
      // Subject を使って非同期レスポンスを制御する
      const subject = new Subject<Child>();
      apiService.createChild.and.returnValue(subject.asObservable());

      component.save();

      // save() 呼び出し直後（レスポンス前）は loading が true
      expect(component.loading()).toBeTrue();

      // レスポンスを発行
      subject.next(mockChild);
      subject.complete();
      tick();

      expect(component.loading()).toBeFalse();
    }));
  });

  describe('キャンセル処理', () => {
    it('キャンセルボタンクリックで /dashboard へ遷移すること', () => {
      component.cancel();
      expect(router.navigate).toHaveBeenCalledWith(['/dashboard']);
    });
  });
});
