//go:build benchmark

package main

import (
	"testing"
)

func testPool(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('a'+i%26)) + string(rune('A'+i/26))
	}
	return out
}

func TestPlanGroupsSingleParameter(t *testing.T) {
	pool := testPool(10)
	plan, err := PlanGroups(pool, 1, 4, 0.15)
	if err != nil {
		t.Fatal(err)
	}
	if plan.OverlapCountWant != 0 || plan.Stride != 1 {
		t.Fatalf("K=1 should have no overlap, got count=%d stride=%d", plan.OverlapCountWant, plan.Stride)
	}
	if plan.DistinctGroups != 4 {
		t.Fatalf("expected 4 distinct groups, got %d", plan.DistinctGroups)
	}
}

func TestPlanGroupsAdjacentOverlap(t *testing.T) {
	pool := testPool(139)
	plan, err := PlanGroups(pool, 20, 4, 0.15)
	if err != nil {
		t.Fatal(err)
	}
	if plan.OverlapCountWant != 3 {
		t.Fatalf("round(20*0.15)=3, got %d", plan.OverlapCountWant)
	}
	for g, shared := range plan.AchievedPerPair[:3] {
		if shared != 3 {
			t.Fatalf("pair %d shares %d, want 3", g, shared)
		}
	}
	if plan.AchievedPerPair[3] != 0 {
		t.Fatalf("wrap pair 3->0 must be reported as 0 shared when windows do not reach it, got %d", plan.AchievedPerPair[3])
	}
	for g, window := range plan.Groups {
		if len(window) != 20 {
			t.Fatalf("group %d has %d params", g, len(window))
		}
		seen := map[string]bool{}
		for _, p := range window {
			if seen[p] {
				t.Fatalf("group %d repeats %s", g, p)
			}
			seen[p] = true
		}
	}
	if plan.DistinctGroups != 4 {
		t.Fatalf("expected 4 groups as planned, got %d", plan.DistinctGroups)
	}
}

func TestPlanGroupsFullPoolIsUnavoidable(t *testing.T) {
	pool := testPool(5)
	plan, err := PlanGroups(pool, 5, 4, 0.15)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.FullPoolOverlapAt {
		t.Fatal("K=N must be flagged as full-pool overlap")
	}
	if plan.OverlapAchieved != 1 {
		t.Fatalf("K=N overlap should be 1, got %v", plan.OverlapAchieved)
	}
	if plan.DistinctGroups != 4 {
		t.Fatalf("K=N windows are rotations (distinct ordering), got %d", plan.DistinctGroups)
	}
	for g, shared := range plan.AchievedPerPair {
		if shared != plan.Size {
			t.Fatalf("K=N pair %d must share the full window, got %d", g, shared)
		}
	}
}

func TestPlanGroupsRejectsBadInput(t *testing.T) {
	pool := testPool(3)
	if _, err := PlanGroups(pool, 4, 2, 0.15); err == nil {
		t.Fatal("size beyond pool must fail")
	}
	if _, err := PlanGroups(pool, 0, 2, 0.15); err == nil {
		t.Fatal("size 0 must fail")
	}
	if _, err := PlanGroups(pool, 2, 0, 0.15); err == nil {
		t.Fatal("zero groups must fail")
	}
	if _, err := PlanGroups(pool, 2, 2, 1); err == nil {
		t.Fatal("overlap 1 must fail")
	}
}

func TestSumCompleteTicksTrimsEdgeOnly(t *testing.T) {
	got, err := SumCompleteTicks([][]int64{{1, 2, 3, 4}, {10, 20, 30}, {100, 200, 300, 400}}, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{111, 222, 333}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tick %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSumCompleteTicksRejectsIncomplete(t *testing.T) {
	if _, err := SumCompleteTicks([][]int64{{1, 2, 3, 4}, {10}}, 4); err == nil {
		t.Fatal("member counts spanning more than one tick must fail")
	}
	if _, err := SumCompleteTicks([][]int64{{1, 2}, {10, 20}}, 5); err == nil {
		t.Fatal("too few ticks for the window must fail as saturation")
	}
	if _, err := SumCompleteTicks(nil, 3); err == nil {
		t.Fatal("no members must fail")
	}
}

func TestComputeDistributionPercentiles(t *testing.T) {
	values := []int64{10, 20, 30, 40, 50}
	d := ComputeDistribution(values)
	if d.Count != 5 || d.Min != 10 || d.Max != 50 {
		t.Fatalf("count/min/max wrong: %+v", d)
	}
	if d.Median != 30 || d.Mean != 30 {
		t.Fatalf("median/mean wrong: %+v", d)
	}
	if d.P1 != 10.4 || d.P99 != 49.6 {
		t.Fatalf("interpolated p1/p99 wrong: p1=%v p99=%v", d.P1, d.P99)
	}
	if d.P30 != 22 || d.P70 != 38 {
		t.Fatalf("p30/p70 wrong: %v %v", d.P30, d.P70)
	}
	if empty := ComputeDistribution(nil); empty.Count != 0 {
		t.Fatal("empty distribution must have count 0")
	}
}

func TestEligibleParametersByCoverage(t *testing.T) {
	pool := []string{"a", "b", "c"}
	counts := map[string]int{"a": 5, "b": 1}
	eligible, unavailable := EligibleParameters(pool, counts, 2)
	if len(eligible) != 1 || eligible[0] != "a" {
		t.Fatalf("eligible = %v", eligible)
	}
	if len(unavailable) != 2 {
		t.Fatalf("unavailable = %v", unavailable)
	}
}

func TestSweepSizesResolvesTokens(t *testing.T) {
	got, err := SweepSizes("1, 4,max", 139)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 4, 139}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	all, err := SweepSizes("all", 5)
	if err != nil || len(all) != 5 {
		t.Fatalf("all: %v %v", all, err)
	}
	if _, err := SweepSizes("0", 5); err == nil {
		t.Fatal("size 0 must fail")
	}
	if _, err := SweepSizes("9", 5); err == nil {
		t.Fatal("size beyond pool must fail")
	}
	for _, invalid := range []string{"1junk", ",,", "4.5"} {
		if _, err := SweepSizes(invalid, 5); err == nil {
			t.Fatalf("invalid size %q must fail", invalid)
		}
	}
}
