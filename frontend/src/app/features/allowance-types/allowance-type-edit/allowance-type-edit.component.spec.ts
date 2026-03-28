import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog } from '@angular/material/dialog';
import { of, throwError } from 'rxjs';
import { AllowanceTypeEditComponent } from './allowance-type-edit.component';
import { ApiService, AllowanceType } from '../../../core/api.service';

describe('AllowanceTypeEditComponent', () => {
  let component: AllowanceTypeEditComponent;
  let fixture: ComponentFixture<AllowanceTypeEditComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;
  let dialog: MatDialog;

  const mockAllowanceType: AllowanceType = {
    id: 'at-1',
    name: 'お皿洗い',
    amount: 50,
  };

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', [
      'getAllowanceType',
      'updateAllowanceType',
      'deleteAllowanceType',
    ]);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    apiService.getAllowanceType.and.returnValue(of(mockAllowanceType));

    await TestBed.configureTestingModule({
      imports: [AllowanceTypeEditComponent, ReactiveFormsModule, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? 'at-1' : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AllowanceTypeEditComponent);
    component = fixture.componentInstance;
    // 実際の MatDialog インスタンスを取得してスパイする
    dialog = TestBed.inject(MatDialog);
  });

  describe('初期表示', () => {
    it('ngOnInit で getAllowanceType が呼ばれフォームに値がセットされること', fakeAsync(() => {
      fixture.detectChanges();
      tick();

      expect(apiService.getAllowanceType).toHaveBeenCalledWith('at-1');
      expect(component.form.get('name')!.value).toBe('お皿洗い');
      expect(component.form.get('amount')!.value).toBe(50);
    }));

    it('初期ロードエラー時に SnackBar が表示されること', fakeAsync(() => {
      apiService.getAllowanceType.and.returnValue(throwError(() => new Error('API error')));

      fixture.detectChanges();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('種類の情報取得に失敗しました', '閉じる', { duration: 3000 });
    }));
  });

  describe('保存処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('保存で updateAllowanceType が正しい引数で呼ばれ /allowance-types へ遷移すること', fakeAsync(() => {
      apiService.updateAllowanceType.and.returnValue(of(mockAllowanceType));

      component.form.get('name')!.setValue('掃除機がけ');
      component.form.get('amount')!.setValue(100);

      component.save();
      tick();

      expect(apiService.updateAllowanceType).toHaveBeenCalledWith('at-1', {
        name: '掃除機がけ',
        amount: 100,
      });
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types']);
    }));

    it('保存失敗時に SnackBar が表示されること', fakeAsync(() => {
      apiService.updateAllowanceType.and.returnValue(throwError(() => new Error('API error')));

      component.form.get('name')!.setValue('お皿洗い');
      component.form.get('amount')!.setValue(50);

      component.save();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('種類の更新に失敗しました', '閉じる', { duration: 3000 });
    }));
  });

  describe('キャンセル処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('キャンセルで /allowance-types へ遷移すること', () => {
      component.cancel();
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types']);
    });
  });

  describe('削除処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('削除確認ダイアログが開き、確認後 deleteAllowanceType が呼ばれ /allowance-types へ遷移すること', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(true),
      } as any);
      apiService.deleteAllowanceType.and.returnValue(of(void 0));

      component.confirmDelete();
      tick();

      expect(dialog.open).toHaveBeenCalled();
      expect(apiService.deleteAllowanceType).toHaveBeenCalledWith('at-1');
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types']);
    }));

    it('ダイアログでキャンセルした場合 deleteAllowanceType が呼ばれないこと', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(false),
      } as any);

      component.confirmDelete();
      tick();

      expect(apiService.deleteAllowanceType).not.toHaveBeenCalled();
    }));

    it('削除確認ダイアログに正しいタイトルとメッセージが渡されること', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(false),
      } as any);

      component.confirmDelete();
      tick();

      const dialogArgs = (dialog.open as jasmine.Spy).calls.mostRecent().args;
      const config = dialogArgs[1] as { data?: { title?: string; message?: string } };
      expect(config?.data?.title).toBe('種類を削除');
      expect(config?.data?.message).toBe('この種類を削除しますか？この操作は取り消せません。');
    }));

    it('削除失敗時に SnackBar が表示されること', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(true),
      } as any);
      apiService.deleteAllowanceType.and.returnValue(throwError(() => new Error('API error')));

      component.confirmDelete();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('種類の削除に失敗しました', '閉じる', { duration: 3000 });
    }));
  });
});
