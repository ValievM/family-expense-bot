import type { Theme } from '../lib/theme';
import { Icon } from './Icon';

export function ThemeToggle({ theme, onToggle }: { theme: Theme; onToggle: () => void }) {
  const dark = theme === 'dark';
  return (
    <button
      type="button"
      className="icon-btn"
      onClick={onToggle}
      aria-label={dark ? 'Включить светлую тему' : 'Включить тёмную тему'}
      title="Переключить тему"
    >
      <Icon name={dark ? 'sun' : 'moon'} />
    </button>
  );
}
