import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of, throwError } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ChoreRegisterComponent } from './chore-register.component';
import { ApiService, Child, AllowanceType } from '../../core/api.service';

// テスト用フィクスチャデータ
const mockChildren: Child[] = [
  { id: 'c1', name: 'たろう', age: 8, base_allowance: 1000, balance: 1200 },
  { id: 'c2', name: 'はなこ', age: 6, base_allowance: 800, balance: 500 },
];

const mockAllowanceTypes: AllowanceType[] = [
  { id: 't1', name: 'お皿洗い', amount: 50 },
  { id: 't2', name: '掃除機かけ', amount: 80 },
];

describe('ChoreRegisterComponent', () => {
  let component: ChoreRegisterComponent;
  let fixture: ComponentFixture<ChoreRegisterComponent>;
  let apiServiceSpy: jasmine.SpyObj<ApiService>;
  let routerSpy: jasmine.SpyObj<Router>;
  // standalone component が MatSnackBarModule を import するため、TestBed root injector の
  // provider が component 内の inject() に届かない。
  // そのため component インスタンスから実際の MatSnackBar を取得して spyOn する。
  let snackBar: MatSnackBar;

  beforeEach(async () => {
    // ApiService はスタブ化して HTTP 通信を発生させない
    apiServiceSpy = jasmine.createSpyObj<ApiService>('ApiService', [
      'getChildren',
      'getAllowanceTypes',
      'createRecord',
    ]);
    apiServiceSpy.getChildren.and.returnValue(of(mockChildren));
    apiServiceSpy.getAllowanceTypes.and.returnValue(of(mockAllowanceTypes));
    apiServiceSpy.createRecord.and.returnValue(
      of({ id: 'r1', type: 'income', amount: 50, description: 'お皿洗い', date: '2026-03-27', created_at: '2026-03-27T10:00:00Z', allowance_type_id: 't1' })
    );

    routerSpy = jasmine.createSpyObj<Router>('Router', ['navigate']);

    await TestBed.configureTestingModule({
      imports: [ChoreRegisterComponent, NoopAnimationsModule],
      providers: [
        { provide: ApiService, useValue: apiServiceSpy },
        { provide: Router, useValue: routerSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ChoreRegisterComponent);
    component = fixture.componentInstance;

    // ngOnInit（detectChanges）より前にスパイを設定し、初期化中の呼び出しも検出できるようにする
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

    it('ngOnInit で子ども一覧とお手伝い種類を取得する', () => {
      expect(apiServiceSpy.getChildren).toHaveBeenCalledTimes(1);
      expect(apiServiceSpy.getAllowanceTypes).toHaveBeenCalledTimes(1);
    });

    it('初期ステップは select-child である', () => {
      expect(component.step()).toBe('select-child');
    });

    it('取得した子ども一覧がシグナルに反映される', () => {
      expect(component.children()).toEqual(mockChildren);
    });

    it('取得したお手伝い種類一覧がシグナルに反映される', () => {
      expect(component.allowanceTypes()).toEqual(mockAllowanceTypes);
    });
  });

  // ── ステップ1: 子ども選択 ──────────────────────────────────

  describe('selectChild', () => {
    it('子どもを選択するとステップが select-type に変わる', () => {
      component.selectChild(mockChildren[0]);
      expect(component.step()).toBe('select-type');
    });

    it('選択した子どもが selectedChild シグナルに保存される', () => {
      component.selectChild(mockChildren[1]);
      expect(component.selectedChild()).toEqual(mockChildren[1]);
    });
  });

  // ── ステップ2: お手伝い種類選択 ────────────────────────────

  describe('selectType', () => {
    beforeEach(() => {
      // ステップ2の前提: 子どもを選択済みにする
      component.selectChild(mockChildren[0]);
    });

    it('種類を選択すると createRecord が正しい引数で呼ばれる', fakeAsync(() => {
      const d = new Date();
      const today = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
      component.selectType(mockAllowanceTypes[0]);
      tick();

      expect(apiServiceSpy.createRecord).toHaveBeenCalledWith('c1', {
        type: 'income',
        amount: 50,
        description: 'お皿洗い',
        date: today,
        allowance_type_id: 't1',
      });
    }));

    it('登録成功後にステップが done に変わる', fakeAsync(() => {
      component.selectType(mockAllowanceTypes[0]);
      tick();
      expect(component.step()).toBe('done');
    }));

    it('登録成功後に selectedType シグナルが設定される', fakeAsync(() => {
      component.selectType(mockAllowanceTypes[0]);
      tick();
      expect(component.selectedType()).toEqual(mockAllowanceTypes[0]);
    }));

    it('selectedChild が null の場合は createRecord を呼ばない', () => {
      component.selectedChild.set(null);
      component.selectType(mockAllowanceTypes[0]);
      expect(apiServiceSpy.createRecord).not.toHaveBeenCalled();
    });

    it('API エラー時は SnackBar を表示してステップを変えない', () => {
      apiServiceSpy.createRecord.and.returnValue(throwError(() => new Error('500')));
      component.selectType(mockAllowanceTypes[0]);

      expect(snackBar.open).toHaveBeenCalledWith('登録に失敗しました', '閉じる', { duration: 3000 });
      expect(component.step()).toBe('select-type');
    });
  });

  // ── goBack ───────────────────────────────────────────────────

  describe('goBack', () => {
    it('ステップを select-child に戻して selectedChild をリセットする', () => {
      component.selectChild(mockChildren[0]);
      expect(component.step()).toBe('select-type');

      component.goBack();

      expect(component.step()).toBe('select-child');
      expect(component.selectedChild()).toBeNull();
    });
  });

  // ── reset ────────────────────────────────────────────────────

  describe('reset', () => {
    it('すべての状態をリセットしてステップ1に戻る', fakeAsync(() => {
      // 完了状態まで進める
      component.selectChild(mockChildren[0]);
      component.selectType(mockAllowanceTypes[0]);
      tick();
      expect(component.step()).toBe('done');

      component.reset();

      expect(component.step()).toBe('select-child');
      expect(component.selectedChild()).toBeNull();
      expect(component.selectedType()).toBeNull();
    }));
  });

  // ── goToDashboard ────────────────────────────────────────────

  describe('goToDashboard', () => {
    it('/dashboard へ遷移する', () => {
      component.goToDashboard();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/dashboard']);
    });
  });

  // ── エラーハンドリング（初期ロード） ─────────────────────────
  // 新しいコンポーネントを作成してエラーが snackBar.open を呼ぶことを確認する。
  // 同じ TestBed injector から生成されるため snackBar は同一シングルトンインスタンスであり、
  // beforeEach でインストールしたスパイがそのまま有効。

  describe('初期ロードのエラーハンドリング', () => {
    // forkJoin による並行取得のため、どちらがエラーになっても同一のエラーメッセージを表示する
    it('getChildren がエラーの場合は SnackBar を表示する', () => {
      apiServiceSpy.getChildren.and.returnValue(throwError(() => new Error('404')));

      const errorFixture = TestBed.createComponent(ChoreRegisterComponent);
      errorFixture.detectChanges();

      expect(snackBar.open).toHaveBeenCalledWith(
        'データの取得に失敗しました',
        '閉じる',
        { duration: 3000 }
      );

      errorFixture.destroy();
    });

    it('getAllowanceTypes がエラーの場合は SnackBar を表示する', () => {
      apiServiceSpy.getAllowanceTypes.and.returnValue(throwError(() => new Error('404')));

      const errorFixture = TestBed.createComponent(ChoreRegisterComponent);
      errorFixture.detectChanges();

      expect(snackBar.open).toHaveBeenCalledWith(
        'データの取得に失敗しました',
        '閉じる',
        { duration: 3000 }
      );

      errorFixture.destroy();
    });
  });

  // ── テンプレートの表示確認 ───────────────────────────────────

  describe('テンプレート表示', () => {
    it('ステップ1で子ども名がカードに表示される', () => {
      fixture.detectChanges();
      const compiled: HTMLElement = fixture.nativeElement;
      const cards = compiled.querySelectorAll('.child-card');
      expect(cards.length).toBe(mockChildren.length);
      expect(cards[0].textContent).toContain('たろう');
      expect(cards[1].textContent).toContain('はなこ');
    });

    it('ステップ2でお手伝い種類カードが表示される', () => {
      component.selectChild(mockChildren[0]);
      fixture.detectChanges();
      const compiled: HTMLElement = fixture.nativeElement;
      const cards = compiled.querySelectorAll('.type-card');
      expect(cards.length).toBe(mockAllowanceTypes.length);
      expect(cards[0].textContent).toContain('お皿洗い');
    });

    it('完了画面で子ども名・種類名・金額が表示される', fakeAsync(() => {
      component.selectChild(mockChildren[0]);
      component.selectType(mockAllowanceTypes[0]);
      tick();
      fixture.detectChanges();

      const compiled: HTMLElement = fixture.nativeElement;
      expect(compiled.textContent).toContain('登録できたよ');
      expect(compiled.textContent).toContain('たろう');
      expect(compiled.textContent).toContain('お皿洗い');
      expect(compiled.textContent).toContain('50');
    }));
  });
});
