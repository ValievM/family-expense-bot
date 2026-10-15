import { useState, type FormEvent } from 'react';
import { ApiError } from '../api/client';
import type { Category, CreateExpense, Expense, Member, UpdateExpense } from '../api/types';
import { initialForm, validateForm, type ExpenseFormState, type FormErrors } from '../lib/expenseForm';
import { buildPatch } from '../lib/patch';
import { Modal } from './Modal';

const NEW_MEMBER = '__new__';

/** Свойства диалога создания и правки расхода. */
interface ExpenseDialogProps {
  expense: Expense | null;
  members: Member[];
  categories: Category[];
  onCreate: (input: CreateExpense) => Promise<void>;
  onUpdate: (id: number, patch: UpdateExpense) => Promise<void>;
  onClose: () => void;
}

/**
 * Принимает: расход (null для создания), справочники и обработчики сохранения и закрытия.
 * Делает: показывает форму, валидирует её как API, при правке отправляет только изменённые поля и выводит ошибки API.
 * Возвращает: React-элемент модальной формы.
 */
export function ExpenseDialog({ expense, members, categories, onCreate, onUpdate, onClose }: ExpenseDialogProps) {
  const editing = expense !== null;
  const [form, setForm] = useState<ExpenseFormState>(() => initialForm(expense));
  const [newMember, setNewMember] = useState(!editing && members.length === 0);
  const [errors, setErrors] = useState<FormErrors>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  /**
   * Принимает: имя поля и новое значение.
   * Делает: обновляет одно поле формы и сбрасывает его ошибку.
   * Возвращает: ничего.
   */
  const set = (key: keyof ExpenseFormState, value: string) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => ({ ...e, [key]: undefined }));
  };

  /**
   * Принимает: событие отправки формы.
   * Делает: валидирует, отправляет POST или PATCH, при успехе закрывает окно, при ошибке показывает сообщение.
   * Возвращает: Promise без значения.
   */
  const submit = async (ev: FormEvent) => {
    ev.preventDefault();
    const res = validateForm(form, categories.map((c) => c.key));
    if (!res.values) {
      setErrors(res.errors);
      return;
    }
    const v = res.values;
    setApiError(null);
    if (expense) {
      const patch = buildPatch(expense, v);
      if (Object.keys(patch).length === 0) {
        onClose();
        return;
      }
      setSaving(true);
      try {
        await onUpdate(expense.id, patch);
      } catch (e) {
        setApiError(e instanceof ApiError ? e.message : 'Не удалось сохранить');
        setSaving(false);
      }
      return;
    }
    setSaving(true);
    try {
      await onCreate({ user_id: v.userId, amount: v.amount, category: v.category, spent_at: v.spentAt, comment: v.comment });
    } catch (e) {
      setApiError(e instanceof ApiError ? e.message : 'Не удалось сохранить');
      setSaving(false);
    }
  };

  return (
    <Modal title={editing ? 'Изменить расход' : 'Новый расход'} onClose={saving ? () => undefined : onClose}>
      <form onSubmit={submit} noValidate className="form">
        <div className="field">
          <label htmlFor="f-user" className="field__label">Участник (Telegram ID)</label>
          {editing ? (
            <input id="f-user" value={form.userId} readOnly aria-readonly="true" />
          ) : (
            <>
              <select
                id="f-user"
                value={newMember ? NEW_MEMBER : form.userId}
                aria-invalid={!!errors.userId}
                onChange={(e) => {
                  const isNew = e.target.value === NEW_MEMBER;
                  setNewMember(isNew);
                  set('userId', isNew ? '' : e.target.value);
                }}
              >
                {!newMember && form.userId === '' && <option value="">Выберите…</option>}
                {members.map((m) => (
                  <option key={m.user_id} value={String(m.user_id)}>{m.user_id}</option>
                ))}
                <option value={NEW_MEMBER}>Новый ID…</option>
              </select>
              {newMember && (
                <input
                  aria-label="Новый Telegram ID"
                  inputMode="numeric"
                  placeholder="Например, 123456789"
                  value={form.userId}
                  aria-invalid={!!errors.userId}
                  onChange={(e) => set('userId', e.target.value)}
                />
              )}
            </>
          )}
          {errors.userId && <span className="field__error" role="alert">{errors.userId}</span>}
        </div>

        <div className="field">
          <label htmlFor="f-amount" className="field__label">Сумма, ₽</label>
          <input id="f-amount" inputMode="decimal" autoComplete="off" placeholder="350,00" value={form.amount} aria-invalid={!!errors.amount} onChange={(e) => set('amount', e.target.value)} />
          {errors.amount && <span className="field__error" role="alert">{errors.amount}</span>}
        </div>

        <div className="field">
          <label htmlFor="f-cat" className="field__label">Категория</label>
          <select id="f-cat" value={form.category} aria-invalid={!!errors.category} onChange={(e) => set('category', e.target.value)}>
            <option value="">Выберите…</option>
            {categories.map((c) => (
              <option key={c.key} value={c.key}>{c.label}</option>
            ))}
          </select>
          {errors.category && <span className="field__error" role="alert">{errors.category}</span>}
        </div>

        <div className="field">
          <label htmlFor="f-date" className="field__label">Дата</label>
          <input id="f-date" type="date" value={form.spentAt} aria-invalid={!!errors.spentAt} onChange={(e) => set('spentAt', e.target.value)} />
          {errors.spentAt && <span className="field__error" role="alert">{errors.spentAt}</span>}
        </div>

        <div className="field">
          <label htmlFor="f-comment" className="field__label">Комментарий</label>
          <textarea id="f-comment" rows={2} value={form.comment} onChange={(e) => set('comment', e.target.value)} />
        </div>

        {apiError && <p className="form-error" role="alert">{apiError}</p>}

        <div className="modal__actions">
          <button type="button" className="btn" onClick={onClose} disabled={saving}>Отмена</button>
          <button type="submit" className="btn btn--primary" disabled={saving}>
            {saving ? 'Сохранение…' : 'Сохранить'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
