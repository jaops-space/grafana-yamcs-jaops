package main

import (
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
	Streams                   int     `json:"streams"`
	UniqueParameters          int     `json:"unique_parameters"`
	PathsPerParameterAverage  float64 `json:"paths_per_parameter_avg"`
	PathsPerParameterMax      int     `json:"paths_per_parameter_max"`
	StreamGoroutines          int     `json:"stream_goroutines"`
	Goroutines                int     `json:"goroutines"`
	SetupNS                   int64   `json:"setup_ns"`
	ScenarioNS                int64   `json:"scenario_ns"`
	ProcessEvents             int64   `json:"process_events"`
	AverageProcessNS          float64 `json:"avg_process_ns"`
	AverageProcessStreams     float64 `json:"avg_process_streams"`
	ReadClearOperations       int64   `json:"read_clear_operations"`
	AverageReadClearNS        float64 `json:"avg_read_clear_ns"`
	ValuesRead                int64   `json:"values_read"`
	ValuesReadPerSecond       float64 `json:"values_read_per_sec"`
	AverageValuesPerReadClear float64 `json:"avg_values_per_read_clear"`
	FreshnessWindowNS         int64   `json:"freshness_window_ns"`
	ValuesReadFresh           int64   `json:"values_read_fresh"`
	ValuesReadStale           int64   `json:"values_read_stale"`
	ValuesReadFreshPercent    float64 `json:"values_read_fresh_pct"`
	ValuesReadStalePercent    float64 `json:"values_read_stale_pct"`
	AverageValueReadAgeNS     float64 `json:"avg_value_read_age_ns"`
	MaxValueReadAgeNS         int64   `json:"max_value_read_age_ns"`
	MaxValueStallNS           int64   `json:"max_value_stall_ns"`
	TickIntervalNS            int64   `json:"tick_interval_ns"`
	AverageTickRunStreamNS    float64 `json:"avg_tick_runstream_ns"`
	MaxTickRunStreamNS        int64   `json:"max_tick_runstream_ns"`
	MaxTickRunStreamPercent   float64 `json:"max_tick_runstream_pct"`
	TicksOverInterval         int     `json:"ticks_over_interval"`
	AverageTickProcessNS      float64 `json:"avg_tick_process_ns"`
	MaxTickProcessNS          int64   `json:"max_tick_process_ns"`
	AverageTickReadSendNS     float64 `json:"avg_tick_read_send_ns"`
	MaxTickReadSendNS         int64   `json:"max_tick_read_send_ns"`
	LiveMemoryGrowthBytes     int64   `json:"live_memory_growth_bytes"`
	TotalAllocatedBytes       uint64  `json:"total_allocated_bytes"`
	NumGCDelta                uint32  `json:"num_gc_delta"`
	ActiveParameterStreams    int     `json:"active_parameter_streams"`
	ActiveYamcsSubscriptions  int     `json:"active_yamcs_subscriptions"`
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
	ReadIntervalMS  int              `json:"read_interval_ms"`
	FreshnessMS     int              `json:"freshness_ms"`
	Parameters      []string         `json:"parameters"`
	Scenarios       []scenarioMetric `json:"scenarios"`
}

type systemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUs      int    `json:"cpus"`
	GoVersion string `json:"go_version"`
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
	readInterval := flag.Duration("read-interval", time.Second, "interval between read-and-clear operations per Grafana stream")
	freshnessWindow := flag.Duration("freshness-window", time.Second, "maximum value age counted as read in the same telemetry cycle")
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
		StartedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		YamcsAddress: *address,
		Instance:     *instance,
		Processor:    *processor,
		System: systemInfo{
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
			GoVersion: runtime.Version(),
		},
		DurationSeconds: duration.Seconds(),
		WarmupSeconds:   warmup.Seconds(),
		ReadIntervalMS:  int(readInterval.Milliseconds()),
		FreshnessMS:     int(freshnessWindow.Milliseconds()),
		Parameters:      parameters,
		Scenarios:       make([]scenarioMetric, 0, len(streamCounts)),
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

	hostErrors, endpointErrors := mux.Connect(ctx, true)
	if len(hostErrors) > 0 || len(endpointErrors) > 0 {
		return scenarioMetric{}, fmt.Errorf("connect hostErrors=%v endpointErrors=%v", hostErrors, endpointErrors)
	}
	endpoint, err := mux.GetEndpoint("quickstart")
	if err != nil {
		return scenarioMetric{}, err
	}

	var processEvents atomic.Int64
	var processNSTotal atomic.Int64
	var processStreamsTotal atomic.Int64
	arrivals := newArrivalTracker()
	tickWork := newTickWorkload(readInterval)
	var scenarioStarted time.Time
	endpoint.ParameterProcessObserver = func(_ string, streamCount int, elapsed time.Duration) {
		processEvents.Add(1)
		processNSTotal.Add(elapsed.Nanoseconds())
		processStreamsTotal.Add(int64(streamCount))
		if !scenarioStarted.IsZero() {
			tickWork.addProcess(time.Since(scenarioStarted), elapsed)
		}
	}
	endpoint.ParameterBufferObserver = func(parameter string, path string, receivedAt time.Time) {
		arrivals.record(parameter, path, receivedAt)
	}

	requests := make([]streamRequest, 0, streams)
	pathsByParameter := map[string]int{}
	setupStarted := time.Now()
	for i := 0; i < streams; i++ {
		parameter := parameters[i%len(parameters)]
		path := fmt.Sprintf("benchmark/scenario/streams-%d/stream-%d", streams, i)
		if err := endpoint.RequestNewParameterStream(ctx, parameter, path); err != nil {
			return scenarioMetric{}, err
		}
		requests = append(requests, streamRequest{parameter: parameter, path: path})
		pathsByParameter[parameter]++
	}
	setupNS := time.Since(setupStarted).Nanoseconds()

	time.Sleep(warmup)
	for _, req := range requests {
		endpoint.GetAndClearParameterStreamBuffer(req.parameter, req.path)
	}
	arrivals.clear()
	processEvents.Store(0)
	processNSTotal.Store(0)
	processStreamsTotal.Store(0)
	runtime.GC()

	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)

	var valuesRead atomic.Int64
	var freshValues atomic.Int64
	var readAgeNSTotal atomic.Int64
	var maxReadAgeNS atomic.Int64
	var readOps atomic.Int64
	var readNSTotal atomic.Int64
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
					tickOffset := time.Duration(0)
					if !scenarioStarted.IsZero() {
						tickOffset = time.Since(scenarioStarted)
					}
					started := time.Now()
					values := endpoint.GetAndClearParameterStreamBuffer(req.parameter, req.path)
					readAt := time.Now()
					receivedAtValues := arrivals.pop(req.parameter, req.path, len(values))
					for _, receivedAt := range receivedAtValues {
						age := readAt.Sub(receivedAt)
						if age < 0 {
							age = 0
						}
						ageNS := age.Nanoseconds()
						readAgeNSTotal.Add(ageNS)
						updateMaxInt64(&maxReadAgeNS, ageNS)
						if age <= freshnessWindow {
							freshValues.Add(1)
						}
					}
					if len(values) > 0 {
						if len(values) > 3 {
							frame := tools.ConvertBufferToAverageFrame(values, req.parameter, false, false, "", false)
							runtime.KeepAlive(frame)
						} else {
							frame := tools.ConvertBufferToFrame(values, req.parameter, false, false, "", false)
							runtime.KeepAlive(frame)
						}
					}
					readSendElapsed := time.Since(started)
					readNSTotal.Add(readSendElapsed.Nanoseconds())
					tickWork.addReadSend(tickOffset, readSendElapsed)
					readOps.Add(1)
					valuesRead.Add(int64(len(values)))
				case <-stop:
					return
				}
			}
		}()
	}

	scenarioStarted = time.Now()
	goroutines := runtime.NumGoroutine()
	time.Sleep(duration)
	scenarioNS := time.Since(scenarioStarted).Nanoseconds()
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
	processCount := processEvents.Load()
	valueCount := valuesRead.Load()
	maxPaths := 0
	for _, count := range pathsByParameter {
		if count > maxPaths {
			maxPaths = count
		}
	}

	metric := scenarioMetric{
		Streams:                  streams,
		UniqueParameters:         len(pathsByParameter),
		PathsPerParameterAverage: float64(streams) / float64(len(pathsByParameter)),
		PathsPerParameterMax:     maxPaths,
		StreamGoroutines:         streams,
		Goroutines:               goroutines,
		SetupNS:                  setupNS,
		ScenarioNS:               scenarioNS,
		ProcessEvents:            processCount,
		ReadClearOperations:      readCount,
		ValuesRead:               valueCount,
		ValuesReadPerSecond:      float64(valueCount) / duration.Seconds(),
		FreshnessWindowNS:        freshnessWindow.Nanoseconds(),
		LiveMemoryGrowthBytes:    int64(memEnd.Alloc) - int64(memStart.Alloc),
		TotalAllocatedBytes:      memEnd.TotalAlloc - memStart.TotalAlloc,
		NumGCDelta:               memEnd.NumGC - memStart.NumGC,
		ActiveParameterStreams:   streams,
		ActiveYamcsSubscriptions: len(pathsByParameter),
	}
	if readCount > 0 {
		metric.AverageReadClearNS = float64(readNSTotal.Load()) / float64(readCount)
		metric.AverageValuesPerReadClear = float64(valueCount) / float64(readCount)
	}
	freshCount := freshValues.Load()
	metric.ValuesReadFresh = freshCount
	metric.ValuesReadStale = valueCount - freshCount
	if valueCount > 0 {
		metric.ValuesReadFreshPercent = 100 * float64(freshCount) / float64(valueCount)
		metric.ValuesReadStalePercent = 100 - metric.ValuesReadFreshPercent
		metric.AverageValueReadAgeNS = float64(readAgeNSTotal.Load()) / float64(valueCount)
		metric.MaxValueReadAgeNS = maxReadAgeNS.Load()
		if metric.MaxValueReadAgeNS > freshnessWindow.Nanoseconds() {
			metric.MaxValueStallNS = metric.MaxValueReadAgeNS - freshnessWindow.Nanoseconds()
		}
	}
	if processCount > 0 {
		metric.AverageProcessNS = float64(processNSTotal.Load()) / float64(processCount)
		metric.AverageProcessStreams = float64(processStreamsTotal.Load()) / float64(processCount)
	}
	tickSummary := tickWork.summary()
	metric.TickIntervalNS = readInterval.Nanoseconds()
	metric.AverageTickRunStreamNS = tickSummary.AverageTotalNS
	metric.MaxTickRunStreamNS = tickSummary.MaxTotalNS
	metric.MaxTickRunStreamPercent = tickSummary.MaxTotalPercent
	metric.TicksOverInterval = tickSummary.TicksOverInterval
	metric.AverageTickProcessNS = tickSummary.AverageProcessNS
	metric.MaxTickProcessNS = tickSummary.MaxProcessNS
	metric.AverageTickReadSendNS = tickSummary.AverageReadSendNS
	metric.MaxTickReadSendNS = tickSummary.MaxReadSendNS
	return metric, nil
}

type tickWorkload struct {
	mu           sync.Mutex
	interval     time.Duration
	processNS    map[int]int64
	readSendNS   map[int]int64
	highestIndex int
}

type tickWorkloadSummary struct {
	AverageTotalNS    float64
	MaxTotalNS        int64
	MaxTotalPercent   float64
	TicksOverInterval int
	AverageProcessNS  float64
	MaxProcessNS      int64
	AverageReadSendNS float64
	MaxReadSendNS     int64
}

func newTickWorkload(interval time.Duration) *tickWorkload {
	if interval <= 0 {
		interval = time.Second
	}
	return &tickWorkload{
		interval:   interval,
		processNS:  map[int]int64{},
		readSendNS: map[int]int64{},
	}
}

func (workload *tickWorkload) addProcess(offset time.Duration, elapsed time.Duration) {
	workload.add(workload.processNS, offset, elapsed)
}

func (workload *tickWorkload) addReadSend(offset time.Duration, elapsed time.Duration) {
	workload.add(workload.readSendNS, offset, elapsed)
}

func (workload *tickWorkload) add(target map[int]int64, offset time.Duration, elapsed time.Duration) {
	if offset < 0 {
		offset = 0
	}
	index := int(offset / workload.interval)
	workload.mu.Lock()
	defer workload.mu.Unlock()
	target[index] += elapsed.Nanoseconds()
	if index > workload.highestIndex {
		workload.highestIndex = index
	}
}

func (workload *tickWorkload) summary() tickWorkloadSummary {
	workload.mu.Lock()
	defer workload.mu.Unlock()
	tickCount := workload.highestIndex + 1
	if tickCount <= 0 {
		return tickWorkloadSummary{}
	}

	var totalSum int64
	var processSum int64
	var readSendSum int64
	var maxTotal int64
	var maxProcess int64
	var maxReadSend int64
	ticksOverInterval := 0
	for i := 0; i < tickCount; i++ {
		process := workload.processNS[i]
		readSend := workload.readSendNS[i]
		total := process + readSend
		totalSum += total
		processSum += process
		readSendSum += readSend
		if total > maxTotal {
			maxTotal = total
		}
		if process > maxProcess {
			maxProcess = process
		}
		if readSend > maxReadSend {
			maxReadSend = readSend
		}
		if total > workload.interval.Nanoseconds() {
			ticksOverInterval++
		}
	}

	return tickWorkloadSummary{
		AverageTotalNS:    float64(totalSum) / float64(tickCount),
		MaxTotalNS:        maxTotal,
		MaxTotalPercent:   100 * float64(maxTotal) / float64(workload.interval.Nanoseconds()),
		TicksOverInterval: ticksOverInterval,
		AverageProcessNS:  float64(processSum) / float64(tickCount),
		MaxProcessNS:      maxProcess,
		AverageReadSendNS: float64(readSendSum) / float64(tickCount),
		MaxReadSendNS:     maxReadSend,
	}
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

func updateMaxInt64(target *atomic.Int64, value int64) {
	for {
		current := target.Load()
		if value <= current || target.CompareAndSwap(current, value) {
			return
		}
	}
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
