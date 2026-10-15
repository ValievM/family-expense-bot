import { useState } from 'react';
import { ApiError, createExpense, deleteExpense, updateExpense } from './api/client';
import type { CreateExpense, Expense, UpdateExpense } from './api/types';
import { ConfirmDialog } from './components/ConfirmDialog';
import { ErrorBanner } from './components/ErrorBanner';
import { ExpenseDialog } from './components/ExpenseDialog';
import { ExpenseTable } from './components/ExpenseTable';
import { FilterBar } from './components/FilterBar';
import { useCategories } from './hooks/useCategories';
import { useExpenses } from './hooks/useExpenses';
import { useFilters } from './hooks/useFilters';
import { useMembers } from './hooks/useMembers';

/** Какой диалог сейчас открыт. */
type DialogState = { kind: 'none' } | { kind: 'form'; expense: Expense | null } | { kind: 'delete'; expense: Expense };

/**
 * Принимает: ничего.
 * Делает: собирает страницу: фильтры, таблицу, состояния загрузки и ошибок, диалоги; после изменений перезагружает данные.
 * Возвращает: корневой React-элемент приложения.
 */
export function App() {
  const [filters, patchFilters, resetFilters] = useFilters();
  const expenses = useExpenses(filters);
  const categories = useCategories();
  const members = useMembers();
  const [dialog, setDialog] = useState<DialogState>({ kind: 'none' });
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  /**
   * Принимает: ничего.
   * Делает: закрывает диалог и обновляет расходы и список участников.
   * Возвращает: ничего.
   */
  const afterMutation = () => {
    setDialog({ kind: 'none' });
    expenses.reload();
    members.reload();
  };

  /**
   * Принимает: данные нового расхода.
   * Делает: отправляет создание и обновляет данные; ошибка уходит в диалог.
   * Возвращает: Promise без значения.
   */
  const handleCreate = async (input: CreateExpense) => {
    await createExpense(input);
    afterMutation();
  };

  /**
   * Принимает: id расхода и патч.
   * Делает: отправляет изменение и обновляет данные; ошибка уходит в диалог.
   * Возвращает: Promise без значения.
   */
  const handleUpdate = async (id: number, patch: UpdateExpense) => {
    await updateExpense(id, patch);
    afterMutation();
  };

  /**
   * Принимает: расход, который нужно удалить.
   * Делает: отправляет удаление, блокирует кнопки на время запроса, при ошибке показывает её в диалоге.
   * Возвращает: Promise без значения.
   */
  const handleDelete = async (e: Expense) => {
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteExpense(e.id);
      afterMutation();
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : 'Не удалось удалить');
    } finally {
      setDeleting(false);
    }
  };

  const refsReady = categories.data !== null && members.data !== null;
  const items = expenses.data;

  return (
    <div className="app">
      <header className="app__header">
        <h1>Семейные расходы</h1>
        <button type="button" className="btn btn--primary" disabled={!refsReady} onClick={() => setDialog({ kind: 'form', expense: null })}>
          Добавить расход
        </button>
      </header>

      <main>
        {(categories.error || members.error) && (
          <ErrorBanner
            message={(categories.error ?? members.error)?.message ?? 'Ошибка загрузки справочников'}
            onRetry={() => {
              categories.reload();
              members.reload();
            }}
          />
        )}

        <FilterBar
          filters={filters}
          members={members.data ?? []}
          categories={categories.data ?? []}
          onChange={patchFilters}
          onReset={resetFilters}
        />

        {expenses.error && <ErrorBanner message={expenses.error.message} onRetry={expenses.reload} />}

        <section aria-live="polite" aria-busy={expenses.loading}>
          {expenses.loading && !items && <p className="state">Загрузка…</p>}
          {items && items.length === 0 && !expenses.error && (
            <p className="state">Расходов не найдено. Измените фильтры или добавьте расход.</p>
          )}
          {items && items.length > 0 && (
            <div className={expenses.loading ? 'is-loading' : undefined}>
              <ExpenseTable
                items={items}
                busy={expenses.loading}
                onEdit={(e) => setDialog({ kind: 'form', expense: e })}
                onDelete={(e) => {
                  setDeleteError(null);
                  setDialog({ kind: 'delete', expense: e });
                }}
              />
            </div>
          )}
        </section>
      </main>

      {dialog.kind === 'form' && refsReady && (
        <ExpenseDialog
          expense={dialog.expense}
          members={members.data ?? []}
          categories={categories.data ?? []}
          onCreate={handleCreate}
          onUpdate={handleUpdate}
          onClose={() => setDialog({ kind: 'none' })}
        />
      )}
      {dialog.kind === 'delete' && (
        <ConfirmDialog
          title="Удалить расход?"
          message={`Расход №${dialog.expense.id} (${dialog.expense.category_label}, участник ${dialog.expense.user_id}) будет удалён безвозвратно.`}
          confirmLabel="Удалить"
          saving={deleting}
          error={deleteError}
          onConfirm={() => void handleDelete(dialog.expense)}
          onCancel={() => setDialog({ kind: 'none' })}
        />
      )}
    </div>
  );
}
