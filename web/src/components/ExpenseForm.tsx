import { useRef, useState, type FormEvent, type KeyboardEvent } from 'react';
import { ApiError } from '../api/client';
import type { Category, CreateExpense, Expense, Member, UpdateExpense } from '../api/types';
import { bannerMessage, errorTarget, hasTechnicalDetails } from '../lib/apiErrors';
import { addDays, formatDate, MONTHS_NOMINATIVE, parseISODate } from '../lib/date';
import {
  commentLength,
  FIELD_MESSAGES,
  initialForm,
  MAX_COMMENT_LENGTH,
  validateForm,
  type ExpenseFormState,
  type FormErrors,
  type FormField,
} from '../lib/expenseForm';
import type { NameMap } from '../lib/members';
import { formatRubles, parseRubles } from '../lib/money';
import { buildPatch } from '../lib/patch';
import { CategoryIcon } from './CategoryIcon';
import { Dialog } from './Dialog';
import { ErrorBanner } from './ErrorBanner';
import { Icon } from './Icon';
import { MemberNameEditor } from './MemberNameEditor';
import { Person } from './Person';

const NEW_MEMBER = '__new__';
const COMMENT_WARN_AT = 450;
const FIELD_ORDER: FormField[] = ['userId', 'amount', 'category', 'spentAt', 'comment'];

interface ExpenseFormProps {
  expense: Expense | null;
  members: Member[];
  categories: Category[];
  names: NameMap;
  today: string;
  defaults: { userId: string; category: string };
  onRename: (id: number, name: string) => void;
  onCreate: (input: CreateExpense) => Promise<Expense>;
  onUpdate: (id: number, patch: UpdateExpense) => Promise<Expense>;
  onSaved: (saved: Expense, created: boolean) => void;
  onNotFound: () => void;
  onDelete: (e: Expense) => void;
  onClose: () => void;
}

/**
 * Принимает: трату (null — создание), справочники, локальные имена и обработчики.
 * Делает: валидирует как API, раскладывает ошибки сервера по полям, при правке шлёт только изменения,
 *   спрашивает подтверждение перед закрытием с несохранёнными данными.
 * Возвращает: диалог-«лист книги» с формой.
 */
export function ExpenseForm(props: ExpenseFormProps) {
  const { expense, members, categories, names, today, defaults } = props;
  const editing = expense !== null;
  const [initial] = useState<ExpenseFormState>(() =>
    expense ? initialForm(expense) : { ...initialForm(null), spentAt: today, ...defaults },
  );
  const [form, setForm] = useState(initial);
  const [newMember, setNewMember] = useState(!editing && (members.length === 0 || initial.userId === ''));
  const [errors, setErrors] = useState<FormErrors>({});
  const [failure, setFailure] = useState<ApiError | null>(null);
  const [saving, setSaving] = useState(false);
  const amountRef = useRef<HTMLInputElement>(null);
  const formRef = useRef<HTMLFormElement>(null);

  const categoryKeys = categories.map((c) => c.key as string);
  const checked = validateForm(form, categoryKeys);
  const sameAsInitial = FIELD_ORDER.every((k) => form[k] === initial[k]);
  const patchEmpty = expense !== null && checked.values !== undefined && Object.keys(buildPatch(expense, checked.values)).length === 0;
  const unchanged = editing && (sameAsInitial || patchEmpty);
  const preview = parseRubles(form.amount);
  const length = commentLength(form.comment);
  const yesterday = addDays(today, -1);
  const now = parseISODate(today);

  const set = (key: FormField, value: string) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => ({ ...e, [key]: undefined }));
  };

  const showIfInvalid = (key: FormField) => {
    if (form[key].trim() === '') return;
    const message = validateForm(form, categoryKeys).errors?.[key];
    if (message) setErrors((e) => ({ ...e, [key]: message }));
  };

  const focusField = (field: FormField) => {
    const el = formRef.current?.querySelector<HTMLElement>(`[data-field="${field}"]`);
    el?.focus();
  };

  const requestClose = () => {
    if (saving) return;
    if (!sameAsInitial && !window.confirm('Закрыть форму без сохранения изменений?')) return;
    props.onClose();
  };

  const submit = async (ev?: FormEvent) => {
    ev?.preventDefault();
    if (saving || unchanged) return;
    const res = validateForm(form, categoryKeys);
    if (!res.values) {
      setErrors(res.errors);
      const first = FIELD_ORDER.find((f) => res.errors[f]);
      if (first) focusField(first);
      return;
    }
    const v = res.values;
    setFailure(null);
    setSaving(true);
    try {
      const saved = expense
        ? await props.onUpdate(expense.id, buildPatch(expense, v))
        : await props.onCreate({ user_id: v.userId, amount: v.amount, category: v.category, spent_at: v.spentAt, comment: v.comment });
      props.onSaved(saved, !expense);
    } catch (err) {
      setSaving(false);
      const apiErr = err instanceof ApiError ? err : new ApiError(0, 'unknown', String(err));
      const target = errorTarget(apiErr.code);
      if (target.kind === 'field') {
        setErrors({ [target.field]: FIELD_MESSAGES[target.field] });
        focusField(target.field);
      } else if (target.kind === 'notFound') {
        props.onNotFound();
      } else {
        setFailure(apiErr);
      }
    }
  };

  const onCommentKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      formRef.current?.requestSubmit();
    }
  };

  const invalid = (key: FormField) => (errors[key] ? true : undefined);
  const contextAmount =
    preview !== null ? formatRubles(preview) : expense ? formatRubles(expense.amount) : null;

  return (
    <Dialog className="sheet" labelledBy="form-title" onRequestClose={requestClose} initialFocus={amountRef}>
      <div className="dialog-shell">
        <aside className="form-context" aria-hidden="true">
          <div>
            <div className="eyebrow">Семейные расходы</div>
            <h3>
              {contextAmount ?? (
                <>
                  Новая
                  <br />
                  трата
                </>
              )}
            </h3>
            <p>
              {expense
                ? `${expense.category_label} · ${formatDate(expense.spent_at)} · ID ${expense.user_id}`
                : 'Выберите члена семьи и заполните сумму. Запись появится в общем списке.'}
            </p>
          </div>
          <p>
            {now ? `${MONTHS_NOMINATIVE[now.month - 1]} ${now.year}` : ''}
            <br />
            Семейная книга расходов
          </p>
        </aside>

        <div className="dialog-header">
          <div>
            <div className="eyebrow">{expense ? `Запись № ${expense.id}` : 'Новая запись'}</div>
            <h2 id="form-title">{editing ? 'Изменить трату' : 'Добавить трату'}</h2>
            <p>Проверьте сумму, категорию и дату.</p>
          </div>
          <button type="button" className="icon-btn quiet" aria-label="Закрыть форму" onClick={requestClose}>
            <Icon name="close" />
          </button>
        </div>

        <form ref={formRef} className="expense-form" noValidate onSubmit={submit}>
          <div className="form-body">
            {failure && (
              <ErrorBanner
                message={bannerMessage(failure.code)}
                details={hasTechnicalDetails(failure.code) ? `${failure.code}: ${failure.message}` : undefined}
                onRetry={() => void submit()}
              />
            )}

            <div className="field">
              {expense ? (
                <>
                  <div className="field-label" id="owner-label">
                    Член семьи
                  </div>
                  <div className="read-owner" role="group" aria-labelledby="owner-label">
                    <Person id={expense.user_id} names={names} />
                    <div>
                      {expense.user_id}
                      <small>Владельца существующей траты менять нельзя</small>
                    </div>
                  </div>
                  <MemberNameEditor id={expense.user_id} name={names[String(expense.user_id)]} onSave={props.onRename} />
                </>
              ) : (
                <>
                  <label htmlFor={members.length > 0 ? 'f-user' : 'f-new-user'}>Член семьи</label>
                  {members.length > 0 && (
                    <select
                      id="f-user"
                      data-field={newMember ? undefined : 'userId'}
                      value={newMember ? NEW_MEMBER : form.userId}
                      aria-invalid={newMember ? undefined : invalid('userId')}
                      aria-describedby="user-help user-error"
                      onChange={(e) => {
                        const isNew = e.target.value === NEW_MEMBER;
                        setNewMember(isNew);
                        set('userId', isNew ? '' : e.target.value);
                      }}
                    >
                      {members.map((m) => {
                        const name = names[String(m.user_id)];
                        return (
                          <option key={m.user_id} value={String(m.user_id)}>
                            {name ? `${name} · ${m.user_id}` : m.user_id}
                          </option>
                        );
                      })}
                      <option value={NEW_MEMBER}>+ Новый Telegram ID</option>
                    </select>
                  )}
                  {newMember && (
                    <div className="new-user-field">
                      {members.length > 0 && <label htmlFor="f-new-user">Новый Telegram ID</label>}
                      <input
                        id="f-new-user"
                        data-field="userId"
                        inputMode="numeric"
                        autoComplete="off"
                        placeholder="6612214276"
                        value={form.userId}
                        aria-invalid={invalid('userId')}
                        aria-describedby="user-help user-error"
                        onChange={(e) => set('userId', e.target.value)}
                        onBlur={() => showIfInvalid('userId')}
                      />
                    </div>
                  )}
                  <p className="helper" id="user-help">
                    {newMember
                      ? 'Telegram ID человека, который ещё не писал боту.'
                      : 'В списке — Telegram ID членов семьи.'}
                  </p>
                  <p className="error" id="user-error" role="alert">
                    {errors.userId ?? ''}
                  </p>
                  {!newMember && /^\d+$/.test(form.userId) && (
                    <MemberNameEditor id={Number(form.userId)} name={names[form.userId]} onSave={props.onRename} />
                  )}
                </>
              )}
            </div>

            <div className="field">
              <label htmlFor="f-amount">Сумма, ₽</label>
              <input
                ref={amountRef}
                id="f-amount"
                data-field="amount"
                className="amount-input"
                inputMode="decimal"
                autoComplete="off"
                placeholder="350,00"
                value={form.amount}
                aria-invalid={invalid('amount')}
                aria-describedby="amount-help amount-error"
                onChange={(e) => set('amount', e.target.value)}
                onBlur={() => showIfInvalid('amount')}
              />
              <p className="helper" id="amount-help">
                Например, 350 или 350,50. От 0,01 до 1 000 000 000 ₽.
              </p>
              <p className="error" id="amount-error" role="alert">
                {errors.amount ?? ''}
              </p>
            </div>

            <div className="field">
              <fieldset className="category-picks" aria-describedby="category-error">
                <legend>Категория</legend>
                {categories.map((c, i) => (
                  <label key={c.key}>
                    <input
                      type="radio"
                      name="category"
                      value={c.key}
                      data-field={form.category === c.key || (form.category === '' && i === 0) ? 'category' : undefined}
                      checked={form.category === c.key}
                      aria-invalid={invalid('category')}
                      onChange={() => set('category', c.key)}
                    />
                    <span>
                      <CategoryIcon category={c.key} />
                      {c.label}
                    </span>
                  </label>
                ))}
              </fieldset>
              <p className="error" id="category-error" role="alert">
                {errors.category ?? ''}
              </p>
            </div>

            <div className="field">
              <label htmlFor="f-date">Дата</label>
              <input
                id="f-date"
                data-field="spentAt"
                type="date"
                value={form.spentAt}
                aria-invalid={invalid('spentAt')}
                aria-describedby="date-error"
                onChange={(e) => set('spentAt', e.target.value)}
              />
              <div className="date-quick">
                <button type="button" aria-pressed={form.spentAt === today} onClick={() => set('spentAt', today)}>
                  Сегодня
                </button>
                <button type="button" aria-pressed={form.spentAt === yesterday} onClick={() => set('spentAt', yesterday)}>
                  Вчера
                </button>
              </div>
              <p className="error" id="date-error" role="alert">
                {errors.spentAt ?? ''}
              </p>
            </div>

            <div className="field">
              <label htmlFor="f-comment">
                Комментарий <span className="muted optional">· необязательно</span>
              </label>
              <textarea
                id="f-comment"
                data-field="comment"
                maxLength={MAX_COMMENT_LENGTH}
                placeholder="Например, продукты на неделю"
                value={form.comment}
                aria-invalid={invalid('comment')}
                aria-describedby="comment-help comment-error"
                onChange={(e) => set('comment', e.target.value)}
                onKeyDown={onCommentKeyDown}
              />
              <p className={length >= COMMENT_WARN_AT ? 'helper is-near' : 'helper'} id="comment-help">
                До {MAX_COMMENT_LENGTH} символов · <span className="num">{length}</span>/{MAX_COMMENT_LENGTH}
              </p>
              <p className="error" id="comment-error" role="alert">
                {errors.comment ?? ''}
              </p>
            </div>
          </div>

          <div className="dialog-footer">
            {expense && (
              <button type="button" className="quiet danger" disabled={saving} onClick={() => props.onDelete(expense)}>
                Удалить трату
              </button>
            )}
            <div className="footer-actions">
              <button type="button" onClick={requestClose} disabled={saving}>
                Отмена
              </button>
              <button
                type="submit"
                className="primary"
                disabled={saving || unchanged}
                aria-busy={saving}
                title={unchanged ? 'Изменений пока нет' : undefined}
              >
                {saving ? 'Сохраняем…' : editing ? 'Сохранить' : 'Добавить'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </Dialog>
  );
}
