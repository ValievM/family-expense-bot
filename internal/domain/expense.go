package domain

import "time"

// ExpenseID — ноль означает «ещё не сохранён».
type ExpenseID int64

// Expense — корень агрегата; состояние меняется только через NewExpense, Restore и Apply.
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

type NewExpenseInput struct {
	Owner    UserID
	Amount   Money
	Category Category
	SpentOn  Day
	Comment  Comment
	At       time.Time
}

// Changes — частичное изменение; nil означает «поле не меняется».
type Changes struct {
	Amount   *Money
	Category *Category
	SpentOn  *Day
	Comment  *Comment
}

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
// Делает: проверяет инварианты; createdAt и updatedAt равны input.At.
// Возвращает: Expense без ID либо доменную ошибку.
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
// Делает: восстанавливает расход, проверяя инварианты, чтобы повреждённые данные не попали в домен.
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
// Делает: применяет изменения к копии и проверяет инварианты; исходный расход, владелец и ID не меняются.
// Возвращает: Expense с новым updatedAt либо доменную ошибку.
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
// Делает: копирует расход с ID, присвоенным хранилищем после вставки.
// Возвращает: Expense с ID.
func (expense Expense) WithID(id ExpenseID) Expense {
	expense.id = id
	return expense
}

// Принимает: user — предполагаемый владелец.
// Делает: сравнивает владельца расхода с user.
// Возвращает: true, если расход принадлежит user.
func (expense Expense) OwnedBy(user UserID) bool {
	return expense.owner == user
}

func (expense Expense) ID() ExpenseID { return expense.id }

func (expense Expense) Owner() UserID { return expense.owner }

func (expense Expense) Amount() Money { return expense.amount }

func (expense Expense) Category() Category { return expense.category }

func (expense Expense) SpentOn() Day { return expense.spentOn }

func (expense Expense) Comment() Comment { return expense.comment }

func (expense Expense) CreatedAt() time.Time { return expense.createdAt }

func (expense Expense) UpdatedAt() time.Time { return expense.updatedAt }

// Принимает: ничего.
// Делает: проверяет инварианты в порядке owner, amount, category, day.
// Возвращает: nil либо первую найденную ошибку.
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
