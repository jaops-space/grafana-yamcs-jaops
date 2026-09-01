package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

type scenarioMetric struct {
	Streams                 int               `json:"streams"`
	SetupDuration           int64             `json:"setup"`
	ProcessEvents           int64             `json:"process_events"`
	MedianProcessDuration   float64           `json:"avg_process"`
	ReadClearOperations     int64             `json:"read_clear_operations"`
	EmptyReadOperations     int64             `json:"empty_read_operations"`
	NonEmptyReadOperations  int64             `json:"non_empty_read_operations"`
	EmptyReadPercent        float64           `json:"empty_read_pct"`
	MedianReadClearDuration float64           `json:"avg_read_clear"`
	ValuesRead              int64             `json:"values_read"`
	ValuesReadPerSecond     float64           `json:"values_read_per_sec"`
	ValuesReadFresh         int64             `json:"values_read_fresh"`
	ValuesReadStale         int64             `json:"values_read_stale"`
	ValuesReadFreshPercent  float64           `json:"values_read_fresh_pct"`
	MedianTickRunStream     float64           `json:"avg_tick_runstream"`
	MedianTickRunStreamBusy float64           `json:"median_tick_runstream_busy"`
	TickRunStreamBusy       distributionStats `json:"median_tick_runstream_busy_distribution"`
	LiveMemoryGrowthBytes   int64             `json:"live_memory_growth_bytes"`
	TotalAllocatedBytes     uint64            `json:"total_allocated_bytes"`
}

type scenarioResult struct {
	StartedAt       string           `json:"started_at"`
	FinishedAt      string           `json:"finished_at"`
	YamcsAddress    string           `json:"yamcs_address"`
	Instance        string           `json:"instance"`
	Processor       string           `json:"processor"`
	System          systemInfo       `json:"system"`
	DurationSeconds float64          `json:"duration_seconds"`
	WarmupSeconds   float64          `json:"warmup_seconds"`
	WarmupScenario  *warmupScenario  `json:"warmup_scenario,omitempty"`
	ReadIntervalMS  int              `json:"read_interval_ms"`
	FreshnessMS     int              `json:"freshness_ms"`
	Parameters      []string         `json:"parameters"`
	Scenarios       []scenarioMetric `json:"scenarios"`
}

type warmupScenario struct {
	Streams         int     `json:"streams"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type systemInfo struct {
	OS                   string  `json:"os"`
	Arch                 string  `json:"arch"`
	CPUs                 int     `json:"cpus"`
	AvailableLogicalCPUs int     `json:"available_logical_cpus"`
	CPUModel             string  `json:"cpu_model,omitempty"`
	CPUFrequencyMHz      float64 `json:"cpu_frequency_mhz,omitempty"`
	GoVersion            string  `json:"go_version"`
}

type streamRequest struct {
	parameter string
	path      string
}

func main() {
	address := flag.String("address", "localhost:8090", "Yamcs host:port")
	instance := flag.String("instance", "myproject", "Yamcs instance")
	processor := flag.String("processor", "realtime", "Yamcs processor")
	streamsArg := flag.String("streams", "1,5,10,25,50,100", "comma-separated Grafana stream counts")
	parametersArg := flag.String("parameters", "/myproject/Battery1_Voltage,/myproject/Battery2_Voltage,/myproject/Battery1_Temp,/myproject/Battery2_Temp,/myproject/Detector_Temp", "comma-separated Yamcs parameter names")
	duration := flag.Duration("duration", 10*time.Second, "measurement duration for each scenario")
	warmup := flag.Duration("warmup", 3*time.Second, "warmup duration before measuring each scenario")
	warmupScenarioStreams := flag.Int("warmup-scenario-streams", 25, "number of streams for one unmeasured warmup scenario before measured scenarios; 0 disables it")
	warmupScenarioDuration := flag.Duration("warmup-scenario-duration", 3*time.Second, "duration for the unmeasured warmup scenario")
	readInterval := flag.Duration("read-interval", time.Second, "interval between read-and-clear operations per Grafana stream")
	freshnessWindow := flag.Duration("freshness-window", time.Second, "maximum delay counted as read in the same telemetry cycle")
	flag.Parse()

	streamCounts, err := parsePositiveInts(*streamsArg)
	if err != nil {
		exitf("invalid --streams: %v", err)
	}
	parameters := parseList(*parametersArg)
	if len(parameters) == 0 {
		exitf("--parameters must include at least one parameter")
	}

	result := scenarioResult{
		StartedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		YamcsAddress:    *address,
		Instance:        *instance,
		Processor:       *processor,
		System:          collectSystemInfo(),
		DurationSeconds: duration.Seconds(),
		WarmupSeconds:   warmup.Seconds(),
		ReadIntervalMS:  int(readInterval.Milliseconds()),
		FreshnessMS:     int(freshnessWindow.Milliseconds()),
		Parameters:      parameters,
		Scenarios:       make([]scenarioMetric, 0, len(streamCounts)),
	}

	if *warmupScenarioStreams > 0 && *warmupScenarioDuration > 0 {
		_, err := runScenario(*address, *instance, *processor, parameters, *warmupScenarioStreams, *warmupScenarioDuration, *warmup, *readInterval, *freshnessWindow)
		if err != nil {
			exitf("warmup scenario failed: %v", err)
		}
		result.WarmupScenario = &warmupScenario{
			Streams:         *warmupScenarioStreams,
			DurationSeconds: warmupScenarioDuration.Seconds(),
		}
	}

	for _, streams := range streamCounts {
		metric, err := runScenario(*address, *instance, *processor, parameters, streams, *duration, *warmup, *readInterval, *freshnessWindow)
		if err != nil {
			exitf("scenario streams=%d failed: %v", streams, err)
		}
		result.Scenarios = append(result.Scenarios, metric)
	}

	result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		exitf("could not encode results: %v", err)
	}
}

func collectSystemInfo() systemInfo {
	model, frequencyMHz := readLinuxCPUInfo()
	return systemInfo{
		OS:                   runtime.GOOS,
		Arch:                 runtime.GOARCH,
		CPUs:                 runtime.NumCPU(),
		AvailableLogicalCPUs: runtime.NumCPU(),
		CPUModel:             model,
		CPUFrequencyMHz:      frequencyMHz,
		GoVersion:            runtime.Version(),
	}
}

func readLinuxCPUInfo() (string, float64) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", 0
	}
	defer file.Close()

	var model string
	var frequencyMHz float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if model == "" && (key == "model name" || key == "Hardware" || key == "Processor") {
			model = value
		}
		if frequencyMHz == 0 && key == "cpu MHz" {
			if _, err := fmt.Sscanf(value, "%f", &frequencyMHz); err != nil {
				frequencyMHz = 0
			}
		}
		if model != "" && frequencyMHz != 0 {
			break
		}
	}
	return model, frequencyMHz
}

func runScenario(address string, instance string, processor string, parameters []string, streams int, duration time.Duration, warmup time.Duration, readInterval time.Duration, freshnessWindow time.Duration) (scenarioMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), warmup+duration+30*time.Second)
	defer cancel()

	mux, err := source.NewMultiplexer(&config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"quickstart": {
				ID:       "quickstart",
				Name:     "Yamcs Quickstart",
				Path:     address,
				Protobuf: true,
			},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			"quickstart": {
				ID:        "quickstart",
				Name:      "Yamcs Quickstart",
				Host:      "quickstart",
				Instance:  instance,
				Processor: processor,
			},
		},
	}, &config.YamcsSecureConfiguration{Hosts: map[string]*config.YamcsSecureHost{}})
	if err != nil {
		return scenarioMetric{}, err
	}
	defer mux.Dispose()

	hostErrors, endpointErrors := mux.ConnectSync(ctx, true)
	if len(hostErrors) > 0 || len(endpointErrors) > 0 {
		return scenarioMetric{}, fmt.Errorf("connect hostErrors=%v endpointErrors=%v", hostErrors, endpointErrors)
	}
	endpoint, err := mux.GetEndpoint("quickstart")
	if err != nil {
		return scenarioMetric{}, err
	}

	var processEvents atomic.Int64
	processDurations := &durationRecorder{}
	arrivals := newArrivalTracker()
	tickWork := newTickWorkload(readInterval)
	var scenarioStarted time.Time
	endpoint.ParameterProcessObserver = func(_ string, _ int, elapsed time.Duration) {
		processEvents.Add(1)
		processDurations.add(elapsed.Nanoseconds())
	}
	endpoint.ParameterArrivalObserver = func(parameter string, path string, receivedAt time.Time) {
		arrivals.record(parameter, path, receivedAt)
	}

	requests := make([]streamRequest, 0, streams)
	setupStarted := time.Now()
	for i := 0; i < streams; i++ {
		parameter := parameters[i%len(parameters)]
		path := fmt.Sprintf("benchmark/scenario/streams-%d/stream-%d", streams, i)
		if err := endpoint.RequestNewParameterStream(ctx, parameter, path); err != nil {
			return scenarioMetric{}, err
		}
		requests = append(requests, streamRequest{parameter: parameter, path: path})
	}
	setupNanos := time.Since(setupStarted).Nanoseconds()

	time.Sleep(warmup)
	for _, req := range requests {
		endpoint.DrainParameterStream(req.parameter, req.path)
	}
	arrivals.clear()
	processEvents.Store(0)
	processDurations = &durationRecorder{}
	runtime.GC()

	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)

	var valuesRead atomic.Int64
	var freshValues atomic.Int64
	var readOps atomic.Int64
	var emptyReads atomic.Int64
	var nonEmptyReads atomic.Int64
	readDurations := &durationRecorder{}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(requests))
	for _, req := range requests {
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(readInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					startOffset := time.Duration(0)
					if !scenarioStarted.IsZero() {
						startOffset = time.Since(scenarioStarted)
					}
					started := time.Now()
					values := endpoint.DrainParameterStream(req.parameter, req.path)
					readAt := time.Now()
					receivedAtValues := arrivals.pop(req.parameter, req.path, len(values))
					for _, receivedAt := range receivedAtValues {
						age := readAt.Sub(receivedAt)
						if age < 0 {
							age = 0
						}
						if age <= freshnessWindow {
							freshValues.Add(1)
						}
					}
					if len(values) > 0 {
						if len(values) > 3 {
							frame := tools.ConvertBufferToAverageFrame(values, req.parameter, false, false, false)
							runtime.KeepAlive(frame)
						} else {
							frame := tools.ConvertBufferToFrame(values, req.parameter, false, false, false)
							runtime.KeepAlive(frame)
						}
					}
					readSendElapsed := time.Since(started)
					readDurations.add(readSendElapsed.Nanoseconds())
					tickWork.addReadSendSpan(startOffset, startOffset+readSendElapsed)
					readOps.Add(1)
					if len(values) == 0 {
						emptyReads.Add(1)
					} else {
						nonEmptyReads.Add(1)
					}
					valuesRead.Add(int64(len(values)))
				case <-stop:
					return
				}
			}
		}()
	}

	scenarioStarted = time.Now()
	time.Sleep(duration)
	close(stop)
	wg.Wait()

	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)

	for _, req := range requests {
		if err := endpoint.WithdrawParameterStreamRequest(context.Background(), req.parameter, req.path); err != nil {
			return scenarioMetric{}, err
		}
	}

	readCount := readOps.Load()
	emptyReadCount := emptyReads.Load()
	nonEmptyReadCount := nonEmptyReads.Load()
	processCount := processEvents.Load()
	valueCount := valuesRead.Load()

	metric := scenarioMetric{
		Streams:                streams,
		SetupDuration:          setupNanos,
		ProcessEvents:          processCount,
		ReadClearOperations:    readCount,
		EmptyReadOperations:    emptyReadCount,
		NonEmptyReadOperations: nonEmptyReadCount,
		ValuesRead:             valueCount,
		ValuesReadPerSecond:    float64(valueCount) / duration.Seconds(),
		LiveMemoryGrowthBytes:  int64(memEnd.Alloc) - int64(memStart.Alloc),
		TotalAllocatedBytes:    memEnd.TotalAlloc - memStart.TotalAlloc,
	}
	if readCount > 0 {
		metric.MedianReadClearDuration = readDurations.median()
		metric.EmptyReadPercent = 100 * float64(emptyReadCount) / float64(readCount)
	}
	freshCount := freshValues.Load()
	metric.ValuesReadFresh = freshCount
	metric.ValuesReadStale = valueCount - freshCount
	if valueCount > 0 {
		metric.ValuesReadFreshPercent = 100 * float64(freshCount) / float64(valueCount)
	}
	if processCount > 0 {
		metric.MedianProcessDuration = processDurations.median()
	}
	tickSummary := tickWork.summary()
	metric.MedianTickRunStream = tickSummary.MedianTotalNanos
	metric.MedianTickRunStreamBusy = tickSummary.Busy.Median
	metric.TickRunStreamBusy = tickSummary.Busy
	return metric, nil
}

type tickWorkload struct {
	mu           sync.Mutex
	interval     time.Duration
	readSendSpan map[int]tickSpan
	busyNanos    map[int]int64
	highestIndex int
}

type tickSpan struct {
	startNanos int64
	endNanos   int64
	seen       bool
}

// distributionStats bundles a full statistical summary of a set of
// measurements (median/min/max plus a percentile spread) so metrics that
// want distribution data declare a single field instead of one field per
// statistic. Serializes as a nested JSON object, e.g.
// "median_tick_runstream_busy_distribution": {"median": ..., "p95": ...}.
type distributionStats struct {
	Median float64 `json:"median"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	P1     float64 `json:"p1"`
	P5     float64 `json:"p5"`
	P30    float64 `json:"p30"`
	P70    float64 `json:"p70"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

func computeDistribution(values []int64) distributionStats {
	if len(values) == 0 {
		return distributionStats{}
	}
	return distributionStats{
		Median: medianInt64(values),
		Min:    minInt64(values),
		Max:    maxInt64(values),
		P1:     percentileInt64(values, 0.01),
		P5:     percentileInt64(values, 0.05),
		P30:    percentileInt64(values, 0.30),
		P70:    percentileInt64(values, 0.70),
		P95:    percentileInt64(values, 0.95),
		P99:    percentileInt64(values, 0.99),
	}
}

type tickWorkloadSummary struct {
	MedianTotalNanos float64
	Busy             distributionStats
}

type durationRecorder struct {
	mu     sync.Mutex
	values []int64
}

func (recorder *durationRecorder) add(value int64) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.values = append(recorder.values, value)
}

func (recorder *durationRecorder) median() float64 {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return medianInt64(recorder.values)
}

func newTickWorkload(interval time.Duration) *tickWorkload {
	if interval <= 0 {
		interval = time.Second
	}
	return &tickWorkload{
		interval:     interval,
		readSendSpan: map[int]tickSpan{},
		busyNanos:    map[int]int64{},
	}
}

func (workload *tickWorkload) addReadSendSpan(startOffset time.Duration, endOffset time.Duration) {
	if startOffset < 0 {
		startOffset = 0
	}
	if endOffset < startOffset {
		endOffset = startOffset
	}
	index := int(startOffset / workload.interval)
	startNanos := startOffset.Nanoseconds()
	endNanos := endOffset.Nanoseconds()

	workload.mu.Lock()
	defer workload.mu.Unlock()

	span := workload.readSendSpan[index]
	if !span.seen || startNanos < span.startNanos {
		span.startNanos = startNanos
	}
	if !span.seen || endNanos > span.endNanos {
		span.endNanos = endNanos
	}
	span.seen = true
	workload.readSendSpan[index] = span

	if index > workload.highestIndex {
		workload.highestIndex = index
	}
	workload.busyNanos[index] += endNanos - startNanos
}

func (workload *tickWorkload) summary() tickWorkloadSummary {
	workload.mu.Lock()
	defer workload.mu.Unlock()
	tickCount := workload.runStreamTickCount()
	if tickCount <= 0 {
		return tickWorkloadSummary{}
	}

	totals := make([]int64, 0, tickCount)
	busyTotals := make([]int64, 0, tickCount)
	for i := 0; i < tickCount; i++ {
		span, seen := workload.readSendSpan[i]
		if !seen || !span.seen {
			// No stream reported activity for this tick window at all - it
			// was never observed, not genuinely a zero-duration busy tick.
			// Including it as a literal 0 would fabricate a false minimum
			// (and a false trough in the density band), so skip it.
			continue
		}
		totals = append(totals, span.endNanos-span.startNanos)
		busyTotals = append(busyTotals, workload.busyNanos[i])
	}
	if len(busyTotals) == 0 {
		return tickWorkloadSummary{}
	}

	return tickWorkloadSummary{
		MedianTotalNanos: medianInt64(totals),
		Busy:             computeDistribution(busyTotals),
	}
}

// percentileInt64 returns the linearly-interpolated percentile (p in [0,1])
// of values, matching the conventional "nearest-rank with interpolation"
// method used by numpy's default (`linear`) percentile interpolation.
func percentileInt64(values []int64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if len(sorted) == 1 {
		return float64(sorted[0])
	}
	rank := p * float64(len(sorted)-1)
	lower := int(rank)
	if lower >= len(sorted)-1 {
		return float64(sorted[len(sorted)-1])
	}
	fraction := rank - float64(lower)
	return float64(sorted[lower]) + fraction*float64(sorted[lower+1]-sorted[lower])
}

func minInt64(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return float64(minimum)
}

func maxInt64(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return float64(maximum)
}

func medianInt64(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return float64(sorted[middle])
	}
	return float64(sorted[middle-1]+sorted[middle]) / 2
}

func (workload *tickWorkload) runStreamTickCount() int {
	highest := -1
	for index, span := range workload.readSendSpan {
		if span.seen && index > highest {
			highest = index
		}
	}
	if highest >= 0 {
		return highest + 1
	}
	return workload.highestIndex + 1
}

type arrivalTracker struct {
	mu     sync.Mutex
	values map[string][]time.Time
}

func newArrivalTracker() *arrivalTracker {
	return &arrivalTracker{values: map[string][]time.Time{}}
}

func (tracker *arrivalTracker) record(parameter string, path string, receivedAt time.Time) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	key := streamKey(parameter, path)
	tracker.values[key] = append(tracker.values[key], receivedAt)
}

func (tracker *arrivalTracker) pop(parameter string, path string, count int) []time.Time {
	if count == 0 {
		return nil
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	key := streamKey(parameter, path)
	values := tracker.values[key]
	if len(values) < count {
		count = len(values)
	}
	out := append([]time.Time(nil), values[:count]...)
	tracker.values[key] = values[count:]
	return out
}

func (tracker *arrivalTracker) clear() {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.values = map[string][]time.Time{}
}

func streamKey(parameter string, path string) string {
	return parameter + "\x00" + path
}

func parsePositiveInts(value string) ([]int, error) {
	parts := parseList(value)
	out := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		var parsed int
		if _, err := fmt.Sscanf(part, "%d", &parsed); err != nil {
			return nil, err
		}
		if parsed <= 0 {
			return nil, fmt.Errorf("%d is not positive", parsed)
		}
		if !seen[parsed] {
			seen[parsed] = true
			out = append(out, parsed)
		}
	}
	sort.Ints(out)
	return out, nil
}

func parseList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
