package tools

import (
	"math"
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

func TestConvertAlarmInfoToThresholds_HandlesInclusiveExclusiveAndMissingSides(t *testing.T) {
	maxInclusive := new(10.0)
	maxExclusive := new(20.0)
	minInclusive := new(30.0)
	minExclusive := new(40.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_NORMAL.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{Level: mdb.AlarmLevelType_WARNING.Enum(), MaxInclusive: maxInclusive},
			{Level: mdb.AlarmLevelType_CRITICAL.Enum(), MaxExclusive: maxExclusive},
			{Level: mdb.AlarmLevelType_DISTRESS.Enum(), MinInclusive: minInclusive},
			{Level: mdb.AlarmLevelType_SEVERE.Enum(), MinExclusive: minExclusive},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 5 {
		t.Fatalf("expected 5 thresholds, got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) || thresholds[0].Color != "green" {
		t.Fatalf("expected base threshold at -Inf with normal color")
	}

	gotValues := []float64{
		float64(thresholds[1].Value),
		float64(thresholds[2].Value),
		float64(thresholds[3].Value),
		float64(thresholds[4].Value),
	}

	wantValues := []float64{
		10,
		math.Nextafter(20, math.Inf(-1)),
		30,
		math.Nextafter(40, math.Inf(1)),
	}

	for i := range wantValues {
		if gotValues[i] != wantValues[i] {
			t.Fatalf("threshold[%d] value mismatch: got %v want %v", i, gotValues[i], wantValues[i])
		}
	}
}

func TestConvertAlarmInfoToThresholds_PrefersStricterWhenBothBoundsExist(t *testing.T) {
	lowerInclusive := new(5.0)
	lowerExclusive := new(5.0)
	upperInclusive := new(9.0)
	upperExclusive := new(9.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_WARNING.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        mdb.AlarmLevelType_CRITICAL.Enum(),
				MinInclusive: lowerInclusive,
				MinExclusive: lowerExclusive,
			},
			{
				Level:        mdb.AlarmLevelType_DISTRESS.Enum(),
				MaxInclusive: upperInclusive,
				MaxExclusive: upperExclusive,
			},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 3 {
		t.Fatalf("expected 3 thresholds (including default level), got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) {
		t.Fatalf("expected first threshold at -Inf for non-normal default level")
	}

	upper := math.Nextafter(9, math.Inf(-1))
	lower := math.Nextafter(5, math.Inf(1))

	if float64(thresholds[1].Value) != lower {
		t.Fatalf("expected stricter lower bound %v, got %v", lower, float64(thresholds[1].Value))
	}

	if float64(thresholds[2].Value) != upper {
		t.Fatalf("expected stricter upper bound %v, got %v", upper, float64(thresholds[2].Value))
	}
}

func TestConvertAlarmInfoToThresholds_BoundedRangeUsesInvertedSemantics(t *testing.T) {
	minInclusive := new(1.0)
	maxInclusive := new(5.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_NORMAL.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        mdb.AlarmLevelType_WARNING.Enum(),
				MinInclusive: minInclusive,
				MaxInclusive: maxInclusive,
			},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 3 {
		t.Fatalf("expected 3 thresholds, got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) || thresholds[0].Color != "yellow" {
		t.Fatalf("expected warning at -Inf")
	}

	if float64(thresholds[1].Value) != 1 || thresholds[1].Color != "green" {
		t.Fatalf("expected normal start at 1")
	}

	wantUpperWarning := math.Nextafter(5, math.Inf(1))
	if float64(thresholds[2].Value) != wantUpperWarning || thresholds[2].Color != "yellow" {
		t.Fatalf("expected warning restart at %v, got value=%v color=%q", wantUpperWarning, float64(thresholds[2].Value), thresholds[2].Color)
	}
}

func TestConvertAlarmInfoToThresholds_NoUpperTransparentTailWhenUnboundedUpper(t *testing.T) {
	minInclusive := new(2.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_NORMAL.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        mdb.AlarmLevelType_WARNING.Enum(),
				MinInclusive: minInclusive,
			},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 2 {
		t.Fatalf("expected 2 thresholds, got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) || thresholds[0].Color != "transparent" {
		t.Fatalf("expected first transparent threshold at -Inf")
	}

	if float64(thresholds[1].Value) != 2 {
		t.Fatalf("expected warning start at 2, got %v", float64(thresholds[1].Value))
	}
}

func TestConvertAlarmInfoToThresholds_InvertedSingleRange(t *testing.T) {
	minInclusive := new(9.0)
	maxInclusive := new(15.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_NORMAL.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        mdb.AlarmLevelType_CRITICAL.Enum(),
				MinInclusive: minInclusive,
				MaxInclusive: maxInclusive,
			},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 3 {
		t.Fatalf("expected 3 thresholds, got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) || thresholds[0].Color != "red" {
		t.Fatalf("expected critical at -Inf")
	}

	if float64(thresholds[1].Value) != 9 || thresholds[1].Color != "green" {
		t.Fatalf("expected normal start at 9")
	}

	if float64(thresholds[2].Value) != math.Nextafter(15, math.Inf(1)) || thresholds[2].Color != "red" {
		t.Fatalf("expected critical restart just above 15")
	}
}

func TestConvertAlarmInfoToThresholds_MultipleInvertedRanges(t *testing.T) {
	warningMin := new(10.0)
	warningMax := new(20.0)
	criticalMin := new(5.0)
	criticalMax := new(25.0)

	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: mdb.AlarmLevelType_NORMAL.Enum(),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        mdb.AlarmLevelType_WARNING.Enum(),
				MinInclusive: warningMin,
				MaxInclusive: warningMax,
			},
			{
				Level:        mdb.AlarmLevelType_CRITICAL.Enum(),
				MinInclusive: criticalMin,
				MaxInclusive: criticalMax,
			},
		},
	}

	thresholds := ConvertAlarmInfoToThresholds(alarmInfo)
	if len(thresholds) != 5 {
		t.Fatalf("expected 5 thresholds, got %d", len(thresholds))
	}

	if float64(thresholds[0].Value) != math.Inf(-1) || thresholds[0].Color != "red" {
		t.Fatalf("expected critical at -Inf")
	}

	if float64(thresholds[1].Value) != 5 || thresholds[1].Color != "yellow" {
		t.Fatalf("expected warning at 5")
	}

	if float64(thresholds[2].Value) != 10 || thresholds[2].Color != "green" {
		t.Fatalf("expected normal at 10")
	}

	if float64(thresholds[3].Value) != math.Nextafter(20, math.Inf(1)) || thresholds[3].Color != "yellow" {
		t.Fatalf("expected warning after 20")
	}

	if float64(thresholds[4].Value) != math.Nextafter(25, math.Inf(1)) || thresholds[4].Color != "red" {
		t.Fatalf("expected critical after 25")
	}
}
