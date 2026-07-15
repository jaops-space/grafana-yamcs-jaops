package tools

import (
	"math"
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrAlarmLevel(v mdb.AlarmLevelType) *mdb.AlarmLevelType {
	return &v
}

func TestConvertAlarmInfoToThresholds_HandlesInclusiveExclusiveAndMissingSides(t *testing.T) {
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_NORMAL),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{Level: ptrAlarmLevel(mdb.AlarmLevelType_WARNING), MaxInclusive: ptrFloat64(10)},
			{Level: ptrAlarmLevel(mdb.AlarmLevelType_CRITICAL), MaxExclusive: ptrFloat64(20)},
			{Level: ptrAlarmLevel(mdb.AlarmLevelType_DISTRESS), MinInclusive: ptrFloat64(30)},
			{Level: ptrAlarmLevel(mdb.AlarmLevelType_SEVERE), MinExclusive: ptrFloat64(40)},
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
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_WARNING),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_CRITICAL),
				MinInclusive: ptrFloat64(5),
				MinExclusive: ptrFloat64(5),
			},
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_DISTRESS),
				MaxInclusive: ptrFloat64(9),
				MaxExclusive: ptrFloat64(9),
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
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_NORMAL),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_WARNING),
				MinInclusive: ptrFloat64(1),
				MaxInclusive: ptrFloat64(5),
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
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_NORMAL),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_WARNING),
				MinInclusive: ptrFloat64(2),
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
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_NORMAL),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_CRITICAL),
				MinInclusive: ptrFloat64(9),
				MaxInclusive: ptrFloat64(15),
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
	alarmInfo := &mdb.AlarmInfo{
		DefaultLevel: ptrAlarmLevel(mdb.AlarmLevelType_NORMAL),
		StaticAlarmRanges: []*mdb.AlarmRange{
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_WARNING),
				MinInclusive: ptrFloat64(10),
				MaxInclusive: ptrFloat64(20),
			},
			{
				Level:        ptrAlarmLevel(mdb.AlarmLevelType_CRITICAL),
				MinInclusive: ptrFloat64(5),
				MaxInclusive: ptrFloat64(25),
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
