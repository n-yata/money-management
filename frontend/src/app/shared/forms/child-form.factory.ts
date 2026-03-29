import { FormBuilder, Validators } from '@angular/forms';

/**
 * 子ども情報フォームを生成するファクトリ関数。
 * ChildNewComponent と ChildEditComponent で共通利用する。
 */
export function createChildForm(fb: FormBuilder) {
  return fb.group({
    name: ['', [Validators.required, Validators.maxLength(20)]],
    age: [null as number | null, [Validators.required, Validators.min(1), Validators.max(18)]],
    base_allowance: [null as number | null, [Validators.required, Validators.min(0)]],
  });
}
