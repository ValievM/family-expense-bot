import type { MemberView } from '../lib/members';
import { Avatar } from './Avatar';

export function Person({ id, view }: { id: number; view: MemberView }) {
  return (
    <span className="person" title={view.details}>
      <Avatar id={id} name={view.name} />
      <span className="person-name">{view.label}</span>
    </span>
  );
}
