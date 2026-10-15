const NBSP = ' ';

/**
 * Принимает: строку, введённую пользователем ("350", "350,5", "350.50").
 * Делает: проверяет формат (до двух знаков после разделителя) и переводит рубли в копейки без float-ошибок.
 * Возвращает: целое число копеек больше нуля или null, если ввод некорректен.
 */
export function parseRubles(input: string): number | null {
  const m = /^(\d{1,12})(?:[.,](\d{1,2}))?$/.exec(input.trim().replace(/\s+/g, ''));
  if (!m) return null;
  const kopecks = Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0'));
  return kopecks > 0 ? kopecks : null;
}

/**
 * Принимает: сумму в копейках.
 * Делает: форматирует её как рубли с разделителем тысяч и знаком валюты.
 * Возвращает: строку вида "350,00 ₽" (с неразрывными пробелами).
 */
export function formatRubles(kopecks: number): string {
  const sign = kopecks < 0 ? '-' : '';
  const abs = Math.abs(kopecks);
  const rub = Math.floor(abs / 100)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, NBSP);
  const kop = (abs % 100).toString().padStart(2, '0');
  return `${sign}${rub},${kop}${NBSP}₽`;
}

/**
 * Принимает: сумму в копейках.
 * Делает: превращает её в строку для поля ввода.
 * Возвращает: строку вида "350,50" без знака валюты.
 */
export function kopecksToInput(kopecks: number): string {
  return `${Math.floor(kopecks / 100)},${(kopecks % 100).toString().padStart(2, '0')}`;
}
