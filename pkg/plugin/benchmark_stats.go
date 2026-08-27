//go:build benchmark

package plugin

import (
	"encoding/json"
	"net/http"
	"runtime"
	"slices"
	"sync"
	"time"
)

type benchmarkStatsSnapshot struct {
	RunStreamRuntimeNS          int64   `json:"run_stream_runtime_ns"`
	RunStreamMedianRuntimeNS    int64   `json:"run_stream_median_runtime_ns"`
	RunStreamSamples            int     `json:"run_stream_samples"`
	RunStreamCalls              int64   `json:"run_stream_calls"`
	FramesSent                  int64   `json:"frames_sent"`
	ValuesSent                  int64   `json:"values_sent"`
	UniqueStreamPaths           int     `json:"unique_stream_paths"`
	WindowSeconds               float64 `json:"window_seconds"`
	BackendAllocBytes           uint64  `json:"backend_alloc_bytes"`
	BackendHeapAllocBytes       uint64  `json:"backend_heap_alloc_bytes"`
	BackendHeapAllocGrowthBytes uint64  `json:"backend_heap_alloc_growth_bytes"`
	BackendHeapInuseBytes       uint64  `json:"backend_heap_inuse_bytes"`
	BackendHeapObjects          uint64  `json:"backend_heap_objects"`
	BackendSysBytes             uint64  `json:"backend_sys_bytes"`
}

type benchmarkStatsCollector struct {
	mu                     sync.Mutex
	startedAt              time.Time
	paths                  map[string]struct{}
	samples                []int64
	targetSamples          int
	runtimeNS              int64
	runCalls               int64
	framesSent             int64
	valuesSent             int64
	baselineHeapAllocBytes uint64
}

var streamBenchmarkStats = newBenchmarkStatsCollector()

func newBenchmarkStatsCollector() *benchmarkStatsCollector {
	return &benchmarkStatsCollector{
		paths: map[string]struct{}{},
	}
}

func (stats *benchmarkStatsCollector) reset(targetSamples int) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.startedAt = time.Now()
	stats.paths = map[string]struct{}{}
	stats.samples = nil
	stats.targetSamples = targetSamples
	stats.runtimeNS = 0
	stats.runCalls = 0
	stats.framesSent = 0
	stats.valuesSent = 0
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	stats.baselineHeapAllocBytes = mem.HeapAlloc
}

func (stats *benchmarkStatsCollector) recordRunStream(path string) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.runCalls++
	stats.paths[path] = struct{}{}
	if stats.startedAt.IsZero() {
		stats.startedAt = time.Now()
	}
}

func (stats *benchmarkStatsCollector) recordRunStreamWork(path string, elapsed time.Duration, values int) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	if stats.targetSamples > 0 && len(stats.samples) >= stats.targetSamples {
		return
	}
	stats.paths[path] = struct{}{}
	stats.runtimeNS += elapsed.Nanoseconds()
	stats.framesSent++
	stats.valuesSent += int64(values)
	stats.samples = append(stats.samples, elapsed.Nanoseconds())
	if stats.startedAt.IsZero() {
		stats.startedAt = time.Now()
	}
}

func (stats *benchmarkStatsCollector) snapshot() benchmarkStatsSnapshot {
	stats.mu.Lock()
	startedAt := stats.startedAt
	pathCount := len(stats.paths)
	samples := slices.Clone(stats.samples)
	runtimeNS := stats.runtimeNS
	runCalls := stats.runCalls
	framesSent := stats.framesSent
	valuesSent := stats.valuesSent
	baselineHeapAllocBytes := stats.baselineHeapAllocBytes
	stats.mu.Unlock()
	windowSeconds := 0.0
	if !startedAt.IsZero() {
		windowSeconds = time.Since(startedAt).Seconds()
	}
	medianRuntimeNS := int64(0)
	if len(samples) > 0 {
		slices.Sort(samples)
		mid := len(samples) / 2
		if len(samples)%2 == 0 {
			medianRuntimeNS = (samples[mid-1] + samples[mid]) / 2
		} else {
			medianRuntimeNS = samples[mid]
		}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	heapAllocGrowthBytes := uint64(0)
	if mem.HeapAlloc > baselineHeapAllocBytes {
		heapAllocGrowthBytes = mem.HeapAlloc - baselineHeapAllocBytes
	}
	return benchmarkStatsSnapshot{
		RunStreamRuntimeNS:          runtimeNS,
		RunStreamMedianRuntimeNS:    medianRuntimeNS,
		RunStreamSamples:            len(samples),
		RunStreamCalls:              runCalls,
		FramesSent:                  framesSent,
		ValuesSent:                  valuesSent,
		UniqueStreamPaths:           pathCount,
		WindowSeconds:               windowSeconds,
		BackendAllocBytes:           mem.Alloc,
		BackendHeapAllocBytes:       mem.HeapAlloc,
		BackendHeapAllocGrowthBytes: heapAllocGrowthBytes,
		BackendHeapInuseBytes:       mem.HeapInuse,
		BackendHeapObjects:          mem.HeapObjects,
		BackendSysBytes:             mem.Sys,
	}
}

func (d *Datasource) handleBenchmarkReset(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		TargetSamples int `json:"target_samples"`
	}
	if req.Body != nil {
		_ = json.NewDecoder(req.Body).Decode(&payload)
	}
	streamBenchmarkStats.reset(payload.TargetSamples)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(streamBenchmarkStats.snapshot())
}

func (d *Datasource) handleBenchmarkStats(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(streamBenchmarkStats.snapshot())
}
