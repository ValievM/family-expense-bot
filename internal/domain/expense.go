package domain

import "time"

// ExpenseID — идентификатор расхода; нулевое значение означает «ещё не сохранён».
type ExpenseID int64

// Expense — сущность и корень агрегата «расход». Поля закрыты: состояние меняется только через
// конструктор и Apply, поэтому ни один экземпляр не нарушает инварианты (положительная сумма,
// известная категория, заданные день и владелец).
type Expense struct {
	id        ExpenseID
	owner     UserID
	amount    Money
	category  Category
	spentOn   Day
	comment   Comment
	createdAt time.Time
	updatedAt time.Time
}

// NewExpenseInput — данные для создания нового расхода.
type NewExpenseInput struct {
	Owner    UserID
	Amount   Money
	Category Category
	SpentOn  Day
	Comment  Comment
	At       time.Time // момент создания
}

// Changes — частичное изменение расхода; nil означает «поле не меняется».
type Changes struct {
	Amount   *Money
	Category *Category
	SpentOn  *Day
	Comment  *Comment
}

// Snapshot — полное состояние расхода для восстановления из хранилища.
type Snapshot struct {
	ID        ExpenseID
	Owner     UserID
	Amount    Money
	Category  Category
	SpentOn   Day
	Comment   Comment
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Принимает: input — данные нового расхода.
// Делает: создаёт расход и проверяет все инварианты; время создания и изменения равны input.At.
// Возвращает: Expense без ID либо доменную ошибку (ErrInvalidUser, ErrInvalidAmount, ErrUnknownCategory, ErrInvalidDate).
func NewExpense(input NewExpenseInput) (Expense, error) {
	expense := Expense{
		owner: input.Owner, amount: input.Amount, category: input.Category,
		spentOn: input.SpentOn, comment: input.Comment,
		createdAt: input.At, updatedAt: input.At,
	}
	if err := expense.validate(); err != nil {
		return Expense{}, err
	}
	return expense, nil
}

// Принимает: snapshot — состояние, прочитанное из хранилища.
// Делает: восстанавливает расход и проверяет инварианты, чтобы повреждённые данные не попадали в домен.
// Возвращает: Expense либо доменную ошибку.
func Restore(snapshot Snapshot) (Expense, error) {
	expense := Expense{
		id: snapshot.ID, owner: snapshot.Owner, amount: snapshot.Amount, category: snapshot.Category,
		spentOn: snapshot.SpentOn, comment: snapshot.Comment,
		createdAt: snapshot.CreatedAt, updatedAt: snapshot.UpdatedAt,
	}
	if err := expense.validate(); err != nil {
		return Expense{}, err
	}
	return expense, nil
}

// Принимает: changes — изменяемые поля; now — момент изменения.
// Делает: применяет изменения к копии и проверяет инварианты целиком; исходный расход не меняется, владелец и ID неизменны.
// Возвращает: обновлённый Expense с новым временем изменения либо доменную ошибку.
func (expense Expense) Apply(changes Changes, now time.Time) (Expense, error) {
	updated := expense
	if changes.Amount != nil {
		updated.amount = *changes.Amount
	}
	if changes.Category != nil {
		updated.category = *changes.Category
	}
	if changes.SpentOn != nil {
		updated.spentOn = *changes.SpentOn
	}
	if changes.Comment != nil {
		updated.comment = *changes.Comment
	}
	if err := updated.validate(); err != nil {
		return Expense{}, err
	}
	updated.updatedAt = now
	return updated, nil
}

// Принимает: id — идентификатор, присвоенный хранилищем.
// Делает: возвращает копию расхода с ID; используется репозиторием после вставки.
// Возвращает: Expense с ID.
func (expense Expense) WithID(id ExpenseID) Expense {
	expense.id = id
	return expense
}

// Принимает: user — предполагаемый владелец.
// Делает: проверяет принадлежность расхода пользователю; правило владения живёт в домене.
// Возвращает: true, если расход принадлежит user.
func (expense Expense) OwnedBy(user UserID) bool {
	return expense.owner == user
}

// Принимает: ничего.
// Делает: читает идентификатор расхода.
// Возвращает: ExpenseID; ноль означает «ещё не сохранён».
func (expense Expense) ID() ExpenseID { return expense.id }

// Принимает: ничего.
// Делает: читает владельца.
// Возвращает: UserID.
func (expense Expense) Owner() UserID { return expense.owner }

// Принимает: ничего.
// Делает: читает сумму.
// Возвращает: Money.
func (expense Expense) Amount() Money { return expense.amount }

// Принимает: ничего.
// Делает: читает категорию.
// Возвращает: Category.
func (expense Expense) Category() Category { return expense.category }

// Принимает: ничего.
// Делает: читает день траты.
// Возвращает: Day.
func (expense Expense) SpentOn() Day { return expense.spentOn }

// Принимает: ничего.
// Делает: читает комментарий.
// Возвращает: Comment.
func (expense Expense) Comment() Comment { return expense.comment }

// Принимает: ничего.
// Делает: читает время создания.
// Возвращает: time.Time.
func (expense Expense) CreatedAt() time.Time { return expense.createdAt }

// Принимает: ничего.
// Делает: читает время последнего изменения.
// Возвращает: time.Time.
func (expense Expense) UpdatedAt() time.Time { return expense.updatedAt }

// Принимает: ничего.
// Делает: проверяет инварианты агрегата в фиксированном порядке.
// Возвращает: nil либо первую найденную доменную ошибку.
func (expense Expense) validate() error {
	switch {
	case expense.owner.IsZero():
		return ErrInvalidUser
	case !expense.amount.IsPositive():
		return ErrInvalidAmount
	case !expense.category.IsValid():
		return ErrUnknownCategory
	case expense.spentOn.IsZero():
		return ErrInvalidDate
	}
	return nil
}
