const PATHS = {
  logo: 'M4 5h16v15H4z M8 2v6 M16 2v6 M8 12h8 M8 16h5',
  plus: 'M12 5v14 M5 12h14',
  close: 'M6 6l12 12 M18 6L6 18',
  edit: 'M14 5l5 5 M4 20l4-1 12-12-4-4L4 15z',
  trash: 'M4 7h16 M10 11v6 M14 11v6 M6 7l1 13h10l1-13 M9 7V4h6v3',
  filter: 'M4 6h16 M7 12h10 M10 18h4 M8 3v6 M16 9v6 M12 15v6',
  chevron: 'M6 9l6 6 6-6',
  sun: 'M12 3v2 M12 19v2 M3 12h2 M19 12h2 M5.6 5.6L7 7 M17 17l1.4 1.4 M5.6 18.4L7 17 M17 7l1.4-1.4 M16 12a4 4 0 1 0-8 0a4 4 0 1 0 8 0',
  moon: 'M20 14a8 8 0 0 1-10-10A8 8 0 1 0 20 14',
  alert: 'M12 8v5 M12 16v.1 M21 12a9 9 0 1 0-18 0a9 9 0 1 0 18 0',
  message: 'M4 5h16v11H9l-5 4z M8 9h8 M8 12h5',
} as const;

export type IconName = keyof typeof PATHS;

export function Icon({ name }: { name: IconName }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d={PATHS[name]} />
    </svg>
  );
}
