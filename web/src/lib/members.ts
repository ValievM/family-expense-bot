import type { Member } from '../api/types';
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
 * Принимает: Telegram ID и отображаемое имя (может отсутствовать).
 * Делает: берёт первые буквы двух слов имени (без «@»), без букв — две последние цифры ID.
 * Возвращает: одну-две заглавные буквы или цифры для аватара.
 */
export function monogram(id: number, name?: string): string {
  const letters = (name ?? '')
    .replace(/^@/, '')
    .split(/[\s._-]+/)
    .map((w) => Array.from(w).find((ch) => /\p{L}/u.test(ch)))
    .filter((ch): ch is string => ch !== undefined);
  if (letters.length === 0) return String(id).slice(-2);
  return letters
    .slice(0, 2)
    .map((ch) => ch.toUpperCase())
    .join('');
}

/** Член семьи в том виде, в каком его показывает интерфейс. */
export interface MemberView {
  /** Подпись в списках; при совпадении с другим человеком дополнена коротким ID. */
  label: string;
  /** Имя без уточнения ID; пустое, если известен только ID. Из него строится монограмма. */
  name: string;
  /** Вторичный текст: «Telegram ID 6612214276 · @valiev». */
  details: string;
}

export type MemberDirectory = ReadonlyMap<number, MemberView>;

/**
 * Принимает: неизвестный элемент ответа GET /api/users.
 * Делает: проверяет user_id, недостающие или нестроковые name/username заменяет пустой строкой
 *   (старый сервер присылал только user_id), убирает «@» в начале username.
 * Возвращает: Member или null, если user_id некорректен.
 */
export function normalizeMember(raw: unknown): Member | null {
  if (!raw || typeof raw !== 'object') return null;
  const r = raw as Record<string, unknown>;
  if (typeof r.user_id !== 'number' || !Number.isSafeInteger(r.user_id) || r.user_id <= 0) return null;
  const str = (v: unknown) => (typeof v === 'string' ? v.trim() : '');
  return { user_id: r.user_id, name: str(r.name), username: str(r.username).replace(/^@+/, '') };
}

export type LabelSource = 'local' | 'telegram' | 'username' | 'id';

/**
 * Принимает: Telegram ID, локальное имя и данные с сервера (оба могут отсутствовать).
 * Делает: выбирает подпись по приоритету: имя на этом устройстве → имя из Telegram → @username → «…4276».
 * Возвращает: подпись и её источник.
 */
export function pickLabel(
  id: number,
  localName: string | undefined,
  member: Pick<Member, 'name' | 'username'> | undefined,
): { label: string; source: LabelSource } {
  const local = (localName ?? '').trim();
  if (local !== '') return { label: local, source: 'local' };
  if (member?.name) return { label: member.name, source: 'telegram' };
  if (member?.username) return { label: `@${member.username}`, source: 'username' };
  return { label: shortId(id), source: 'id' };
}

/**
 * Принимает: Telegram ID и данные с сервера.
 * Делает: собирает полный ID и @username, чтобы различать людей с похожими именами.
 * Возвращает: «Telegram ID 6612214276 · @valiev».
 */
export function memberDetails(id: number, member?: Pick<Member, 'username'>): string {
  return member?.username ? `Telegram ID ${id} · @${member.username}` : `Telegram ID ${id}`;
}

/**
 * Принимает: ID всех людей на экране, список с сервера и локальные имена.
 * Делает: строит подписи через pickLabel; одинаковые подписи уточняет
 *   коротким ID, а если совпадают и последние цифры — полным.
 * Возвращает: справочник ID → MemberView.
 */
export function buildDirectory(ids: number[], members: Member[], names: NameMap): MemberDirectory {
  const byId = new Map(members.map((m) => [m.user_id, m]));
  const all = [...new Set([...members.map((m) => m.user_id), ...ids])];
  const base = all.map((id) => {
    const member = byId.get(id);
    const { label, source } = pickLabel(id, names[String(id)], member);
    return { id, name: source === 'id' ? '' : label, details: memberDetails(id, member) };
  });
  const plain = (b: (typeof base)[number]) => b.name || shortId(b.id);
  const withShort = (b: (typeof base)[number]) => (b.name ? `${b.name} · ${shortId(b.id)}` : String(b.id));
  const withFull = (b: (typeof base)[number]) => (b.name ? `${b.name} · ${b.id}` : String(b.id));

  const plainDup = duplicates(base.map(plain));
  const shortDup = duplicates(base.map((b) => (plainDup.has(norm(plain(b))) ? withShort(b) : plain(b))));
  const out = new Map<number, MemberView>();
  for (const b of base) {
    let label = plain(b);
    if (plainDup.has(norm(label))) label = shortDup.has(norm(withShort(b))) ? withFull(b) : withShort(b);
    out.set(b.id, { label, name: b.name, details: b.details });
  }
  return out;
}

function norm(label: string): string {
  return label.trim().toLocaleLowerCase('ru');
}

function duplicates(labels: string[]): Set<string> {
  const seen = new Set<string>();
  const dup = new Set<string>();
  for (const l of labels.map(norm)) {
    if (seen.has(l)) dup.add(l);
    seen.add(l);
  }
  return dup;
}

export function memberView(dir: MemberDirectory, id: number): MemberView {
  return dir.get(id) ?? { label: shortId(id), name: '', details: memberDetails(id) };
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
