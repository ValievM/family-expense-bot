import type { Expense } from '../api/types';
import type { DayGroup as Group } from '../lib/groupByDay';
import { dayLabel } from '../lib/groupByDay';
import type { NameMap } from '../lib/members';
import { formatRubles } from '../lib/money';
import { ExpenseRow } from './ExpenseRow';

interface DayGroupProps {
  group: Group;
  today: string;
  names: NameMap;
  highlightId: number | null;
  onEdit: (e: Expense, opener: HTMLElement) => void;
  onDelete: (e: Expense, opener: HTMLElement) => void;
}

export function DayGroup({ group, today, names, highlightId, onEdit, onDelete }: DayGroupProps) {
  const label = dayLabel(group.date, today);
  const headingId = `day-${group.date}`;
  return (
    <li className="day">
      <section aria-labelledby={headingId}>
        <div className="day-head">
          <h3 id={headingId}>
            <b>{label.day}</b>{' '}
            <span>
              {label.month}
              {label.year && ` ${label.year}`}
            </span>
            {label.relative && (
              <>
                {' '}
                <span className="day-relative">{label.relative}</span>
              </>
            )}
          </h3>
          <span className="num day-total">
            <span className="visually-hidden">За день: </span>
            {formatRubles(group.total)}
          </span>
        </div>
        <ul className="day-items">
          {group.items.map((e) => (
            <ExpenseRow
              key={e.id}
              expense={e}
              names={names}
              highlighted={e.id === highlightId}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </ul>
      </section>
    </li>
  );
}
