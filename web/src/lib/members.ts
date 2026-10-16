import { safeGet, safeSet, type StorageLike } from './storage';

/** Локальные имена: Telegram ID строкой → имя. Живут только в этом браузере. */
export type NameMap = Record<string, string>;

export const NAMES_KEY = 'family-member-names';
export const MAX_NAME_LENGTH = 40;

/**
 * Принимает: Telegram ID.
 * Делает: оставляет последние четыре цифры, чтобы длинные ID не занимали строку.
 * Возвращает: «…4276»; короткий ID целиком.
 */
export function shortId(id: number): string {
  const s = String(id);
  return s.length > 4 ? `…${s.slice(-4)}` : s;
}

/**
 * Принимает: Telegram ID.
 * Делает: хеширует цифры (FNV-1a), чтобы цвет человека был одинаковым везде и всегда.
 * Возвращает: тон цвета 0–359.
 */
export function memberHue(id: number): number {
  let h = 0x811c9dc5;
  for (const ch of String(id)) {
    h ^= ch.charCodeAt(0);
    h = Math.imul(h, 0x01000193);
  }
  return (h >>> 0) % 360;
}

/**
 * Принимает: Telegram ID и локальное имя (может отсутствовать).
 * Делает: берёт первые буквы двух слов имени, без имени — две последние цифры ID.
 * Возвращает: одну-две буквы или цифры для аватара.
 */
export function monogram(id: number, name?: string): string {
  const words = (name ?? '').trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return String(id).slice(-2);
  return words
    .slice(0, 2)
    .map((w) => Array.from(w)[0].toUpperCase())
    .join('');
}

export function memberLabel(id: number, names: NameMap): string {
  return names[String(id)] ?? shortId(id);
}

export function memberOptionLabel(id: number, names: NameMap): string {
  const name = names[String(id)];
  return name ? `${name} · ${shortId(id)}` : `ID ${shortId(id)}`;
}

/**
 * Принимает: хранилище (или null).
 * Делает: читает имена и отбрасывает всё, что не похоже на {ID: строка}.
 * Возвращает: словарь имён; пустой, если хранилище недоступно или испорчено.
 */
export function loadNames(storage: StorageLike | null): NameMap {
  const raw = safeGet(storage, NAMES_KEY);
  if (!raw) return {};
  try {
    const data: unknown = JSON.parse(raw);
    if (!data || typeof data !== 'object' || Array.isArray(data)) return {};
    const out: NameMap = {};
    for (const [k, v] of Object.entries(data)) {
      if (/^\d+$/.test(k) && typeof v === 'string' && v.trim() !== '') out[k] = v.trim();
    }
    return out;
  } catch {
    return {};
  }
}

/**
 * Принимает: текущие имена, Telegram ID и новое имя.
 * Делает: обрезает пробелы и длину; пустое имя удаляет запись.
 * Возвращает: новый словарь (старый не меняется).
 */
export function withName(names: NameMap, id: number, name: string): NameMap {
  const next = { ...names };
  const clean = Array.from(name.trim()).slice(0, MAX_NAME_LENGTH).join('');
  if (clean === '') delete next[String(id)];
  else next[String(id)] = clean;
  return next;
}

export function saveNames(storage: StorageLike | null, names: NameMap): boolean {
  return safeSet(storage, NAMES_KEY, JSON.stringify(names));
}
