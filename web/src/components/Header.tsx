import type { Theme } from '../lib/theme';
import { Icon } from './Icon';
import { ThemeToggle } from './ThemeToggle';

export function Header({ theme, onToggleTheme }: { theme: Theme; onToggleTheme: () => void }) {
  return (
    <header className="app-header">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">
          <Icon name="logo" />
        </span>
        Семейные расходы
      </div>
      <div className="header-right">
        <ThemeToggle theme={theme} onToggle={onToggleTheme} />
      </div>
    </header>
  );
}
