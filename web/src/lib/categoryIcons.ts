/** Контуры иконок 24×24 (stroke, currentColor), как в прототипе «Журнал». */
const PATHS: Record<string, string> = {
  food: 'M5 3v7 M8 3v7 M3 3v5a3 3 0 0 0 6 0 M6 11v10 M17 3c-4 4-4 10 0 10h2 M19 3v18',
  transport:
    'M5 16H3V8l3-4h12l3 4v8h-2 M7 16h10 M3 10h18 M8 17a2 2 0 1 0-4 0a2 2 0 1 0 4 0 M20 17a2 2 0 1 0-4 0a2 2 0 1 0 4 0',
  housing: 'M3 11l9-8 9 8 M5 10v11h14V10 M10 21v-7h4v7',
  entertainment: 'M8 4l12 8-12 8z',
  health: 'M9 3h6v6h6v6h-6v6H9v-6H3V9h6z',
  shopping: 'M5 7h14l2 14H3z M8 8V6a4 4 0 0 1 8 0v2',
  other: 'M5 5h5v5H5z M14 5h5v5h-5z M5 14h5v5H5z M14 14h5v5h-5z',
};

/** Нейтральная «метка» для категории, которой ещё нет в таблице. */
export const FALLBACK_ICON_PATH = 'M3 12V4h8l10 10-8 8z M8 8.5v.01';

/**
 * Принимает: ключ категории из API.
 * Делает: ищет иконку в таблице; новая категория не ломает интерфейс.
 * Возвращает: SVG-путь иконки или запасной путь.
 */
export function categoryIconPath(key: string): string {
  return Object.prototype.hasOwnProperty.call(PATHS, key) ? PATHS[key] : FALLBACK_ICON_PATH;
}
