package types

import (
	"errors"
	"testing"

	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

func TestPaginatedRequestIterator_HasNextInitial(t *testing.T) {
	manager := &corehttp.HTTPManager{Query: map[string]string{}}
	iterator := NewPaginatedRequestIterator[int](manager, func() (int, string, error) {
		return 0, "", nil
	})

	if !iterator.HasNext() {
		t.Fatalf("expected HasNext to be true before first fetch")
	}
}

func TestPaginatedRequestIterator_NextAppliesQueryAndContinuation(t *testing.T) {
	manager := &corehttp.HTTPManager{Query: map[string]string{}}
	calls := 0

	fetch := func() (int, string, error) {
		calls++
		switch calls {
		case 1:
			if got := manager.Query["q"]; got != "abc" {
				t.Fatalf("first call: expected query q=abc, got %q", got)
			}
			if _, ok := manager.Query["next"]; ok {
				t.Fatalf("first call: did not expect continuation token")
			}
			return 11, "tok-1", nil
		case 2:
			if got := manager.Query["q"]; got != "abc" {
				t.Fatalf("second call: expected query q=abc, got %q", got)
			}
			if got := manager.Query["next"]; got != "tok-1" {
				t.Fatalf("second call: expected continuation next=tok-1, got %q", got)
			}
			return 22, "", nil
		default:
			t.Fatalf("unexpected additional fetch call: %d", calls)
			return 0, "", nil
		}
	}

	iterator := NewPaginatedRequestIterator(manager, fetch)
	iterator.SetQuery(map[string]string{"q": "abc"})

	first, err := iterator.Next()
	if err != nil {
		t.Fatalf("first Next failed: %v", err)
	}
	if first != 11 {
		t.Fatalf("first Next expected 11, got %d", first)
	}
	if !iterator.HasNext() {
		t.Fatalf("expected HasNext true after receiving continuation token")
	}

	second, err := iterator.Next()
	if err != nil {
		t.Fatalf("second Next failed: %v", err)
	}
	if second != 22 {
		t.Fatalf("second Next expected 22, got %d", second)
	}
	if iterator.HasNext() {
		t.Fatalf("expected HasNext false after empty continuation token")
	}
}

func TestPaginatedRequestIterator_ErrorClearsContinuation(t *testing.T) {
	manager := &corehttp.HTTPManager{Query: map[string]string{}}
	wantErr := errors.New("fetch failed")

	iterator := NewPaginatedRequestIterator[int](manager, func() (int, string, error) {
		return 0, "tok-will-be-cleared", wantErr
	})
	iterator.SetQuery(map[string]string{"q": "abc"})

	_, err := iterator.Next()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	if iterator.HasNext() {
		t.Fatalf("expected HasNext false after fetch error clears continuation")
	}

	if got := manager.Query["q"]; got != "abc" {
		t.Fatalf("expected query to be set before fetch, got q=%q", got)
	}
}

func TestPaginatedRequestIterator_SetQueryOverridesExistingKey(t *testing.T) {
	manager := &corehttp.HTTPManager{Query: map[string]string{}}

	iterator := NewPaginatedRequestIterator[int](manager, func() (int, string, error) {
		if got := manager.Query["q"]; got != "new" {
			t.Fatalf("expected latest query value q=new, got %q", got)
		}
		if got := manager.Query["limit"]; got != "100" {
			t.Fatalf("expected secondary query value limit=100, got %q", got)
		}
		return 1, "", nil
	})

	iterator.SetQuery(map[string]string{"q": "old"})
	iterator.SetQuery(map[string]string{"q": "new", "limit": "100"})

	_, err := iterator.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
}
