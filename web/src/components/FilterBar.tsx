import type { Category, Member } from '../api/types';
import { hasActiveFilters, type Filters } from '../lib/filters';

interface FilterBarProps {
  filters: Filters;
  members: Member[];
  categories: Category[];
  onChange: (patch: Partial<Filters>) => void;
  onReset: () => void;
}

export function FilterBar({ filters, members, categories, onChange, onReset }: FilterBarProps) {
  return (
    <form className="filters" aria-label="Фильтры" onSubmit={(e) => e.preventDefault()}>
      <label className="field">
        <span className="field__label">Участник</span>
        <select value={filters.user_id} onChange={(e) => onChange({ user_id: e.target.value })}>
          <option value="">Все</option>
          {members.map((m) => (
            <option key={m.user_id} value={String(m.user_id)}>
              {m.user_id}
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span className="field__label">Категория</span>
        <select value={filters.category} onChange={(e) => onChange({ category: e.target.value })}>
          <option value="">Все</option>
          {categories.map((c) => (
            <option key={c.key} value={c.key}>
              {c.label}
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span className="field__label">С даты</span>
        <input type="date" value={filters.from} max={filters.to || undefined} onChange={(e) => onChange({ from: e.target.value })} />
      </label>
      <label className="field">
        <span className="field__label">По дату</span>
        <input type="date" value={filters.to} min={filters.from || undefined} onChange={(e) => onChange({ to: e.target.value })} />
      </label>
      <button type="button" className="btn" onClick={onReset} disabled={!hasActiveFilters(filters)}>
        Сбросить
      </button>
    </form>
  );
}
