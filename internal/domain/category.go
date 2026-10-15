// Package domain defines expense values and their business validation without adapter dependencies.
package domain

// Category is the stable key used to classify an expense.
type Category string

// Category identifiers remain stable because storage and consumers share these keys.
const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

type categoryOption struct {
	key   Category
	label string
}

var categoryOptions = [...]categoryOption{
	{key: CategoryFood, label: "еда"},
	{key: CategoryTransport, label: "транспорт"},
	{key: CategoryHousing, label: "жильё"},
	{key: CategoryEntertainment, label: "развлечения"},
	{key: CategoryHealth, label: "здоровье"},
	{key: CategoryShopping, label: "покупки"},
	{key: CategoryOther, label: "другое"},
}

// Label returns the localized display text shared by human-facing surfaces.
func (category Category) Label() string {
	for _, option := range categoryOptions {
		if option.key == category {
			return option.label
		}
	}
	return string(category)
}

// IsValid reports whether the category belongs to the closed domain set.
func (category Category) IsValid() bool {
	for _, option := range categoryOptions {
		if option.key == category {
			return true
		}
	}
	return false
}

// Categories returns the stable category order without exposing the internal catalog.
func Categories() []Category {
	categories := make([]Category, len(categoryOptions))
	for index, option := range categoryOptions {
		categories[index] = option.key
	}
	return categories
}
