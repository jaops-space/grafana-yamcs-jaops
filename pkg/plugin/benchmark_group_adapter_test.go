//go:build benchmark

package plugin

import (
	"testing"
	"time"
)

func TestGroupSampleCollectionIsOptIn(t *testing.T) {
	stats := newBenchmarkStatsCollector()
	stats.reset(0)
	stats.recordRunStreamWork("one", time.Microsecond, 1)
	if len(stats.samplesByPath()) != 0 || len(stats.samples) != 1 {
		t.Fatal("ordinary benchmark sampling must not duplicate per-path samples")
	}
	stats.collectPathSamples = true
	stats.recordRunStreamWork("one", 2*time.Microsecond, 1)
	snapshot := stats.samplesByPath()
	if len(snapshot["one"]) != 1 || snapshot["one"][0] != 2000 {
		t.Fatalf("wrong per-path spans: %v", snapshot)
	}
	snapshot["one"][0] = 123
	if stats.samplesByPath()["one"][0] != 2000 {
		t.Fatal("snapshot must not alias the collector")
	}
	stats.reset(0)
	stats.recordRunStreamWork("one", time.Microsecond, 1)
	if len(stats.samplesByPath()) != 0 {
		t.Fatal("reset must disable group-only collection")
	}
}
