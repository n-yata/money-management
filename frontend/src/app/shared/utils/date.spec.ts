import { formatDate } from './date';

describe('formatDate', () => {
  describe('ゼロパディング', () => {
    it('1桁の月・日がゼロパディングされること', () => {
      const date = new Date(2026, 0, 5); // 2026-01-05
      expect(formatDate(date)).toBe('2026-01-05');
    });

    it('1桁の月のみゼロパディングされること', () => {
      const date = new Date(2026, 2, 15); // 2026-03-15
      expect(formatDate(date)).toBe('2026-03-15');
    });

    it('1桁の日のみゼロパディングされること', () => {
      const date = new Date(2026, 9, 5); // 2026-10-05
      expect(formatDate(date)).toBe('2026-10-05');
    });
  });

  describe('2桁の月・日', () => {
    it('2桁の月・日はそのまま出力されること', () => {
      const date = new Date(2026, 11, 31); // 2026-12-31
      expect(formatDate(date)).toBe('2026-12-31');
    });
  });

  describe('境界値', () => {
    it('12月31日が正しくフォーマットされること', () => {
      const date = new Date(2026, 11, 31); // 2026-12-31
      expect(formatDate(date)).toBe('2026-12-31');
    });

    it('1月1日が正しくフォーマットされること', () => {
      const date = new Date(2026, 0, 1); // 2026-01-01
      expect(formatDate(date)).toBe('2026-01-01');
    });
  });
});
