import type { Category } from '../api/types';
import type { Filters } from '../lib/filters';
import { memberView, type MemberDirectory, type NameMap } from '../lib/members';
import { periodChipLabel } from '../lib/periods';
import { Icon } from './Icon';
import { MemberNameEditor } from './MemberNameEditor';

interface FilterChipsProps {
  filters: Filters;
  categories: Category[];
  names: NameMap;
  directory: MemberDirectory;
  onChange: (patch: Partial<Filters>) => void;
  onReset: () => void;
  onRename: (id: number, name: string) => void;
}

export function FilterChips({ filters, categories, names, directory, onChange, onReset, onRename }: FilterChipsProps) {
  const memberId = /^\d+$/.test(filters.user_id) ? Number(filters.user_id) : null;
  const member = memberId === null ? null : memberView(directory, memberId);
  const chips: { key: string; label: string; title?: string; clear: Partial<Filters> }[] = [];
  if (filters.user_id) {
    const who = member ? member.label : filters.user_id;
    chips.push({ key: 'user', label: `Член семьи: ${who}`, title: member?.details, clear: { user_id: '' } });
  }
  if (filters.category) {
    const label = categories.find((c) => c.key === filters.category)?.label ?? filters.category;
    chips.push({ key: 'category', label, clear: { category: '' } });
  }
  if (filters.from || filters.to) {
    chips.push({ key: 'period', label: periodChipLabel(filters.from, filters.to), clear: { from: '', to: '' } });
  }
  if (chips.length === 0) return null;

  return (
    <div className="active-filters">
      {chips.map((c) => (
        <button
          key={c.key}
          type="button"
          className="chip"
          title={c.title}
          aria-label={`Сбросить фильтр: ${c.label}`}
          onClick={() => onChange(c.clear)}
        >
          {c.label}
          <Icon name="close" />
        </button>
      ))}
      {memberId !== null && member && (
        <MemberNameEditor
          id={memberId}
          name={names[String(memberId)]}
          label={member.label}
          onSave={onRename}
          className="quiet chip"
        />
      )}
      {chips.length > 1 && (
        <button type="button" className="quiet chip" onClick={onReset}>
          Сбросить всё
        </button>
      )}
    </div>
  );
}
