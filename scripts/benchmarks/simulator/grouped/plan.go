//go:build benchmark

package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// GroupPlan is the set of logical client groups for one group size K.
// Group g is the circular window pool[(g*stride + j) mod N] for j in [0, K).
type GroupPlan struct {
	Size              int
	Groups            [][]string
	OverlapTarget     float64
	OverlapCountWant  int
	Stride            int
	OverlapAchieved   float64
	AchievedPerPair   []int
	DistinctGroups    int
	FullPoolOverlapAt bool
}

// PlanGroups builds `groups` logical client groups of `size` parameters from pool,
// with adjacent groups sharing round(size*overlap) parameters where the circular
// window arithmetic allows it. Large sizes cannot always hit the target: when the
// window covers the whole pool every group is identical, and that overlap is
// reported as achieved rather than hidden.
func PlanGroups(pool []string, size, groups int, overlap float64) (GroupPlan, error) {
	n := len(pool)
	if size < 1 || size > n {
		return GroupPlan{}, fmt.Errorf("group size %d outside pool of %d parameters", size, n)
	}
	if groups < 1 {
		return GroupPlan{}, errors.New("groups must be at least 1")
	}
	if math.IsNaN(overlap) || overlap < 0 || overlap >= 1 {
		return GroupPlan{}, fmt.Errorf("overlap %.3f outside [0,1)", overlap)
	}
	wantShared := int(math.Round(float64(size) * overlap))
	stride := size - wantShared
	if stride < 1 {
		stride = 1
	}

	plan := GroupPlan{
		Size:             size,
		OverlapTarget:    overlap,
		OverlapCountWant: wantShared,
		Stride:           stride,
		Groups:           make([][]string, groups),
	}
	for g := 0; g < groups; g++ {
		window := make([]string, size)
		for j := 0; j < size; j++ {
			window[j] = pool[(g*stride+j)%n]
		}
		plan.Groups[g] = window
	}

	distinct := map[string]struct{}{}
	for _, window := range plan.Groups {
		distinct[fmt.Sprint(window)] = struct{}{}
	}
	plan.DistinctGroups = len(distinct)

	shareSum := 0.0
	for g := 0; g < groups; g++ {
		shared := sharedCount(plan.Groups[g], plan.Groups[(g+1)%groups])
		if groups == 1 {
			shared = size
		}
		plan.AchievedPerPair = append(plan.AchievedPerPair, shared)
		shareSum += float64(shared) / float64(size)
	}
	plan.OverlapAchieved = shareSum / float64(groups)
	plan.FullPoolOverlapAt = size == n
	return plan, nil
}

func sharedCount(a, b []string) int {
	set := make(map[string]struct{}, len(a))
	for _, p := range a {
		set[p] = struct{}{}
	}
	shared := 0
	for _, p := range b {
		if _, ok := set[p]; ok {
			shared++
		}
	}
	return shared
}

// SumCompleteTicks normalizes a split (one stream per parameter) group into one
// per-group-tick series. Member streams start at different phases, so each may
// record one edge tick more or fewer than expected per window; those edge ticks
// are trimmed. A member that is missing more than one tick, or whose count
// disagrees with the others by more than one, is an incomplete observation and
// returns an error instead of a partial sum that would undercount the group.
func SumCompleteTicks(members [][]int64, expected int) ([]int64, error) {
	if len(members) == 0 {
		return nil, errors.New("split group has no member streams")
	}
	lo, hi := len(members[0]), len(members[0])
	for _, m := range members[1:] {
		lo = min(lo, len(m))
		hi = max(hi, len(m))
	}
	if hi-lo > 1 {
		return nil, fmt.Errorf("incomplete split observation: member tick counts span %d..%d", lo, hi)
	}
	if lo < expected-1 {
		return nil, fmt.Errorf("incomplete split observation: %d ticks recorded, expected about %d (saturation or missing ticks)", lo, expected)
	}
	out := make([]int64, lo)
	for i := 0; i < lo; i++ {
		var sum int64
		for _, m := range members {
			sum += m[i]
		}
		out[i] = sum
	}
	return out, nil
}

// Distribution summarizes nanosecond samples. Percentiles use linear interpolation
// between closest ranks (the NumPy default).
type Distribution struct {
	Count  int     `json:"count"`
	Median float64 `json:"median"`
	Mean   float64 `json:"mean"`
	P1     float64 `json:"p1"`
	P5     float64 `json:"p5"`
	P30    float64 `json:"p30"`
	P70    float64 `json:"p70"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

func ComputeDistribution(values []int64) Distribution {
	if len(values) == 0 {
		return Distribution{}
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var sum float64
	for _, v := range sorted {
		sum += float64(v)
	}
	return Distribution{
		Count:  len(sorted),
		Median: percentile(sorted, 50),
		Mean:   sum / float64(len(sorted)),
		P1:     percentile(sorted, 1),
		P5:     percentile(sorted, 5),
		P30:    percentile(sorted, 30),
		P70:    percentile(sorted, 70),
		P95:    percentile(sorted, 95),
		P99:    percentile(sorted, 99),
		Min:    float64(sorted[0]),
		Max:    float64(sorted[len(sorted)-1]),
	}
}

func percentile(sorted []int64, p float64) float64 {
	if len(sorted) == 1 {
		return float64(sorted[0])
	}
	rank := p / 100 * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	frac := rank - float64(lo)
	return float64(sorted[lo]) + frac*(float64(sorted[hi])-float64(sorted[lo]))
}

// EligibleParameters splits the candidate pool by live coverage: a parameter is
// eligible when the probe window drained at least minValues engineering values for it.
func EligibleParameters(pool []string, counts map[string]int, minValues int) (eligible, unavailable []string) {
	for _, p := range pool {
		if counts[p] >= minValues {
			eligible = append(eligible, p)
		} else {
			unavailable = append(unavailable, p)
		}
	}
	return eligible, unavailable
}

// SweepSizes resolves a size token list such as "1,4,max" or "all" against the pool size.
func SweepSizes(spec string, poolSize int) ([]int, error) {
	if poolSize < 1 {
		return nil, errors.New("parameter pool must not be empty")
	}
	if spec == "" || spec == "all" {
		out := make([]int, 0, poolSize)
		for k := 1; k <= poolSize; k++ {
			out = append(out, k)
		}
		return out, nil
	}
	seen := map[int]bool{}
	var out []int
	for _, token := range splitComma(spec) {
		k := 0
		if token == "max" {
			k = poolSize
		} else {
			var err error
			k, err = strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid size %q", token)
			}
		}
		if k < 1 || k > poolSize {
			return nil, fmt.Errorf("size %d outside 1..%d", k, poolSize)
		}
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Ints(out)
	if len(out) == 0 {
		return nil, errors.New("sizes must contain at least one group size")
	}
	return out, nil
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if t := trimSpace(s[start:i]); t != "" {
				out = append(out, t)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
