/** Ключ категории расхода из закрытого списка API. */
export type CategoryKey =
  | 'food'
  | 'transport'
  | 'housing'
  | 'entertainment'
  | 'health'
  | 'shopping'
  | 'other';

/** Расход как его отдаёт API; amount в копейках. */
export interface Expense {
  id: number;
  user_id: number;
  amount: number;
  category: CategoryKey;
  category_label: string;
  spent_at: string;
  comment: string;
  created_at: string;
  updated_at: string;
}

/** Категория со своей русской подписью. */
export interface Category {
  key: CategoryKey;
  label: string;
}

/** Участник семьи, у которого есть хотя бы один расход. */
export interface Member {
  user_id: number;
}

/** Тело POST /api/expenses; amount в копейках. */
export interface CreateExpense {
  user_id: number;
  amount: number;
  category: CategoryKey;
  spent_at?: string;
  comment?: string;
}

/** Тело PATCH /api/expenses/{id}: только изменённые поля. */
export interface UpdateExpense {
  amount?: number;
  category?: CategoryKey;
  spent_at?: string;
  comment?: string;
}

/** Параметры запроса списка расходов; пустая строка означает "без фильтра". */
export interface ListParams {
  user_id: string;
  category: string;
  from: string;
  to: string;
}
