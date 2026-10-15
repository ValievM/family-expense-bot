import type { Expense } from '../api/types';
import { formatDate } from '../lib/date';
import { formatRubles } from '../lib/money';
import { totalKopecks } from '../lib/total';

/** Свойства таблицы расходов. */
interface ExpenseTableProps {
  items: Expense[];
  busy: boolean;
  onEdit: (e: Expense) => void;
  onDelete: (e: Expense) => void;
}

/**
 * Принимает: список расходов, флаг занятости и обработчики правки и удаления.
 * Делает: рисует таблицу (на узких экранах карточки) и строку итога по показанным строкам.
 * Возвращает: React-элемент таблицы.
 */
export function ExpenseTable({ items, busy, onEdit, onDelete }: ExpenseTableProps) {
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th scope="col">Дата</th>
            <th scope="col">Участник</th>
            <th scope="col">Категория</th>
            <th scope="col">Комментарий</th>
            <th scope="col" className="num">Сумма</th>
            <th scope="col"><span className="visually-hidden">Действия</span></th>
          </tr>
        </thead>
        <tbody>
          {items.map((e) => (
            <tr key={e.id}>
              <td data-label="Дата">{formatDate(e.spent_at)}</td>
              <td data-label="Участник">{e.user_id}</td>
              <td data-label="Категория"><span className="chip">{e.category_label}</span></td>
              <td data-label="Комментарий" className="comment">{e.comment || '—'}</td>
              <td data-label="Сумма" className="num">{formatRubles(e.amount)}</td>
              <td className="actions">
                <button type="button" className="btn btn--small" disabled={busy} onClick={() => onEdit(e)} aria-label={`Изменить расход ${e.id}`}>
                  Изменить
                </button>
                <button type="button" className="btn btn--small btn--danger" disabled={busy} onClick={() => onDelete(e)} aria-label={`Удалить расход ${e.id}`}>
                  Удалить
                </button>
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <th scope="row" colSpan={4}>Итого ({items.length})</th>
            <td className="num total">{formatRubles(totalKopecks(items))}</td>
            <td />
          </tr>
        </tfoot>
      </table>
    </div>
  );
}
