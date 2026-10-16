import type { KeyboardEvent, MouseEvent } from 'react';
import type { Expense } from '../api/types';
import { memberLabel, type NameMap } from '../lib/members';
import { formatRubles } from '../lib/money';
import { CategoryIcon } from './CategoryIcon';
import { Icon } from './Icon';
import { Person } from './Person';

interface ExpenseRowProps {
  expense: Expense;
  names: NameMap;
  highlighted: boolean;
  onEdit: (e: Expense, opener: HTMLElement) => void;
  onDelete: (e: Expense, opener: HTMLElement) => void;
}

export function ExpenseRow({ expense: e, names, highlighted, onEdit, onDelete }: ExpenseRowProps) {
  const amount = formatRubles(e.amount);
  const what = e.comment || e.category_label;
  const summary = `${e.category_label}, ${amount}, ${memberLabel(e.user_id, names)}, ${e.comment || 'без комментария'}`;

  const onKeyDown = (ev: KeyboardEvent<HTMLLIElement>) => {
    if (ev.target !== ev.currentTarget || (ev.key !== 'Enter' && ev.key !== ' ')) return;
    ev.preventDefault();
    onEdit(e, ev.currentTarget);
  };
  const act = (handler: typeof onEdit) => (ev: MouseEvent<HTMLButtonElement>) => {
    ev.stopPropagation();
    handler(e, ev.currentTarget);
  };

  return (
    <li
      className={highlighted ? 'expense highlight' : 'expense'}
      data-id={e.id}
      tabIndex={0}
      aria-label={summary}
      onClick={(ev) => onEdit(e, ev.currentTarget)}
      onKeyDown={onKeyDown}
    >
      <Person id={e.user_id} names={names} />
      <span className="category">
        <CategoryIcon category={e.category} />
        <span className="category-label">{e.category_label}</span>
      </span>
      <span className={e.comment ? 'comment' : 'comment muted'} title={e.comment || undefined}>
        {e.comment || 'Без комментария'}
      </span>
      <span className="amount num">{amount}</span>
      <span className="row-actions">
        <button type="button" className="edit-button" aria-label={`Изменить: ${what}, ${amount}`} onClick={act(onEdit)}>
          <Icon name="edit" />
        </button>
        <button
          type="button"
          className="edit-button delete-button"
          aria-label={`Удалить: ${what}, ${amount}`}
          onClick={act(onDelete)}
        >
          <Icon name="trash" />
        </button>
      </span>
    </li>
  );
}
