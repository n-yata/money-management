import { TestBed } from '@angular/core/testing';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { createChildForm } from './child-form.factory';

describe('createChildForm', () => {
  let fb: FormBuilder;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ReactiveFormsModule],
    });
    fb = TestBed.inject(FormBuilder);
  });

  describe('初期値なし', () => {
    it('初期値なしで呼んだ場合、フォームが invalid であること', () => {
      const form = createChildForm(fb);
      expect(form.invalid).toBeTrue();
    });

    it('name の初期値が空文字であること', () => {
      const form = createChildForm(fb);
      expect(form.get('name')!.value).toBe('');
    });

    it('age の初期値が null であること', () => {
      const form = createChildForm(fb);
      expect(form.get('age')!.value).toBeNull();
    });

    it('base_allowance の初期値が null であること', () => {
      const form = createChildForm(fb);
      expect(form.get('base_allowance')!.value).toBeNull();
    });
  });

  describe('初期値あり', () => {
    it('全フィールドに値をセットした場合、フォームに値が反映されること', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 8, base_allowance: 1000 });

      expect(form.get('name')!.value).toBe('たろう');
      expect(form.get('age')!.value).toBe(8);
      expect(form.get('base_allowance')!.value).toBe(1000);
    });

    it('全フィールドに有効な値がセットされた場合、フォームが valid であること', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 8, base_allowance: 1000 });

      expect(form.valid).toBeTrue();
    });
  });

  describe('name バリデーション', () => {
    it('20文字の name は valid であること', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'あ'.repeat(20), age: 8, base_allowance: 1000 });

      expect(form.get('name')!.valid).toBeTrue();
    });

    it('21文字の name は invalid であること（maxLength 超過）', () => {
      const form = createChildForm(fb);
      form.get('name')!.setValue('あ'.repeat(21));

      expect(form.get('name')!.hasError('maxlength')).toBeTrue();
    });

    it('name が空のとき invalid であること（required）', () => {
      const form = createChildForm(fb);
      form.get('name')!.setValue('');

      expect(form.get('name')!.hasError('required')).toBeTrue();
    });
  });

  describe('age バリデーション', () => {
    it('age が 1 のとき valid であること（min 境界値）', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 1, base_allowance: 1000 });

      expect(form.get('age')!.valid).toBeTrue();
    });

    it('age が 18 のとき valid であること（max 境界値）', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 18, base_allowance: 1000 });

      expect(form.get('age')!.valid).toBeTrue();
    });

    it('age が 0 のとき invalid であること（min 未満）', () => {
      const form = createChildForm(fb);
      form.get('age')!.setValue(0);

      expect(form.get('age')!.hasError('min')).toBeTrue();
    });

    it('age が 19 のとき invalid であること（max 超過）', () => {
      const form = createChildForm(fb);
      form.get('age')!.setValue(19);

      expect(form.get('age')!.hasError('max')).toBeTrue();
    });

    it('age が null のとき invalid であること（required）', () => {
      const form = createChildForm(fb);
      form.get('age')!.setValue(null);

      expect(form.get('age')!.hasError('required')).toBeTrue();
    });
  });

  describe('base_allowance バリデーション', () => {
    it('base_allowance が 0 のとき valid であること（min 境界値）', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 8, base_allowance: 0 });

      expect(form.get('base_allowance')!.valid).toBeTrue();
    });

    it('base_allowance が正の値のとき valid であること', () => {
      const form = createChildForm(fb);
      form.patchValue({ name: 'たろう', age: 8, base_allowance: 500 });

      expect(form.get('base_allowance')!.valid).toBeTrue();
    });

    it('base_allowance が -1 のとき invalid であること（min 未満）', () => {
      const form = createChildForm(fb);
      form.get('base_allowance')!.setValue(-1);

      expect(form.get('base_allowance')!.hasError('min')).toBeTrue();
    });

    it('base_allowance が null のとき invalid であること（required）', () => {
      const form = createChildForm(fb);
      form.get('base_allowance')!.setValue(null);

      expect(form.get('base_allowance')!.hasError('required')).toBeTrue();
    });
  });
});
