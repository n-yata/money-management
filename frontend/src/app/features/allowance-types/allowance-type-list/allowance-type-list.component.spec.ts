import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBar } from '@angular/material/snack-bar';
import { of, throwError } from 'rxjs';
import { AllowanceTypeListComponent } from './allowance-type-list.component';
import { ApiService, AllowanceType } from '../../../core/api.service';

const mockAllowanceTypes: AllowanceType[] = [
  { id: 'at-1', name: 'お皿洗い', amount: 50 },
  { id: 'at-2', name: '掃除機がけ', amount: 100 },
];

describe('AllowanceTypeListComponent', () => {
  let component: AllowanceTypeListComponent;
  let fixture: ComponentFixture<AllowanceTypeListComponent>;
  let apiService: jasmine.SpyObj<ApiService>;
  let router: jasmine.SpyObj<Router>;
  let snackBar: jasmine.SpyObj<MatSnackBar>;

  beforeEach(async () => {
    apiService = jasmine.createSpyObj('ApiService', ['getAllowanceTypes']);
    router = jasmine.createSpyObj('Router', ['navigate']);
    snackBar = jasmine.createSpyObj('MatSnackBar', ['open']);

    apiService.getAllowanceTypes.and.returnValue(of(mockAllowanceTypes));

    await TestBed.configureTestingModule({
      imports: [AllowanceTypeListComponent, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: Router, useValue: router },
        { provide: MatSnackBar, useValue: snackBar },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AllowanceTypeListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  describe('初期表示', () => {
    it('ngOnInit で getAllowanceTypes が呼ばれること', () => {
      expect(apiService.getAllowanceTypes).toHaveBeenCalledTimes(1);
    });

    it('取得した種類一覧がシグナルに反映されること', fakeAsync(() => {
      tick();
      expect(component.allowanceTypes()).toEqual(mockAllowanceTypes);
    }));

    it('ローディング完了後は loading が false になること', fakeAsync(() => {
      tick();
      expect(component.loading()).toBeFalse();
    }));
  });

  describe('APIエラー時', () => {
    it('getAllowanceTypes がエラーの場合は SnackBar が表示されること', fakeAsync(() => {
      apiService.getAllowanceTypes.and.returnValue(throwError(() => new Error('API error')));

      const errorFixture = TestBed.createComponent(AllowanceTypeListComponent);
      errorFixture.detectChanges();
      tick();

      expect(snackBar.open).toHaveBeenCalledWith('種類一覧の取得に失敗しました', '閉じる', { duration: 3000 });

      errorFixture.destroy();
    }));

    it('API エラー時は loading が false になること', fakeAsync(() => {
      apiService.getAllowanceTypes.and.returnValue(throwError(() => new Error('API error')));

      const errorFixture = TestBed.createComponent(AllowanceTypeListComponent);
      const errorComponent = errorFixture.componentInstance;
      errorFixture.detectChanges();
      tick();

      expect(errorComponent.loading()).toBeFalse();

      errorFixture.destroy();
    }));
  });

  describe('画面遷移', () => {
    it('goToDashboard で /dashboard へ遷移すること', () => {
      component.goToDashboard();
      expect(router.navigate).toHaveBeenCalledWith(['/dashboard']);
    });

    it('goToNew で /allowance-types/new へ遷移すること', () => {
      component.goToNew();
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types/new']);
    });

    it('goToEdit で /allowance-types/:id/edit へ遷移すること', () => {
      component.goToEdit(mockAllowanceTypes[0]);
      expect(router.navigate).toHaveBeenCalledWith(['/allowance-types', 'at-1', 'edit']);
    });
  });
});
