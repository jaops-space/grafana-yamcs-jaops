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

	thresholds := []*data.Threshold{}
	ranges := []Range{}

	// Collect all min/max values from alarm ranges
	for _, alarmRange := range alarmInfo.StaticAlarmRanges {
		if alarmRange.MinExclusive != nil {
			ranges = append(ranges, Range{Value: *alarmRange.MinExclusive, Level: alarmRange.GetLevel()})
		} else if alarmRange.MinInclusive != nil {
			ranges = append(ranges, Range{Value: *alarmRange.MinInclusive, Level: alarmRange.GetLevel()})
		}

		if alarmRange.MaxExclusive != nil {
			ranges = append(ranges, Range{Value: *alarmRange.MaxExclusive, Level: alarmRange.GetLevel()})
		} else if alarmRange.MaxInclusive != nil {
			ranges = append(ranges, Range{Value: *alarmRange.MaxInclusive, Level: alarmRange.GetLevel()})
		}
	}

	// Sort thresholds in ascending order
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Value < ranges[j].Value
	})

	// Handle inverse logic: If default level is NOT NORMAL, set the first threshold to -Infinity
	if alarmInfo.GetDefaultLevel() != mdb.AlarmLevelType_NORMAL {
		threshold := data.NewThreshold(math.Inf(-1), AlarmLevelColors[alarmInfo.GetDefaultLevel()], "")
		thresholds = append(thresholds, &threshold)
	}

	// Convert to Grafana thresholds
	for _, r := range ranges {
		color, exists := AlarmLevelColors[r.Level]
		if !exists {
			color = "gray" // Default color for unknown levels
		}
		threshold := data.NewThreshold(r.Value, color, "")
		thresholds = append(thresholds, &threshold)
	}

	return thresholds
}
