package plugin

import (
	"testing"
)

func TestParameterPathValueFormatsAggregateAndArrayMembers(t *testing.T) {
	got := parameterPathValue("/drone/Motors", []string{"[0]", "rpm"})
	want := "/drone/Motors[0].rpm"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
