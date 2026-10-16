import type {
  Category,
  CreateExpense,
  Expense,
  ListParams,
  Member,
  UpdateExpense,
} from './types';
import { normalizeMember } from '../lib/members';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

/**
 * Принимает: метод, путь, необязательное тело.
 * Делает: единственный fetch проекта; сбои превращает в ApiError.
 * Возвращает: разобранный ответ (undefined для 204).
 */
async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'network', 'Нет связи с сервером. Проверьте соединение.');
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const err = (data as { error?: { code?: string; message?: string } } | null)?.error;
    throw new ApiError(
      res.status,
      err?.code ?? 'unknown',
      err?.message ?? `Ошибка сервера (${res.status})`,
    );
  }
  return data as T;
}

function toQuery(params: ListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== '') q.set(k, v);
  }
  const s = q.toString();
  return s === '' ? '' : `?${s}`;
}

export async function fetchCategories(): Promise<Category[]> {
  return (await request<{ items: Category[] }>('GET', '/api/categories')).items;
}

export async function fetchMembers(): Promise<Member[]> {
  const items = (await request<{ items?: unknown }>('GET', '/api/users')).items;
  return Array.isArray(items) ? items.map(normalizeMember).filter((m): m is Member => m !== null) : [];
}

export async function fetchExpenses(params: ListParams): Promise<Expense[]> {
  return (await request<{ items: Expense[] }>('GET', `/api/expenses${toQuery(params)}`)).items;
}

export function createExpense(input: CreateExpense): Promise<Expense> {
  return request<Expense>('POST', '/api/expenses', input);
}

export function updateExpense(id: number, patch: UpdateExpense): Promise<Expense> {
  return request<Expense>('PATCH', `/api/expenses/${id}`, patch);
}

export function deleteExpense(id: number): Promise<void> {
  return request<void>('DELETE', `/api/expenses/${id}`);
}
