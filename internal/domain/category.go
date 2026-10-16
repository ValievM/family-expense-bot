// Package domain — значения расходов и их бизнес-валидация без зависимостей от адаптеров.
package domain

// Category — стабильный ключ, по которому классифицируется расход.
type Category string

// Идентификаторы категорий неизменны, так как хранилище и потребители используют одни и те же ключи.
const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

// categoryOption — пара из ключа категории и её отображаемого названия.
type categoryOption struct {
	key   Category
	label string
}

// categoryOptions — закрытый упорядоченный каталог категорий.
var categoryOptions = [...]categoryOption{
	{key: CategoryFood, label: "еда"},
	{key: CategoryTransport, label: "транспорт"},
	{key: CategoryHousing, label: "жильё"},
	{key: CategoryEntertainment, label: "развлечения"},
	{key: CategoryHealth, label: "здоровье"},
	{key: CategoryShopping, label: "покупки"},
	{key: CategoryOther, label: "другое"},
}

// Принимает: ничего (получатель — категория).
// Делает: ищет русское название категории в каталоге.
// Возвращает: название категории; для неизвестной категории — сам ключ как строку.
func (category Category) Label() string {
	for _, option := range categoryOptions {
		if option.key == category {
			return option.label
		}
	}
	return string(category)
}

// Принимает: ничего (получатель — категория).
// Делает: проверяет, что категория входит в закрытый набор домена.
// Возвращает: true, если категория известна, иначе false.
func (category Category) IsValid() bool {
	for _, option := range categoryOptions {
		if option.key == category {
			return true
		}
	}
	return false
}

// Принимает: ничего.
// Делает: собирает ключи категорий из каталога в новый срез, не раскрывая внутренний каталог.
// Возвращает: копию среза категорий в стабильном порядке.
func Categories() []Category {
	categories := make([]Category, len(categoryOptions))
	for index, option := range categoryOptions {
		categories[index] = option.key
	}
	return categories
}

// Принимает: value — строка с ключом категории.
// Делает: проверяет, что ключ принадлежит закрытому набору.
// Возвращает: Category либо ErrUnknownCategory.
func ParseCategory(value string) (Category, error) {
	category := Category(value)
	if !category.IsValid() {
		return "", ErrUnknownCategory
	}
	return category, nil
}
