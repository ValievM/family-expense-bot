// Package domain defines expense values and their business validation without adapter dependencies.
package domain

// Category is the stable key used to classify an expense.
type Category string

const (
	// CategoryFood is persisted under a stable key shared with APIs and tools.
	CategoryFood Category = "food"
	// CategoryTransport is persisted under a stable key shared with APIs and tools.
	CategoryTransport Category = "transport"
	// CategoryHousing is persisted under a stable key shared with APIs and tools.
	CategoryHousing Category = "housing"
	// CategoryEntertainment is persisted under a stable key shared with APIs and tools.
	CategoryEntertainment Category = "entertainment"
	// CategoryHealth is persisted under a stable key shared with APIs and tools.
	CategoryHealth Category = "health"
	// CategoryShopping is persisted under a stable key shared with APIs and tools.
	CategoryShopping Category = "shopping"
	// CategoryOther is persisted under a stable key shared with APIs and tools.
	CategoryOther Category = "other"
)

const (
	categoryFoodLabel          = "еда"
	categoryTransportLabel     = "транспорт"
	categoryHousingLabel       = "жильё"
	categoryEntertainmentLabel = "развлечения"
	categoryHealthLabel        = "здоровье"
	categoryShoppingLabel      = "покупки"
	categoryOtherLabel         = "другое"
)

// Label returns the localized display text shared by human-facing surfaces.
func (category Category) Label() string {
	switch category {
	case CategoryFood:
		return categoryFoodLabel
	case CategoryTransport:
		return categoryTransportLabel
	case CategoryHousing:
		return categoryHousingLabel
	case CategoryEntertainment:
		return categoryEntertainmentLabel
	case CategoryHealth:
		return categoryHealthLabel
	case CategoryShopping:
		return categoryShoppingLabel
	case CategoryOther:
		return categoryOtherLabel
	default:
		return string(category)
	}
}

// IsValid reports whether the category belongs to the closed domain set.
func (category Category) IsValid() bool {
	switch category {
	case CategoryFood, CategoryTransport, CategoryHousing, CategoryEntertainment, CategoryHealth, CategoryShopping, CategoryOther:
		return true
	default:
		return false
	}
}

// Date represents a calendar date without a time zone.
type Date struct {
	Year  int
	Month int
	Day   int
}

// IsValid checks the Gregorian calendar date without an infrastructure dependency.
func (date Date) IsValid() bool {
	if date.Year < 1 || date.Month < 1 || date.Month > 12 || date.Day < 1 {
		return false
	}

	daysInMonth := 31
	switch date.Month {
	case 2:
		daysInMonth = 28
		if isLeapYear(date.Year) {
			daysInMonth = 29
		}
	case 4, 6, 9, 11:
		daysInMonth = 30
	}

	return date.Day <= daysInMonth
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// Expense is a family's recorded spending with its calendar date and category.
type Expense struct {
	ID     int64
	UserID int64
	// Amount is measured in kopecks to keep RUB values exact.
	Amount   int64
	Category Category
	SpentAt  Date
	Comment  string
}

// ValidationError is a comparable domain error so callers can use errors.Is.
type ValidationError string

// Error returns the stable validation message used at application boundaries.
func (validationError ValidationError) Error() string {
	return string(validationError)
}

const (
	// ErrInvalidAmount identifies amounts that cannot represent a positive expense.
	ErrInvalidAmount ValidationError = "amount must be greater than zero"
	// ErrUnknownCategory identifies categories outside the closed domain set.
	ErrUnknownCategory ValidationError = "category is unknown"
	// ErrInvalidDate identifies dates that do not represent a Gregorian calendar day.
	ErrInvalidDate ValidationError = "date is invalid"
)

// Validate prevents invalid spending values from crossing the domain boundary.
func (expense Expense) Validate() error {
	if expense.Amount <= 0 {
		return ErrInvalidAmount
	}
	if !expense.Category.IsValid() {
		return ErrUnknownCategory
	}
	if !expense.SpentAt.IsValid() {
		return ErrInvalidDate
	}
	return nil
}
