package tools

import (
	"math"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

type Range struct {
	Value float64
	Level mdb.AlarmLevelType
}

type alarmRangeRule struct {
	Level          mdb.AlarmLevelType
	HasMin         bool
	MinValue       float64
	HasMax         bool
	MaxValue       float64
	InvertedBounds bool
}

var alarmSeverity = map[mdb.AlarmLevelType]int{
	mdb.AlarmLevelType_NORMAL:   0,
	mdb.AlarmLevelType_WATCH:    1,
	mdb.AlarmLevelType_WARNING:  2,
	mdb.AlarmLevelType_DISTRESS: 3,
	mdb.AlarmLevelType_CRITICAL: 4,
	mdb.AlarmLevelType_SEVERE:   5,
}

func parseAlarmRangeRule(alarmRange *mdb.AlarmRange) alarmRangeRule {
	rule := alarmRangeRule{Level: alarmRange.GetLevel()}

	if alarmRange.MinExclusive != nil {
		rule.HasMin = true
		rule.MinValue = *alarmRange.MinExclusive
	} else if alarmRange.MinInclusive != nil {
		rule.HasMin = true
		rule.MinValue = *alarmRange.MinInclusive
	}

	if alarmRange.MaxExclusive != nil {
		rule.HasMax = true
		rule.MaxValue = *alarmRange.MaxExclusive
	} else if alarmRange.MaxInclusive != nil {
		rule.HasMax = true
		rule.MaxValue = *alarmRange.MaxInclusive
	}

	rule.InvertedBounds = rule.HasMin && rule.HasMax

	return rule
}

func (r alarmRangeRule) matches(x float64) bool {
	inside := true

	if r.HasMin {
		inside = inside && x >= r.MinValue
	}

	if r.HasMax {
		inside = inside && x <= r.MaxValue
	}

	if r.InvertedBounds {
		return !inside
	}

	return inside
}

func (r alarmRangeRule) matchesAfterBoundary(x float64) bool {
	inside := true

	if r.HasMin {
		inside = inside && x >= r.MinValue
	}

	if r.HasMax {
		inside = inside && x < r.MaxValue
	}

	if r.InvertedBounds {
		return !inside
	}

	return inside
}

func highestAlarmLevel(levels []mdb.AlarmLevelType, fallback mdb.AlarmLevelType) mdb.AlarmLevelType {
	best := fallback
	bestRank := alarmSeverity[best]

	for _, level := range levels {
		rank, exists := alarmSeverity[level]
		if !exists {
			continue
		}
		if rank > bestRank {
			best = level
			bestRank = rank
		}
	}

	return best
}

func levelAtPoint(rules []alarmRangeRule, x float64, baseLevel mdb.AlarmLevelType) mdb.AlarmLevelType {
	active := make([]mdb.AlarmLevelType, 0, len(rules))
	for _, rule := range rules {
		if rule.matches(x) {
			active = append(active, rule.Level)
		}
	}

	return highestAlarmLevel(active, baseLevel)
}

func levelAfterBoundary(rules []alarmRangeRule, x float64, baseLevel mdb.AlarmLevelType) mdb.AlarmLevelType {
	active := make([]mdb.AlarmLevelType, 0, len(rules))
	for _, rule := range rules {
		if rule.matchesAfterBoundary(x) {
			active = append(active, rule.Level)
		}
	}

	return highestAlarmLevel(active, baseLevel)
}

func newThreshold(value float64, color string) *data.Threshold {
	threshold := data.NewThreshold(value, color, "")
	return &threshold
}

func rangeChangePoints(rules []alarmRangeRule) []float64 {
	points := make([]float64, 0, len(rules)*2)

	for _, rule := range rules {
		if rule.HasMin {
			points = append(points, rule.MinValue)
		}

		if rule.HasMax {
			points = append(points, rule.MaxValue)
		}
	}

	sort.Float64s(points)

	unique := make([]float64, 0, len(points))
	for _, p := range points {
		if len(unique) == 0 || p != unique[len(unique)-1] {
			unique = append(unique, p)
		}
	}

	return unique
}

func convertInvertedRangesToThresholds(alarmInfo *mdb.AlarmInfo) []*data.Threshold {
	rules := make([]alarmRangeRule, 0, len(alarmInfo.StaticAlarmRanges))
	for _, alarmRange := range alarmInfo.StaticAlarmRanges {
		rules = append(rules, parseAlarmRangeRule(alarmRange))
	}

	baseLevel := mdb.AlarmLevelType_NORMAL
	levels := []*data.Threshold{}

	current := levelAtPoint(rules, math.Inf(-1), baseLevel)
	firstColor, ok := AlarmLevelColors[current]
	if !ok {
		firstColor = "gray"
	}
	levels = append(levels, newThreshold(math.Inf(-1), firstColor))

	for _, point := range rangeChangePoints(rules) {
		lvl := levelAfterBoundary(rules, point, baseLevel)
		if lvl == current {
			continue
		}

		color, exists := AlarmLevelColors[lvl]
		if !exists {
			color = "gray"
		}
		levels = append(levels, newThreshold(point, color))
		current = lvl
	}

	return levels
}

func upperTransparentStart(alarmRange *mdb.AlarmRange) (float64, bool) {
	if alarmRange == nil {
		return 0, false
	}

	if alarmRange.MaxInclusive != nil {
		return *alarmRange.MaxInclusive, true
	}

	if alarmRange.MaxExclusive != nil {
		return *alarmRange.MaxExclusive, true
	}

	return 0, false
}

func normalizeLowerBoundary(alarmRange *mdb.AlarmRange) (float64, bool) {
	if alarmRange == nil {
		return 0, false
	}

	hasLower := false
	var lower float64

	if alarmRange.MinInclusive != nil {
		lower = *alarmRange.MinInclusive
		hasLower = true
	}

	if alarmRange.MinExclusive != nil {
		if !hasLower || *alarmRange.MinExclusive > lower {
			lower = *alarmRange.MinExclusive
			hasLower = true
		}
	}

	return lower, hasLower
}

func normalizeUpperBoundary(alarmRange *mdb.AlarmRange) (float64, bool) {
	if alarmRange == nil {
		return 0, false
	}

	hasUpper := false
	var upper float64

	if alarmRange.MaxInclusive != nil {
		upper = *alarmRange.MaxInclusive
		hasUpper = true
	}

	if alarmRange.MaxExclusive != nil {
		if !hasUpper || *alarmRange.MaxExclusive < upper {
			upper = *alarmRange.MaxExclusive
			hasUpper = true
		}
	}

	return upper, hasUpper
}

// AlarmLevelColors maps alarm levels to Grafana colors.
var AlarmLevelColors = map[mdb.AlarmLevelType]string{
	mdb.AlarmLevelType_NORMAL:   "green",
	mdb.AlarmLevelType_WATCH:    "cyan",
	mdb.AlarmLevelType_WARNING:  "yellow",
	mdb.AlarmLevelType_DISTRESS: "orange",
	mdb.AlarmLevelType_CRITICAL: "red",
	mdb.AlarmLevelType_SEVERE:   "darkred",
}

// ConvertAlarmInfoToGrafanaThresholds converts an AlarmInfo to Grafana thresholds.
func ConvertAlarmInfoToThresholds(alarmInfo *mdb.AlarmInfo) []*data.Threshold {

	if alarmInfo == nil {
		return nil
	}

	hasInvertedRange := false
	for _, alarmRange := range alarmInfo.StaticAlarmRanges {
		rule := parseAlarmRangeRule(alarmRange)
		if rule.InvertedBounds {
			hasInvertedRange = true
			break
		}
	}

	if hasInvertedRange {
		return convertInvertedRangesToThresholds(alarmInfo)
	}

	thresholds := []*data.Threshold{}
	ranges := []Range{}
	hasUnboundedLower := false
	hasUnboundedUpper := false
	hasUpperTailStart := false
	upperTailStart := 0.0

	// Collect all min/max values from alarm ranges
	for _, alarmRange := range alarmInfo.StaticAlarmRanges {
		lower, hasLower := normalizeLowerBoundary(alarmRange)
		if hasLower {
			ranges = append(ranges, Range{Value: lower, Level: alarmRange.GetLevel()})
		} else {
			hasUnboundedLower = true
		}

		if upper, hasUpper := normalizeUpperBoundary(alarmRange); hasUpper {
			ranges = append(ranges, Range{Value: upper, Level: alarmRange.GetLevel()})

			if tailStart, ok := upperTransparentStart(alarmRange); ok {
				if !hasUpperTailStart || tailStart > upperTailStart {
					upperTailStart = tailStart
					hasUpperTailStart = true
				}
			}
		} else {
			hasUnboundedUpper = true
		}
	}

	// Sort thresholds in ascending order
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Value < ranges[j].Value
	})

	// always set the base threshold at -Inf so Grafana has an explicit start step.
	baseColor := AlarmLevelColors[alarmInfo.GetDefaultLevel()]
	if baseColor == "" {
		baseColor = "gray"
	}
	if len(ranges) > 0 && !hasUnboundedLower {
		baseColor = "transparent"
	}
	thresholds = append(thresholds, newThreshold(math.Inf(-1), baseColor))

	// Convert to Grafana thresholds
	for _, r := range ranges {
		color, exists := AlarmLevelColors[r.Level]
		if !exists {
			color = "gray" // Default color for unknown levels
		}
		thresholds = append(thresholds, newThreshold(r.Value, color))
	}

	// add transparent upper tail if no alarm range covers +Infinity.
	if len(ranges) > 0 && !hasUnboundedUpper && hasUpperTailStart {
		thresholds = append(thresholds, newThreshold(upperTailStart, "transparent"))
	}

	return thresholds
}
