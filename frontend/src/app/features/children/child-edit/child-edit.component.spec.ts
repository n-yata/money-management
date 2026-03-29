import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { of, throwError } from 'rxjs';
import { ChildEditComponent } from './child-edit.component';
import { ApiService, Child } from '../../../core/api.service';

describe('ChildEditComponent', () => {
  let component: ChildEditComponent;
  let fixture: ComponentFixture<ChildEditComponent>;
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

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', ['getChild', 'updateChild', 'deleteChild']);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    apiService.getChild.and.returnValue(of(mockChild));

    await TestBed.configureTestingModule({
      imports: [ChildEditComponent, ReactiveFormsModule, NoopAnimationsModule],
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

    fixture = TestBed.createComponent(ChildEditComponent);
    component = fixture.componentInstance;
    // 実際の MatDialog インスタンスを取得してスパイする
    dialog = TestBed.inject(MatDialog);
  });

  describe('初期表示', () => {
    it('ngOnInit で getChild が呼ばれフォームに値がセットされること', fakeAsync(() => {
      fixture.detectChanges();
      tick();

      expect(apiService.getChild).toHaveBeenCalledWith('child-1');
      expect(component.form.get('name')!.value).toBe('たろう');
      expect(component.form.get('age')!.value).toBe(8);
      expect(component.form.get('base_allowance')!.value).toBe(1000);
    }));
  });

  describe('保存処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('保存で updateChild が正しい引数で呼ばれ /children/:id へ遷移すること', fakeAsync(() => {
      apiService.updateChild.and.returnValue(of(mockChild));

      component.form.get('name')!.setValue('はなこ');
      component.form.get('age')!.setValue(10);
      component.form.get('base_allowance')!.setValue(1500);

      component.save();
      tick();

      expect(apiService.updateChild).toHaveBeenCalledWith('child-1', {
        name: 'はなこ',
        age: 10,
        base_allowance: 1500,
      });
      expect(router.navigate).toHaveBeenCalledWith(['/children', 'child-1']);
    }));

    it('保存失敗時に SnackBar が表示されること', fakeAsync(() => {
      apiService.updateChild.and.returnValue(throwError(() => new Error('API error')));

      // フォームに有効な値を明示的にセット
      component.form.get('name')!.setValue('たろう');
      component.form.get('age')!.setValue(8);
      component.form.get('base_allowance')!.setValue(1000);

      component.save();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith(
        '子どもの情報更新に失敗しました',
        '閉じる',
        { duration: 3000 }
      );
    }));
  });

  describe('キャンセル処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('キャンセルで /children/:id へ遷移すること', () => {
      component.cancel();
      expect(router.navigate).toHaveBeenCalledWith(['/children', 'child-1']);
    });
  });

  describe('削除処理', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('削除確認ダイアログが開き、確認後 deleteChild が呼ばれ /dashboard へ遷移すること', fakeAsync(() => {
      // 実際の MatDialog.open をスパイして確認ダイアログの結果を制御する
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(true),
      } as unknown as MatDialogRef<unknown>);
      apiService.deleteChild.and.returnValue(of(void 0));

      component.confirmDelete();
      tick();

      expect(dialog.open).toHaveBeenCalled();
      expect(apiService.deleteChild).toHaveBeenCalledWith('child-1');
      expect(router.navigate).toHaveBeenCalledWith(['/dashboard']);
    }));

    it('ダイアログでキャンセルした場合 deleteChild が呼ばれないこと', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(false),
      } as unknown as MatDialogRef<unknown>);

      component.confirmDelete();
      tick();

      expect(apiService.deleteChild).not.toHaveBeenCalled();
    }));

    it('削除確認ダイアログに正しいメッセージが渡されること', fakeAsync(() => {
      spyOn(dialog, 'open').and.returnValue({
        afterClosed: () => of(false),
      } as unknown as MatDialogRef<unknown>);

      component.confirmDelete();
      tick();

      const dialogArgs = (dialog.open as jasmine.Spy).calls.mostRecent().args;
      const config = dialogArgs[1] as { data?: { title?: string; message?: string } };
      expect(config?.data?.message).toBe('本当に削除しますか？この操作は取り消せません。');
    }));
  });
});
