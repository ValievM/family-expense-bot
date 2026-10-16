import { useRef } from 'react';
import type { ApiError } from '../api/client';
import type { Expense } from '../api/types';
import { bannerMessage, hasTechnicalDetails } from '../lib/apiErrors';
import { formatDate } from '../lib/date';
import type { NameMap } from '../lib/members';
import { formatRubles } from '../lib/money';
import { CategoryIcon } from './CategoryIcon';
import { Dialog } from './Dialog';
import { ErrorBanner } from './ErrorBanner';
import { Person } from './Person';

interface ConfirmDialogProps {
  expense: Expense;
  names: NameMap;
  deleting: boolean;
  error: ApiError | null;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({ expense: e, names, deleting, error, onConfirm, onCancel }: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);
  return (
    <Dialog
      className="confirm"
      labelledBy="confirm-title"
      initialFocus={cancelRef}
      onRequestClose={() => {
        if (!deleting) onCancel();
      }}
    >
      <div className="dialog-header">
        <div>
          <div className="eyebrow">Удаление записи</div>
          <h2 id="confirm-title">Удалить трату?</h2>
          <p id="confirm-desc">Запись удалится без возможности восстановления.</p>
        </div>
      </div>
      <div className="form-body">
        <div className="delete-amount num">{formatRubles(e.amount)}</div>
        <dl className="delete-facts">
          <div>
            <dt>Категория</dt>
            <dd className="category">
              <CategoryIcon category={e.category} />
              {e.category_label}
            </dd>
          </div>
          <div>
            <dt>Дата</dt>
            <dd>{formatDate(e.spent_at)}</dd>
          </div>
          <div>
            <dt>Член семьи</dt>
            <dd>
              <Person id={e.user_id} names={names} />
            </dd>
          </div>
          <div>
            <dt>Комментарий</dt>
            <dd className={e.comment ? 'delete-comment' : 'delete-comment muted'}>{e.comment || 'Без комментария'}</dd>
          </div>
        </dl>
        {error && (
          <ErrorBanner
            message={bannerMessage(error.code)}
            details={hasTechnicalDetails(error.code) ? `${error.code}: ${error.message}` : undefined}
          />
        )}
      </div>
      <div className="dialog-footer">
        <div className="footer-actions">
          <button ref={cancelRef} type="button" onClick={onCancel} disabled={deleting}>
            Отмена
          </button>
          <button type="button" className="danger-solid" onClick={onConfirm} disabled={deleting} aria-busy={deleting}>
            {deleting ? 'Удаляем…' : error ? 'Повторить удаление' : 'Удалить трату'}
          </button>
        </div>
      </div>
    </Dialog>
  );
}
