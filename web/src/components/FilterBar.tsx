import { useRef, useState } from 'react';
import type { Category, Member } from '../api/types';
import type { Filters } from '../lib/filters';
import { memberOptionLabel, type NameMap } from '../lib/members';
import { detectPreset, PERIOD_OPTIONS, presetRange, type PeriodPreset } from '../lib/periods';
import { Icon } from './Icon';

interface FilterBarProps {
  filters: Filters;
  members: Member[];
  categories: Category[];
  names: NameMap;
  today: string;
  activeCount: number;
  periodError: string | null;
  onChange: (patch: Partial<Filters>) => void;
  onReset: () => void;
}

export function FilterBar(props: FilterBarProps) {
  const { filters, members, categories, names, today, activeCount, periodError, onChange, onReset } = props;
  const [open, setOpen] = useState(false);
  // «Свой период» не меняет даты, поэтому помним, для каких границ его выбрали.
  const [customFor, setCustomFor] = useState<string | null>(null);
  const fromRef = useRef<HTMLInputElement>(null);

  const rangeKey = `${filters.from}|${filters.to}`;
  const preset: PeriodPreset = customFor === rangeKey ? 'custom' : detectPreset(filters.from, filters.to, today);
  const memberIds = members.map((m) => String(m.user_id));
  const unknownMember = filters.user_id !== '' && !memberIds.includes(filters.user_id);
  const unknownCategory = filters.category !== '' && !categories.some((c) => c.key === filters.category);

  const choosePreset = (value: PeriodPreset) => {
    const range = presetRange(value, today);
    if (range) {
      setCustomFor(null);
      onChange(range);
    } else {
      setCustomFor(rangeKey);
      fromRef.current?.focus();
    }
  };

  return (
    <>
      <button
        type="button"
        className="mobile-filter"
        aria-expanded={open}
        aria-controls="filters"
        onClick={() => setOpen((o) => !o)}
      >
        <span>
          <Icon name="filter" />
          Фильтры ({activeCount})
        </span>
        <Icon name="chevron" />
      </button>
      <aside className={open ? 'filter-box open' : 'filter-box'} id="filters" aria-label="Фильтры трат">
        <div className="filter-title">
          Фильтры
          <button type="button" onClick={onReset} disabled={activeCount === 0}>
            Сбросить всё
          </button>
        </div>
        <div className="filter-field">
          <label htmlFor="filter-user">Член семьи</label>
          <select id="filter-user" value={filters.user_id} onChange={(e) => onChange({ user_id: e.target.value })}>
            <option value="">Все члены семьи</option>
            {members.map((m) => (
              <option key={m.user_id} value={String(m.user_id)}>
                {memberOptionLabel(m.user_id, names)}
              </option>
            ))}
            {unknownMember && <option value={filters.user_id}>ID {filters.user_id}</option>}
          </select>
        </div>
        <div className="filter-field">
          <label htmlFor="filter-category">Категория</label>
          <select
            id="filter-category"
            value={filters.category}
            onChange={(e) => onChange({ category: e.target.value })}
          >
            <option value="">Все категории</option>
            {categories.map((c) => (
              <option key={c.key} value={c.key}>
                {c.label}
              </option>
            ))}
            {unknownCategory && <option value={filters.category}>{filters.category}</option>}
          </select>
        </div>
        <div className="filter-field period-field">
          <label htmlFor="filter-period">Период</label>
          <select id="filter-period" value={preset} onChange={(e) => choosePreset(e.target.value as PeriodPreset)}>
            {PERIOD_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div className="date-pair">
          <div>
            <label htmlFor="filter-from">С</label>
            <input
              ref={fromRef}
              id="filter-from"
              type="date"
              value={filters.from}
              aria-invalid={periodError ? true : undefined}
              aria-describedby="period-error"
              onChange={(e) => onChange({ from: e.target.value })}
            />
          </div>
          <div>
            <label htmlFor="filter-to">По</label>
            <input
              id="filter-to"
              type="date"
              value={filters.to}
              aria-invalid={periodError ? true : undefined}
              aria-describedby="period-error"
              onChange={(e) => onChange({ to: e.target.value })}
            />
          </div>
        </div>
      </aside>
      <p className="error period-error" id="period-error" role="alert">
        {periodError ?? ''}
      </p>
    </>
  );
}
