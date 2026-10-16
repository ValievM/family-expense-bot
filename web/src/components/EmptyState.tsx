import { Icon } from './Icon';

interface EmptyStateProps {
  kind: 'first' | 'filtered';
  canAdd: boolean;
  onAdd: (opener: HTMLElement) => void;
  onReset: () => void;
}

export function EmptyState({ kind, canAdd, onAdd, onReset }: EmptyStateProps) {
  if (kind === 'filtered') {
    return (
      <div className="empty">
        <h2>Ничего не найдено</h2>
        <p>Попробуйте другой период или сбросьте фильтры.</p>
        <button type="button" onClick={onReset}>
          Сбросить фильтры
        </button>
      </div>
    );
  }
  return (
    <div className="empty empty-first">
      <span className="empty-mark" aria-hidden="true">
        <Icon name="message" />
      </span>
      <h2>Здесь появятся траты семьи</h2>
      <p>
        Траты попадают сюда, когда кто-то из семьи пишет боту в Telegram сумму и на что потрачено.
        Например:
      </p>
      <p className="bot-example" aria-label="Пример сообщения боту: кофе 350">
        <span>кофе 350</span>
      </p>
      <button type="button" className="primary button-icon" disabled={!canAdd} onClick={(e) => onAdd(e.currentTarget)}>
        <Icon name="plus" />
        Добавить трату вручную
      </button>
    </div>
  );
}
