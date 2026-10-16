import type { CSSProperties } from 'react';
import { memberHue, monogram } from '../lib/members';

export function Avatar({ id, name }: { id: number; name?: string }) {
  const style = { '--hue': memberHue(id) } as CSSProperties;
  return (
    <span className="avatar" style={style} aria-hidden="true">
      {monogram(id, name)}
    </span>
  );
}
