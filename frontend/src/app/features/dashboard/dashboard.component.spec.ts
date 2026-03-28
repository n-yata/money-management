import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of, throwError } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { DashboardComponent } from './dashboard.component';
import { ApiService, Child } from '../../core/api.service';

const mockChildren: Child[] = [
  { id: 'c1', name: 'たろう', age: 8, base_allowance: 1000, balance: 1200 },
  { id: 'c2', name: 'はなこ', age: 6, base_allowance: 800, balance: 500 },
];

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;
  let apiServiceSpy: jasmine.SpyObj<ApiService>;
  let routerSpy: jasmine.SpyObj<Router>;
  let snackBar: MatSnackBar;

  beforeEach(async () => {
    apiServiceSpy = jasmine.createSpyObj<ApiService>('ApiService', ['getChildren']);
    apiServiceSpy.getChildren.and.returnValue(of(mockChildren));

    routerSpy = jasmine.createSpyObj<Router>('Router', ['navigate']);

    await TestBed.configureTestingModule({
      imports: [DashboardComponent, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiServiceSpy },
        { provide: Router, useValue: routerSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardComponent);
    component = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    snackBar = (component as any)['snackBar'] as MatSnackBar;
    spyOn(snackBar, 'open');

    fixture.detectChanges();
  });

  // ── 初期化 ──────────────────────────────────────────────────

  describe('初期化', () => {
    it('コンポーネントが生成できる', () => {
      expect(component).toBeTruthy();
    });

    it('ngOnInit で子ども一覧を取得する', () => {
      expect(apiServiceSpy.getChildren).toHaveBeenCalledTimes(1);
    });

    it('取得した子ども一覧がシグナルに反映される', () => {
      expect(component.children()).toEqual(mockChildren);
    });

    it('ローディング完了後は loading が false になる', () => {
      expect(component.loading()).toBeFalse();
    });
  });

  // ── 画面遷移 ─────────────────────────────────────────────────

  describe('画面遷移', () => {
    it('goToChoreRegister で / へ遷移する', () => {
      component.goToChoreRegister();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/']);
    });

    it('goToAllowanceTypes で /allowance-types へ遷移する', () => {
      component.goToAllowanceTypes();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/allowance-types']);
    });

    it('goToChildDetail で /children/:id へ遷移する', () => {
      component.goToChildDetail(mockChildren[0]);
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/children', 'c1']);
    });

    it('goToChildNew で /children/new へ遷移する', () => {
      component.goToChildNew();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/children/new']);
    });
  });

  // ── エラーハンドリング ────────────────────────────────────────

  describe('エラーハンドリング', () => {
    it('getChildren がエラーの場合は SnackBar を表示する', () => {
      apiServiceSpy.getChildren.and.returnValue(throwError(() => new Error('500')));

      const errorFixture = TestBed.createComponent(DashboardComponent);
      errorFixture.detectChanges();

      expect(snackBar.open).toHaveBeenCalledWith(
        '子ども一覧の取得に失敗しました',
        '閉じる',
        { duration: 3000 }
      );

      errorFixture.destroy();
    });

    it('API エラー時は loading が false になる', () => {
      apiServiceSpy.getChildren.and.returnValue(throwError(() => new Error('500')));

      const errorFixture = TestBed.createComponent(DashboardComponent);
      const errorComponent = errorFixture.componentInstance;
      errorFixture.detectChanges();

      expect(errorComponent.loading()).toBeFalse();

      errorFixture.destroy();
    });
  });

  // ── テンプレート表示 ─────────────────────────────────────────

  describe('テンプレート表示', () => {
    it('子どもカードが人数分表示される', () => {
      const cards = fixture.nativeElement.querySelectorAll('.child-card');
      expect(cards.length).toBe(mockChildren.length);
    });

    it('子どもの名前・年齢・残高がカードに表示される', () => {
      const cards: NodeListOf<HTMLElement> = fixture.nativeElement.querySelectorAll('.child-card');
      expect(cards[0].textContent).toContain('たろう');
      expect(cards[0].textContent).toContain('8歳');
      expect(cards[0].textContent).toContain('1,200');
    });

    it('子どもが0人の場合は空メッセージを表示する', () => {
      apiServiceSpy.getChildren.and.returnValue(of([]));
      const emptyFixture = TestBed.createComponent(DashboardComponent);
      emptyFixture.detectChanges();

      const msg: HTMLElement = emptyFixture.nativeElement.querySelector('.empty-message');
      expect(msg).toBeTruthy();
      expect(msg.textContent).toContain('子どもが登録されていません');

      emptyFixture.destroy();
    });
  });
});
