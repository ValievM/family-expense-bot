import { formatRubles } from '../lib/money';
import { plural } from '../lib/plural';

interface SummaryProps {
  total: number | null;
  count: number;
  people: number;
  members: number;
}

export function Summary({ total, count, people, members }: SummaryProps) {
  return (
    <div className="summary">
      <div>
        <div className="summary-label">Всего по выборке</div>
        {total === null ? (
          <div className="summary-total skeleton-bar skeleton-total" aria-hidden="true" />
        ) : (
          <div className="summary-total num">{formatRubles(total)}</div>
        )}
      </div>
      {total !== null && (
        <div className="summary-count">
          <div>{plural(count, ['трата', 'траты', 'трат'])}</div>
          <div>
            {people} из {plural(members, ['члена семьи', 'членов семьи', 'членов семьи'])}
          </div>
        </div>
      )}
    </div>
  );
}
