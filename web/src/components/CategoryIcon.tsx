import { categoryIconPath } from '../lib/categoryIcons';

export function CategoryIcon({ category }: { category: string }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d={categoryIconPath(category)} />
    </svg>
  );
}
