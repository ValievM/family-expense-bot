// Package domain — значения расходов и их бизнес-валидация без зависимостей от адаптеров.
package domain

type Category string

// Ключи категорий неизменны: их хранит БД.
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

// categoryOptions — закрытый каталог; порядок определяет вывод Categories.
var categoryOptions = [...]categoryOption{
	{key: CategoryFood, label: "еда"},
	{key: CategoryTransport, label: "транспорт"},
	{key: CategoryHousing, label: "жильё"},
	{key: CategoryEntertainment, label: "развлечения"},
	{key: CategoryHealth, label: "здоровье"},
	{key: CategoryShopping, label: "покупки"},
	{key: CategoryOther, label: "другое"},
}

// Принимает: ничего.
// Делает: ищет русское название в каталоге.
// Возвращает: название; для неизвестной категории сам ключ.
func (category Category) Label() string {
	for _, option := range categoryOptions {
		if option.key == category {
			return option.label
		}
	}
	return string(category)
}

// Принимает: ничего.
// Делает: ищет категорию в закрытом наборе.
// Возвращает: true, если категория известна.
func (category Category) IsValid() bool {
	for _, option := range categoryOptions {
		if option.key == category {
			return true
		}
	}
	return false
}

// Принимает: ничего.
// Делает: копирует ключи каталога в новый срез.
// Возвращает: категории в стабильном порядке.
func Categories() []Category {
	categories := make([]Category, len(categoryOptions))
	for index, option := range categoryOptions {
		categories[index] = option.key
	}
	return categories
}

// Принимает: value — строка с ключом категории.
// Делает: проверяет, что ключ входит в закрытый набор.
// Возвращает: Category либо ErrUnknownCategory.
func ParseCategory(value string) (Category, error) {
	category := Category(value)
	if !category.IsValid() {
		return "", ErrUnknownCategory
	}
	return category, nil
}
