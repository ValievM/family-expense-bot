const NBSP = ' ';

/**
 * Принимает: ввод пользователя ("350", "350,5", "350.50").
 * Делает: проверяет формат (до 2 знаков), рубли в копейки без float.
 * Возвращает: копейки > 0 или null при некорректном вводе.
 */
export function parseRubles(input: string): number | null {
  const m = /^(\d{1,12})(?:[.,](\d{1,2}))?$/.exec(input.trim().replace(/\s+/g, ''));
  if (!m) return null;
  const kopecks = Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0'));
  return kopecks > 0 ? kopecks : null;
}

/**
 * Принимает: копейки.
 * Делает: форматирует как рубли с разделителем тысяч.
 * Возвращает: "350,00 ₽" (с неразрывными пробелами).
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

export function kopecksToInput(kopecks: number): string {
  return `${Math.floor(kopecks / 100)},${(kopecks % 100).toString().padStart(2, '0')}`;
}
