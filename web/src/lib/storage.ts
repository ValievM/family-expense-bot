/** Узкое подмножество Web Storage: так в тестах легко подставить фейк. */
export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

/**
 * Принимает: ничего.
 * Делает: достаёт localStorage; доступ сам по себе может бросить (приватный режим, запрет cookies).
 * Возвращает: хранилище или null, если оно недоступно.
 */
export function browserStorage(): StorageLike | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null;
  }
}

/**
 * Принимает: хранилище (или null) и ключ.
 * Делает: читает значение, проглатывая ошибки хранилища.
 * Возвращает: строку или null.
 */
export function safeGet(storage: StorageLike | null, key: string): string | null {
  try {
    return storage?.getItem(key) ?? null;
  } catch {
    return null;
  }
}

/**
 * Принимает: хранилище (или null), ключ и значение.
 * Делает: пишет значение, проглатывая ошибки (переполнение, запрет записи).
 * Возвращает: true, если запись удалась.
 */
export function safeSet(storage: StorageLike | null, key: string, value: string): boolean {
  if (!storage) return false;
  try {
    storage.setItem(key, value);
    return true;
  } catch {
    return false;
  }
}
