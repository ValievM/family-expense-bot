package domain

import "testing"

func TestCategoryLabelsAndValidity(t *testing.T) {
	tests := []struct {
		category Category
		label    string
	}{
		{category: CategoryFood, label: "еда"},
		{category: CategoryTransport, label: "транспорт"},
		{category: CategoryHousing, label: "жильё"},
		{category: CategoryEntertainment, label: "развлечения"},
		{category: CategoryHealth, label: "здоровье"},
		{category: CategoryShopping, label: "покупки"},
		{category: CategoryOther, label: "другое"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if !tt.category.IsValid() {
				t.Fatalf("%q is not a valid category", tt.category)
			}
			if got := tt.category.Label(); got != tt.label {
				t.Errorf("Label() = %q, want %q", got, tt.label)
			}
		})
	}

	if Category("travel").IsValid() {
		t.Error("unknown category is valid")
	}
}

func TestCategoriesReturnsStableOrderedCopy(t *testing.T) {
	want := []Category{
		CategoryFood,
		CategoryTransport,
		CategoryHousing,
		CategoryEntertainment,
		CategoryHealth,
		CategoryShopping,
		CategoryOther,
	}

	got := Categories()
	if len(got) != len(want) {
		t.Fatalf("Categories() returned %d values, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("Categories()[%d] = %q, want %q", index, got[index], want[index])
		}
	}

	got[0] = CategoryOther
	if next := Categories()[0]; next != CategoryFood {
		t.Errorf("mutating returned categories changed the catalog: first = %q", next)
	}
}
