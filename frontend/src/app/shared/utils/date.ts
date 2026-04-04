/**
 * Date ユーティリティ関数。
 * 日付の書式変換など、複数コンポーネントで共通して使う関数を集約する。
 */

/**
 * Date オブジェクトを YYYY-MM-DD 形式の文字列に変換する。
 *
 * @param date 変換対象の Date オブジェクト
 * @returns "YYYY-MM-DD" 形式の文字列
 */
export function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}
