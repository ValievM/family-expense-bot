import { useEffect, useMemo, useRef, useState } from 'react';
import { ApiError, createExpense, deleteExpense, updateExpense } from './api/client';
import type { Expense } from './api/types';
import { ConfirmDialog } from './components/ConfirmDialog';
import { DayGroup } from './components/DayGroup';
import { EmptyState } from './components/EmptyState';
import { ErrorBanner } from './components/ErrorBanner';
import { ExpenseForm } from './components/ExpenseForm';
import { FilterBar } from './components/FilterBar';
import { FilterChips } from './components/FilterChips';
import { Header } from './components/Header';
import { Icon } from './components/Icon';
import { Skeleton } from './components/Skeleton';
import { Summary } from './components/Summary';
import { Toast } from './components/Toast';
import { useCategories } from './hooks/useCategories';
import { useExpenses } from './hooks/useExpenses';
import { useFilters } from './hooks/useFilters';
import { useHighlight } from './hooks/useHighlight';
import { useMemberNames } from './hooks/useMemberNames';
import { useMembers } from './hooks/useMembers';
import { useTheme } from './hooks/useTheme';
import { useToast } from './hooks/useToast';
import { bannerMessage, errorTarget, NOT_FOUND_MESSAGE, PERIOD_MESSAGE } from './lib/apiErrors';
import { todayISO } from './lib/date';
import { hasActiveFilters, matchesFilters } from './lib/filters';
import { groupByDay } from './lib/groupByDay';
import { buildDirectory, memberView } from './lib/members';
import { formatRange } from './lib/periods';
import { totalKopecks } from './lib/total';

function asApiError(err: unknown): ApiError {
  return err instanceof ApiError ? err : new ApiError(0, 'unknown', String(err));
}

function isTyping(el: Element | null): boolean {
  return el instanceof HTMLElement && (el.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(el.tagName));
}

export function App() {
  const [theme, toggleTheme] = useTheme();
  const [filters, patchFilters, resetFilters] = useFilters();
  const expenses = useExpenses(filters);
  const categories = useCategories();
  const members = useMembers();
  const [names, rename] = useMemberNames();
  const { toast, show } = useToast();
  const [highlightId, flash] = useHighlight();
  const [form, setForm] = useState<{ expense: Expense | null } | null>(null);
  const [toDelete, setToDelete] = useState<Expense | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<ApiError | null>(null);
  const opener = useRef<HTMLElement | null>(null);
  const addRef = useRef<HTMLButtonElement>(null);
  const mobileAddRef = useRef<HTMLButtonElement>(null);

  const today = todayISO();
  const items = expenses.data;
  const groups = useMemo(() => (items ? groupByDay(items) : []), [items]);
  const people = useMemo(() => new Set((items ?? []).map((e) => e.user_id)).size, [items]);
  const memberData = members.data;
  const directory = useMemo(
    () => buildDirectory((items ?? []).map((e) => e.user_id), memberData ?? [], names),
    [items, memberData, names],
  );
  const canAdd = categories.data !== null && members.data !== null;
  const activeCount = (filters.user_id ? 1 : 0) + (filters.category ? 1 : 0) + (filters.from || filters.to ? 1 : 0);
  const periodInvalid = filters.from !== '' && filters.to !== '' && filters.from > filters.to;
  const periodError = periodInvalid || expenses.error?.code === 'invalid_period' ? PERIOD_MESSAGE : null;
  const listError = expenses.error && expenses.error.code !== 'invalid_period' ? expenses.error : null;
  const refsError = categories.error ?? members.error;
  const refreshing = expenses.loading && items !== null;

  const restoreFocus = () => {
    const el = opener.current;
    opener.current = null;
    window.setTimeout(() => {
      if (el?.isConnected) el.focus();
      else (mobileAddRef.current?.offsetParent ? mobileAddRef.current : addRef.current)?.focus();
    }, 0);
  };

  const openForm = (expense: Expense | null, from: Element | null) => {
    opener.current = from instanceof HTMLElement ? from : null;
    setForm({ expense });
  };

  const openDelete = (expense: Expense, from: HTMLElement | null) => {
    if (!form) opener.current = from;
    setDeleteError(null);
    setToDelete(expense);
  };

  const closeAll = () => {
    setForm(null);
    setToDelete(null);
    restoreFocus();
  };

  const onNotFound = () => {
    closeAll();
    expenses.reload();
    show(NOT_FOUND_MESSAGE);
  };

  const onSaved = (saved: Expense, created: boolean) => {
    closeAll();
    expenses.reload();
    members.reload();
    flash(saved.id);
    const base = `${created ? 'Трата добавлена' : 'Трата изменена'} · ${memberView(directory, saved.user_id).label}`;
    show(matchesFilters(saved, filters) ? base : `${base}. Она не попадает в текущие фильтры.`);
  };

  const confirmDelete = async () => {
    if (!toDelete) return;
    const who = memberView(directory, toDelete.user_id).label;
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteExpense(toDelete.id);
      closeAll();
      expenses.reload();
      members.reload();
      show(`Трата удалена · ${who}`);
    } catch (err) {
      const apiErr = asApiError(err);
      if (errorTarget(apiErr.code).kind === 'notFound') onNotFound();
      else setDeleteError(apiErr);
    } finally {
      setDeleting(false);
    }
  };

  const formOpen = form !== null || toDelete !== null;
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.code !== 'KeyN' || e.metaKey || e.ctrlKey || e.altKey || e.repeat) return;
      if (formOpen || !canAdd || isTyping(document.activeElement)) return;
      e.preventDefault();
      opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      setForm({ expense: null });
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [formOpen, canAdd]);

  const memberList = members.data ?? [];
  const categoryList = categories.data ?? [];
  const defaults = {
    userId: memberList.some((m) => String(m.user_id) === filters.user_id)
      ? filters.user_id
      : String(memberList[0]?.user_id ?? ''),
    category: categoryList.some((c) => c.key === filters.category) ? filters.category : (categoryList[0]?.key ?? ''),
  };

  let list;
  if (items === null) {
    list = expenses.loading ? (
      <>
        <p className="visually-hidden" role="status">
          Загружаем траты…
        </p>
        <Skeleton />
      </>
    ) : null;
  } else if (items.length === 0) {
    list = (
      <EmptyState
        kind={hasActiveFilters(filters) ? 'filtered' : 'first'}
        canAdd={canAdd}
        onAdd={(el) => openForm(null, el)}
        onReset={resetFilters}
      />
    );
  } else {
    list = (
      <ol className="list" aria-label="Траты по дням, новые сверху">
        {groups.map((g) => (
          <DayGroup
            key={g.date}
            group={g}
            today={today}
            directory={directory}
            highlightId={highlightId}
            onEdit={(e, el) => openForm(e, el)}
            onDelete={openDelete}
          />
        ))}
      </ol>
    );
  }

  return (
    <>
      <Header theme={theme} onToggleTheme={toggleTheme} />
      <main>
        <div className="page-heading">
          <div>
            <div className="eyebrow">Семейная книга расходов</div>
            <h1>День за днём</h1>
            <p className="subheading">{formatRange(filters.from, filters.to)}</p>
          </div>
          <button
            ref={addRef}
            type="button"
            className="primary button-icon"
            disabled={!canAdd}
            onClick={(e) => openForm(null, e.currentTarget)}
          >
            <Icon name="plus" />
            Добавить трату
          </button>
        </div>

        {refsError && (
          <ErrorBanner
            message={`Не удалось загрузить списки членов семьи и категорий. ${bannerMessage(refsError.code)}`}
            onRetry={() => {
              categories.reload();
              members.reload();
            }}
          />
        )}

        <FilterBar
          filters={filters}
          members={memberList}
          categories={categoryList}
          directory={directory}
          today={today}
          activeCount={activeCount}
          periodError={periodError}
          onChange={patchFilters}
          onReset={resetFilters}
        />

        <section className="content" aria-labelledby="list-title">
          <h2 id="list-title" className="visually-hidden">
            Траты
          </h2>
          <Summary
            total={items ? totalKopecks(items) : null}
            count={items?.length ?? 0}
            people={people}
            members={Math.max(memberList.length, people)}
          />
          <FilterChips
            filters={filters}
            categories={categoryList}
            names={names}
            directory={directory}
            onChange={patchFilters}
            onReset={resetFilters}
            onRename={rename}
          />
          {listError && <ErrorBanner message={bannerMessage(listError.code)} onRetry={expenses.reload} />}
          <div className="list-meta">
            {refreshing ? (
              <span className="refreshing" role="status">
                Обновляем список…
              </span>
            ) : (
              <span>Траты · новые сверху</span>
            )}
            {items && items.length > 0 && <span className="edit-hint">Нажмите, чтобы исправить</span>}
          </div>
          <div className={refreshing ? 'list-wrap is-refreshing' : 'list-wrap'} aria-busy={expenses.loading}>
            {list}
          </div>
          {items && items.length > 0 && (
            <div className="list-foot">
              <span>Все записи выборки показаны</span>
              <span>Суммы в рублях</span>
            </div>
          )}
        </section>
      </main>

      <div className="mobile-bottom">
        <button
          ref={mobileAddRef}
          type="button"
          className="primary button-icon"
          disabled={!canAdd}
          onClick={(e) => openForm(null, e.currentTarget)}
        >
          <Icon name="plus" />
          Добавить трату
        </button>
      </div>

      {form && canAdd && (
        <ExpenseForm
          expense={form.expense}
          members={memberList}
          categories={categoryList}
          names={names}
          directory={directory}
          today={today}
          defaults={defaults}
          onRename={rename}
          onCreate={createExpense}
          onUpdate={updateExpense}
          onSaved={onSaved}
          onNotFound={onNotFound}
          onDelete={(e) => openDelete(e, null)}
          onClose={() => {
            setForm(null);
            restoreFocus();
          }}
        />
      )}
      {toDelete && (
        <ConfirmDialog
          expense={toDelete}
          directory={directory}
          deleting={deleting}
          error={deleteError}
          onConfirm={() => void confirmDelete()}
          onCancel={() => {
            setToDelete(null);
            if (!form) restoreFocus();
          }}
        />
      )}
      <Toast toast={toast} />
    </>
  );
}
