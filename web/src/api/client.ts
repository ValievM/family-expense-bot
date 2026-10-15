import type {
  Category,
  CreateExpense,
  Expense,
  ListParams,
  Member,
  UpdateExpense,
} from './types';

/** Ошибка API с HTTP-статусом и кодом из тела {"error":{...}}. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  /**
   * Принимает: HTTP-статус, машинный код ошибки и человекочитаемое сообщение.
   * Делает: сохраняет статус и код в полях экземпляра.
   * Возвращает: ничего (конструктор).
   */
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

/**
 * Принимает: HTTP-метод, путь и необязательное тело запроса.
 * Делает: выполняет единственный fetch проекта, разбирает JSON и превращает сбои в ApiError.
 * Возвращает: Promise с разобранным ответом (undefined для 204).
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

/**
 * Принимает: параметры фильтра со строковыми значениями.
 * Делает: собирает query-строку, пропуская пустые значения.
 * Возвращает: строку вида "?a=b" или пустую строку.
 */
function toQuery(params: ListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== '') q.set(k, v);
  }
  const s = q.toString();
  return s === '' ? '' : `?${s}`;
}

/**
 * Принимает: ничего.
 * Делает: запрашивает список категорий.
 * Возвращает: Promise с массивом категорий в порядке отображения.
 */
export async function fetchCategories(): Promise<Category[]> {
  return (await request<{ items: Category[] }>('GET', '/api/categories')).items;
}

/**
 * Принимает: ничего.
 * Делает: запрашивает участников семьи.
 * Возвращает: Promise с массивом участников по возрастанию ID.
 */
export async function fetchMembers(): Promise<Member[]> {
  return (await request<{ items: Member[] }>('GET', '/api/users')).items;
}

/**
 * Принимает: параметры фильтра.
 * Делает: запрашивает расходы, подходящие под фильтр.
 * Возвращает: Promise с массивом расходов, новые первыми.
 */
export async function fetchExpenses(params: ListParams): Promise<Expense[]> {
  return (await request<{ items: Expense[] }>('GET', `/api/expenses${toQuery(params)}`)).items;
}

/**
 * Принимает: данные нового расхода (сумма в копейках).
 * Делает: отправляет POST /api/expenses.
 * Возвращает: Promise с созданным расходом.
 */
export function createExpense(input: CreateExpense): Promise<Expense> {
  return request<Expense>('POST', '/api/expenses', input);
}

/**
 * Принимает: id расхода и объект с изменёнными полями.
 * Делает: отправляет PATCH /api/expenses/{id}.
 * Возвращает: Promise с обновлённым расходом.
 */
export function updateExpense(id: number, patch: UpdateExpense): Promise<Expense> {
  return request<Expense>('PATCH', `/api/expenses/${id}`, patch);
}

/**
 * Принимает: id расхода.
 * Делает: отправляет DELETE /api/expenses/{id}.
 * Возвращает: Promise без значения.
 */
export function deleteExpense(id: number): Promise<void> {
  return request<void>('DELETE', `/api/expenses/${id}`);
}
