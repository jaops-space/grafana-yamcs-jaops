package tools

import "testing"

func TestConvertYamcsUnitToGrafanaUnit(t *testing.T) {
	tests := []struct {
		name string
		unit string
		want string
	}{
		{name: "empty", unit: "", want: ""},
		{name: "meter does not become minute", unit: "m", want: "lengthm"},
		{name: "minute remains Grafana time minute", unit: "min", want: "m"},
		{name: "milliwatt is case sensitive", unit: "mW", want: "mwatt"},
		{name: "megawatt is case sensitive", unit: "MW", want: "megwatt"},
		{name: "henry is case sensitive", unit: "H", want: "henry"},
		{name: "hour is lower-case time unit", unit: "h", want: "h"},
		{name: "volt", unit: "V", want: "volt"},
		{name: "meters per second", unit: "m/s", want: "velocityms"},
		{name: "meters per second squared with unicode exponent", unit: "m/s²", want: "accMS2"},
		{name: "celsius", unit: "°C", want: "celsius"},
		{name: "rpm", unit: "rpm", want: "rotrpm"},
		{name: "unknown custom unit is preserved", unit: "counts/frame", want: "counts/frame"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertYamcsUnitToGrafanaUnit(tt.unit); got != tt.want {
				t.Fatalf("ConvertYamcsUnitToGrafanaUnit(%q) = %q, want %q", tt.unit, got, tt.want)
			}
		})
	}
}
