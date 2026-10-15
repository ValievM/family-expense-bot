package config

import "testing"

func TestLoadTimeZone(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantZone string
	}{
		{name: "default", value: "", wantZone: "Europe/Moscow"},
		{name: "configured", value: "UTC", wantZone: "UTC"},
		{name: "blank after trimming", value: "  ", wantZone: "Europe/Moscow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TZ", tt.value)

			if got := Load().TimeZone; got != tt.wantZone {
				t.Errorf("Load().TimeZone = %q, want %q", got, tt.wantZone)
			}
		})
	}
}
