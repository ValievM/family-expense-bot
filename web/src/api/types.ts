export type CategoryKey =
  | 'food'
  | 'transport'
  | 'housing'
  | 'entertainment'
  | 'health'
  | 'shopping'
  | 'other';

/** amount в копейках. */
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

export interface Category {
  key: CategoryKey;
  label: string;
}

/** name и username приходят из Telegram; пустая строка — неизвестно. */
export interface Member {
  user_id: number;
  name: string;
  username: string;
}

export interface CreateExpense {
  user_id: number;
  amount: number;
  category: CategoryKey;
  spent_at?: string;
  comment?: string;
}

export interface UpdateExpense {
  amount?: number;
  category?: CategoryKey;
  spent_at?: string;
  comment?: string;
}

/** Пустая строка означает "без фильтра". */
export interface ListParams {
  user_id: string;
  category: string;
  from: string;
  to: string;
}
