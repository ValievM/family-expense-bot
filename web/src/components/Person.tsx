import { memberLabel, type NameMap } from '../lib/members';
import { Avatar } from './Avatar';

export function Person({ id, names }: { id: number; names: NameMap }) {
  const label = memberLabel(id, names);
  return (
    <span className="person" title={`Telegram ID ${id}`}>
      <Avatar id={id} name={names[String(id)]} />
      <span className="person-name">{label}</span>
    </span>
  );
}
