import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react';
import { MAX_NAME_LENGTH, shortId } from '../lib/members';

interface MemberNameEditorProps {
  id: number;
  name: string | undefined;
  onSave: (id: number, name: string) => void;
  className?: string;
}

export function MemberNameEditor({ id, name, onSave, className = 'link-btn' }: MemberNameEditorProps) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState('');
  const inputId = useId();
  const hintId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const wasOpen = useRef(false);

  useEffect(() => {
    if (open) inputRef.current?.focus();
    else if (wasOpen.current) buttonRef.current?.focus();
    wasOpen.current = open;
  }, [open]);

  const save = () => {
    onSave(id, draft);
    setOpen(false);
  };

  // Редактор живёт и внутри формы траты: Enter не должен отправлять её, а Esc — закрывать диалог.
  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      save();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      setOpen(false);
    }
  };

  if (!open) {
    return (
      <button
        ref={buttonRef}
        type="button"
        className={className}
        aria-label={`Переименовать ${name ?? `ID ${shortId(id)}`}`}
        title="Имя хранится только на этом устройстве"
        onClick={() => {
          setDraft(name ?? '');
          setOpen(true);
        }}
      >
        Переименовать
      </button>
    );
  }

  return (
    <div className="rename" role="group" aria-label={`Имя для ID ${shortId(id)}`}>
      <label htmlFor={inputId}>Имя для ID {shortId(id)}</label>
      <div className="rename-row">
        <input
          ref={inputRef}
          id={inputId}
          value={draft}
          maxLength={MAX_NAME_LENGTH}
          placeholder="Например, Мама"
          autoComplete="off"
          aria-describedby={hintId}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKeyDown}
        />
        <button type="button" onClick={save}>
          Сохранить
        </button>
        <button type="button" className="quiet" onClick={() => setOpen(false)}>
          Отмена
        </button>
      </div>
      <p className="helper" id={hintId}>
        Имя видно только в этом браузере на этом устройстве. Пустое поле вернёт ID.
      </p>
    </div>
  );
}
