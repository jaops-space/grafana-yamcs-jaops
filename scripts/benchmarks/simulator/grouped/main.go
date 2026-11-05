//go:build benchmark

// Command grouped measures Simulator backend cost per logical client query group
// per tick by driving the real plugin.RunParameterStream against a live Yamcs.
//
// Run it from a plugin checkout (the working directory defines the plugin revision):
//
//	go run -tags benchmark ./scripts/benchmarks/simulator/grouped \
//	  -workload scripts/benchmarks/workload.json -address localhost:8193 \
//	  -sizes 1,2,3,4,8,16,32,64,max -max-groups 64 -repetitions 3 \
//	  -duration 5s -warmup 2s -out grouped-head.json
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/plugin"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
)

const (
	metricSemanticsVersion = 2
	metricName             = "group_tick_busy_ns"
	defaultSizes           = "1,2,3,4,8,16,32,64,max"
	timingBoundary         = "Timer starts when the RunParameterStream ticker fires and stops after SendFrame returns. " +
		"The span covers DrainParameterStream, frame conversion (or buildMultiParameterFrame for grouped streams), " +
		"SetUnitAndThresholds and SendFrame, including SDK packet encoding and the StreamPacketSender.Send call. " +
		"It excludes ticker wait, websocket receive and parsing inside the Yamcs client, and warmup. " +
		"Only ticks that produced a frame are recorded. Split group ticks are the index-aligned sum of member spans; " +
		"member tickers are not phase-locked, so split sums are approximations of one logical client tick."
	probeMinValues = 2
	pointsPerTick  = 1000
)

var hexSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

type options struct {
	workload      string
	address       string
	endpointOver  string
	processor     string
	protobuf      bool
	sizes         string
	mode          string
	maxGroups     int
	repetitions   int
	repStart      int
	duration      time.Duration
	warmup        time.Duration
	tick          time.Duration
	overlap       float64
	minTicks      int
	probe         time.Duration
	out           string
	fixtureRepo   string
	fixtureCommit string
	merge         string
}

type workload struct {
	Semver            string   `json:"semver"`
	Instance          string   `json:"instance"`
	Endpoint          string   `json:"endpoint"`
	GroupedParameters []string `json:"grouped_parameters"`
}

type streamSpec struct {
	path  string
	query plugin.PluginQuery
}

type countingSender struct {
	packets atomic.Int64
	bytes   atomic.Int64
}

func (s *countingSender) Send(p *backend.StreamPacket) error {
	if p == nil {
		return errors.New("nil stream packet")
	}
	s.packets.Add(1)
	s.bytes.Add(int64(len(p.Data)))
	return nil
}

type revisionInfo struct {
	PluginCommit       string            `json:"plugin_commit"`
	PluginBranch       string            `json:"plugin_branch"`
	WorktreeDirtyFiles []string          `json:"worktree_dirty_files"`
	OverlaySHA256      map[string]string `json:"overlay_sha256"`
	WorkingDir         string            `json:"working_dir"`
}

type environmentInfo struct {
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	NumCPU     int    `json:"num_cpu"`
	GOMAXPROCS int    `json:"gomaxprocs"`
	GoVersion  string `json:"go_version"`
}

type scenarioMeta struct {
	GroupSize              int     `json:"group_size"`
	Groups                 int     `json:"groups"`
	PoolSize               int     `json:"pool_size"`
	Stride                 int     `json:"stride"`
	GroupStartIndex        []int   `json:"group_start_index"`
	OverlapTarget          float64 `json:"overlap_target"`
	OverlapCountTarget     int     `json:"overlap_count_target"`
	OverlapAchieved        float64 `json:"overlap_achieved"`
	OverlapPerAdjacentPair []int   `json:"overlap_per_adjacent_pair"`
	DistinctGroups         int     `json:"distinct_groups"`
	FullPoolWindow         bool    `json:"full_pool_window"`
	ActualStreams          int     `json:"actual_streams"`
}

type repSummary struct {
	Repetition int     `json:"repetition"`
	GroupTicks int     `json:"group_ticks"`
	MedianNS   float64 `json:"median_ns"`
}

type scenarioResult struct {
	scenarioMeta
	Repetitions                 int          `json:"repetitions"`
	RepetitionIndices           []int        `json:"repetition_indices"`
	ObservedGroupTicks          int          `json:"observed_group_ticks"`
	NonemptyGroupTicks          int          `json:"nonempty_group_ticks"`
	SerializedPackets           int64        `json:"serialized_packets"`
	SerializedBytes             int64        `json:"serialized_bytes"`
	GroupTickBusyNS             float64      `json:"group_tick_busy_ns"`
	GroupTickBusyNSDistribution Distribution `json:"group_tick_busy_ns_distribution"`
	PerRepetition               []repSummary `json:"per_repetition"`
	RawGroupTickSeriesNS        [][][]int64  `json:"raw_group_tick_series_ns"`
}

type benchOutput struct {
	BenchmarkSemver        string            `json:"benchmark_semver"`
	MetricSemanticsVersion int               `json:"metric_semantics_version"`
	QueryCadence           string            `json:"query_cadence"`
	SplitNormalization     string            `json:"split_normalization"`
	Metric                 string            `json:"metric"`
	Mode                   string            `json:"mode"`
	Revision               revisionInfo      `json:"revision"`
	Environment            environmentInfo   `json:"environment"`
	Invocation             []string          `json:"invocation"`
	FixtureRepo            string            `json:"fixture_repo"`
	FixtureCommit          string            `json:"fixture_commit"`
	Address                string            `json:"address"`
	Protobuf               bool              `json:"protobuf"`
	Instance               string            `json:"instance"`
	Endpoint               string            `json:"endpoint"`
	Processor              string            `json:"processor"`
	TickSeconds            int               `json:"tick_seconds"`
	WarmupSeconds          float64           `json:"warmup_seconds"`
	DurationSeconds        float64           `json:"duration_seconds"`
	OverlapTarget          float64           `json:"overlap_target"`
	MaxGroups              int               `json:"max_groups"`
	MinTicksPerGroup       int               `json:"min_ticks_per_group"`
	RepStart               int               `json:"rep_start"`
	Repetitions            int               `json:"repetitions"`
	ProbeSeconds           float64           `json:"probe_seconds"`
	ProbeValueCounts       map[string]int    `json:"probe_value_counts"`
	ProbeErrors            map[string]string `json:"probe_errors,omitempty"`
	CandidateParameters    int               `json:"candidate_parameters"`
	Parameters             []string          `json:"parameters"`
	UnavailableParameters  []string          `json:"unavailable_parameters"`
	TimingBoundary         string            `json:"timing_boundary"`
	StartedAt              string            `json:"started_at"`
	FinishedAt             string            `json:"finished_at"`
	Scenarios              []scenarioResult  `json:"scenarios"`
}

func main() {
	o, err := parseFlags()
	if err != nil {
		fail(err)
	}
	if o.merge != "" {
		if err := mergeOutputs(strings.Split(o.merge, ","), o.out); err != nil {
			fail(err)
		}
		return
	}
	if err := run(o); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "grouped benchmark error:", err)
	os.Exit(1)
}

func parseFlags() (options, error) {
	var o options
	flag.StringVar(&o.workload, "workload", "scripts/benchmarks/workload.json", "workload JSON with grouped_parameters, instance and endpoint")
	flag.StringVar(&o.address, "address", "localhost:8193", "Yamcs host:port for the benchmark instance (no scheme; https:// sets TLS)")
	flag.StringVar(&o.endpointOver, "endpoint", "", "override endpoint id from workload")
	flag.StringVar(&o.processor, "processor", "realtime", "Yamcs processor name")
	flag.BoolVar(&o.protobuf, "protobuf", false, "use protobuf transport for the Yamcs client")
	flag.StringVar(&o.sizes, "sizes", defaultSizes, "group sizes: comma list, ranges are not supported, tokens 'max' and 'all' (all = 1..pool)")
	flag.StringVar(&o.mode, "mode", "auto", "auto (grouped if the revision has PluginQuery.Parameters, else split), grouped, or split")
	flag.IntVar(&o.maxGroups, "max-groups", 64, "logical client groups per group size (no stream cap is applied)")
	flag.IntVar(&o.repetitions, "repetitions", 3, "measured repetitions per group size")
	flag.IntVar(&o.repStart, "rep-start", 0, "first repetition index (lets the caller alternate revision order across processes)")
	flag.DurationVar(&o.duration, "duration", 5*time.Second, "measured window per repetition")
	flag.DurationVar(&o.warmup, "warmup", 2*time.Second, "warmup per repetition, discarded")
	flag.DurationVar(&o.tick, "tick", time.Second, "RunParameterStream tick (whole seconds)")
	flag.Float64Var(&o.overlap, "overlap", 0.15, "target adjacent-group overlap fraction")
	flag.IntVar(&o.minTicks, "min-ticks", 3, "minimum recorded group ticks per group and per split member")
	flag.DurationVar(&o.probe, "probe", 6*time.Second, "coverage probe window per candidate parameter")
	flag.StringVar(&o.out, "out", "", "output JSON path (stdout if empty)")
	flag.StringVar(&o.fixtureRepo, "fixture-repo", "../grafana-yamcs-demo", "fixture git checkout used to populate fixture_commit")
	flag.StringVar(&o.fixtureCommit, "fixture-commit", "", "fixture commit SHA (default: git rev-parse HEAD of -fixture-repo)")
	flag.StringVar(&o.merge, "merge", "", "comma-separated output JSONs of one revision to pool into -out")
	flag.Parse()
	return o, nil
}

func run(o options) error {
	if o.repetitions < 1 || o.maxGroups < 1 || o.minTicks < 1 {
		return errors.New("repetitions, max-groups and min-ticks must be at least 1")
	}
	if o.duration <= 0 || o.warmup < 0 || o.probe <= 0 {
		return errors.New("duration and probe must be positive, warmup non-negative")
	}
	if o.tick < time.Second || o.tick%time.Second != 0 {
		return errors.New("tick must be a whole number of seconds (at least 1s)")
	}
	if o.overlap < 0 || o.overlap >= 1 {
		return errors.New("overlap must be in [0,1)")
	}
	if o.out == "" {
		return errors.New("-out is required")
	}

	hasParameters := paramsCapability()
	var grouped bool
	switch o.mode {
	case "auto":
		grouped = hasParameters
	case "grouped":
		if !hasParameters {
			return errors.New("mode grouped requires PluginQuery.Parameters at this revision")
		}
		grouped = true
	case "split":
		grouped = false
	default:
		return fmt.Errorf("unknown mode %q", o.mode)
	}
	mode := "split"
	if grouped {
		mode = "grouped"
	}

	wl, err := loadWorkload(o.workload)
	if err != nil {
		return err
	}
	if o.endpointOver != "" {
		wl.Endpoint = o.endpointOver
	}
	if wl.Semver == "" || len(wl.GroupedParameters) == 0 {
		return errors.New("workload must define semver and grouped_parameters")
	}

	rev, err := revisionFacts()
	if err != nil {
		return err
	}
	fixtureCommit := o.fixtureCommit
	if fixtureCommit == "" {
		fixtureCommit, err = gitOutput(o.fixtureRepo, "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("resolve fixture commit: %w", err)
		}
	}
	if !hexSHA.MatchString(fixtureCommit) {
		return fmt.Errorf("fixture commit %q is not a full 40-character SHA", fixtureCommit)
	}

	ctx := context.Background()
	started := time.Now().UTC()
	fmt.Fprintf(os.Stderr, "plugin %s mode=%s address=%s fixture=%s\n", rev.PluginCommit, mode, o.address, fixtureCommit)

	ep, err := connectEndpoint(ctx, o, wl)
	if err != nil {
		return err
	}
	defer ep.Multiplexer.Dispose()

	fmt.Fprintf(os.Stderr, "probing %d candidate parameters for %s\n", len(wl.GroupedParameters), o.probe)
	probeCounts, probeErrs, err := probeCoverage(ctx, ep, wl.GroupedParameters, o.probe)
	if err != nil {
		return err
	}
	eligible, unavailable := EligibleParameters(wl.GroupedParameters, probeCounts, probeMinValues)
	if len(eligible) == 0 {
		return errors.New("no live-eligible parameters")
	}
	sizes, err := SweepSizes(o.sizes, len(eligible))
	if err != nil {
		return err
	}

	out := benchOutput{
		BenchmarkSemver:        wl.Semver,
		MetricSemanticsVersion: metricSemanticsVersion,
		QueryCadence:           fmt.Sprintf("from=0s to=%ds points=%d (identical for split and grouped)", int(o.tick/time.Second)*pointsPerTick, pointsPerTick),
		SplitNormalization:     "per-member tick counts must be within 1 of each other and of duration/tick; edge ticks trimmed; otherwise explicit incomplete-observation error (no zero-cost members)",
		Metric:                 metricName,
		Mode:                   mode,
		Revision:               rev,
		Environment: environmentInfo{
			GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, NumCPU: runtime.NumCPU(),
			GOMAXPROCS: runtime.GOMAXPROCS(0), GoVersion: runtime.Version(),
		},
		Invocation:            os.Args,
		FixtureRepo:           o.fixtureRepo,
		FixtureCommit:         fixtureCommit,
		Address:               o.address,
		Protobuf:              o.protobuf,
		Instance:              wl.Instance,
		Endpoint:              wl.Endpoint,
		Processor:             o.processor,
		TickSeconds:           int(o.tick / time.Second),
		WarmupSeconds:         o.warmup.Seconds(),
		DurationSeconds:       o.duration.Seconds(),
		OverlapTarget:         o.overlap,
		MaxGroups:             o.maxGroups,
		MinTicksPerGroup:      o.minTicks,
		RepStart:              o.repStart,
		Repetitions:           o.repetitions,
		ProbeSeconds:          o.probe.Seconds(),
		ProbeValueCounts:      probeCounts,
		ProbeErrors:           probeErrs,
		CandidateParameters:   len(wl.GroupedParameters),
		Parameters:            eligible,
		UnavailableParameters: unavailable,
		TimingBoundary:        timingBoundary,
		StartedAt:             started.Format(time.RFC3339),
	}

	for _, k := range sizes {
		plan, err := PlanGroups(eligible, k, o.maxGroups, o.overlap)
		if err != nil {
			return err
		}
		meta := metaFromPlan(plan, len(eligible))
		if grouped {
			meta.ActualStreams = len(plan.Groups)
		} else {
			meta.ActualStreams = len(plan.Groups) * k
		}
		fmt.Fprintf(os.Stderr, "K=%d groups=%d streams=%d overlap=%.3f\n", k, len(plan.Groups), meta.ActualStreams, meta.OverlapAchieved)

		var repIdx []int
		var raw [][][]int64
		var packets, bytes int64
		for r := o.repStart; r < o.repStart+o.repetitions; r++ {
			series, p, b, err := runRepetition(ctx, ep, wl.Endpoint, o, plan, grouped, r)
			if err != nil {
				return fmt.Errorf("K=%d repetition %d: %w", k, r, err)
			}
			repIdx = append(repIdx, r)
			raw = append(raw, series)
			packets += p
			bytes += b
			fmt.Fprintf(os.Stderr, "  rep %d: %d group series, %d packets\n", r, len(series), p)
		}
		sc := buildScenario(meta, repIdx, raw, packets, bytes)
		out.Scenarios = append(out.Scenarios, sc)
		fmt.Fprintf(os.Stderr, "  median group_tick_busy_ns=%.0f over %d ticks\n", sc.GroupTickBusyNS, sc.ObservedGroupTicks)

		out.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		if err := writeJSON(o.out, out); err != nil {
			return err
		}
	}
	return nil
}

// paramsCapability reports whether this revision's PluginQuery has the
// multi-parameter Parameters field, detected without a compile-time reference.
func paramsCapability() bool {
	_, ok := reflect.TypeOf(plugin.PluginQuery{}).FieldByName("Parameters")
	return ok
}

func loadWorkload(path string) (workload, error) {
	var wl workload
	raw, err := os.ReadFile(path)
	if err != nil {
		return wl, err
	}
	if err := json.Unmarshal(raw, &wl); err != nil {
		return wl, fmt.Errorf("parse %s: %w", path, err)
	}
	return wl, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func revisionFacts() (revisionInfo, error) {
	var rev revisionInfo
	commit, err := gitOutput(".", "rev-parse", "HEAD")
	if err != nil {
		return rev, err
	}
	if !hexSHA.MatchString(commit) {
		return rev, fmt.Errorf("plugin commit %q is not a full 40-character SHA", commit)
	}
	rev.PluginCommit = commit
	rev.PluginBranch, _ = gitOutput(".", "rev-parse", "--abbrev-ref", "HEAD")
	status, err := gitOutput(".", "status", "--porcelain")
	if err != nil {
		return rev, err
	}
	for _, line := range strings.Split(status, "\n") {
		// gitOutput trims the first line's leading space, so fixed-width slicing is unreliable.
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			rev.WorktreeDirtyFiles = append(rev.WorktreeDirtyFiles, fields[len(fields)-1])
		}
	}
	rev.OverlaySHA256 = map[string]string{}
	for _, f := range []string{"pkg/plugin/benchmark_stats.go", "pkg/plugin/benchmark_group_adapter.go"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			rev.OverlaySHA256[f] = "absent"
			continue
		}
		sum := sha256.Sum256(raw)
		rev.OverlaySHA256[f] = hex.EncodeToString(sum[:])
	}
	rev.WorkingDir, _ = os.Getwd()
	return rev, nil
}

func connectEndpoint(ctx context.Context, o options, wl workload) (*source.YamcsEndpoint, error) {
	cfg := &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"bench": {ID: "bench", Name: "bench", Path: hostPort(o.address), Tls: strings.HasPrefix(o.address, "https://"), Protobuf: o.protobuf},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			wl.Endpoint: {ID: wl.Endpoint, Host: "bench", Instance: wl.Instance, Processor: o.processor},
		},
	}
	secure := &config.YamcsSecureConfiguration{Hosts: map[string]*config.YamcsSecureHost{}}
	mux, err := source.NewMultiplexerWithContext(ctx, cfg, secure)
	if err != nil {
		return nil, err
	}
	hostErrs, endpointErrs := mux.ConnectSync(ctx, true)
	for id, err := range hostErrs {
		if err != nil {
			return nil, fmt.Errorf("host %s: %w", id, err)
		}
	}
	for id, err := range endpointErrs {
		if err != nil {
			return nil, fmt.Errorf("endpoint %s: %w", id, err)
		}
	}
	ep, err := mux.GetEndpoint(wl.Endpoint)
	if err != nil {
		return nil, err
	}
	if err := ep.EnsureReady(); err != nil {
		return nil, fmt.Errorf("endpoint not ready: %w", err)
	}
	return ep, nil
}

// probeCoverage subscribes every candidate parameter once and counts the engineering
// values it drained over the window. Only parameters the live endpoint actually delivers
// count as eligible, so multidimensional leaves that never produce values are excluded.
func probeCoverage(ctx context.Context, ep *source.YamcsEndpoint, pool []string, window time.Duration) (map[string]int, map[string]string, error) {
	counts := map[string]int{}
	errs := map[string]string{}
	paths := map[string]string{}
	for i, p := range pool {
		path := fmt.Sprintf("/bench/probe/%03d", i)
		if err := ep.RequestNewParameterStream(ctx, p, path); err != nil {
			errs[p] = err.Error()
			counts[p] = 0
			continue
		}
		paths[p] = path
	}
	if err := sleepCtx(ctx, window); err != nil {
		return nil, nil, err
	}
	for _, p := range pool {
		path, ok := paths[p]
		if !ok {
			continue
		}
		for _, value := range ep.DrainParameterStream(p, path) {
			if value.GetEngValue() != nil {
				counts[p]++
			}
		}
		if err := ep.WithdrawParameterStreamRequest(ctx, p, path); err != nil {
			return nil, nil, fmt.Errorf("withdraw probe %s: %w", p, err)
		}
	}
	return counts, errs, nil
}

func buildQuery(endpointID string, tickSeconds int, params []string, grouped bool) (plugin.PluginQuery, error) {
	body := map[string]any{
		"type":      "plot",
		"endpoint":  endpointID,
		"parameter": params[0],
		"from":      0,
		"to":        tickSeconds * pointsPerTick,
		"points":    pointsPerTick,
	}
	if grouped {
		body["parameters"] = params
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return plugin.PluginQuery{}, err
	}
	var q plugin.PluginQuery
	if err := json.Unmarshal(raw, &q); err != nil {
		return plugin.PluginQuery{}, err
	}
	return q, nil
}

// runRepetition starts every group's stream(s) concurrently, discards warmup, then
// measures the real per-tick work spans for the window and returns one series per
// logical group (grouped: the group stream's spans; split: index-aligned sum of its members).
func runRepetition(ctx context.Context, ep *source.YamcsEndpoint, endpointID string, o options, plan GroupPlan, grouped bool, rep int) ([][]int64, int64, int64, error) {
	tickSeconds := int(o.tick / time.Second)
	k := plan.Size
	var specs []streamSpec
	groupPaths := make([]string, len(plan.Groups))
	memberPaths := make([][]string, len(plan.Groups))
	for g, window := range plan.Groups {
		if grouped {
			path := fmt.Sprintf("/bench/k%03d/r%02d/g%03d", k, rep, g)
			q, err := buildQuery(endpointID, tickSeconds, window, true)
			if err != nil {
				return nil, 0, 0, err
			}
			specs = append(specs, streamSpec{path: path, query: q})
			groupPaths[g] = path
			continue
		}
		for j, p := range window {
			path := fmt.Sprintf("/bench/k%03d/r%02d/g%03d/m%03d", k, rep, g, j)
			q, err := buildQuery(endpointID, tickSeconds, []string{p}, false)
			if err != nil {
				return nil, 0, 0, err
			}
			specs = append(specs, streamSpec{path: path, query: q})
			memberPaths[g] = append(memberPaths[g], path)
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	senders := make([]*countingSender, len(specs))
	errCh := make(chan error, len(specs))
	var wg sync.WaitGroup
	for i, s := range specs {
		senders[i] = &countingSender{}
		wg.Add(1)
		go func(s streamSpec, sender *countingSender) {
			defer wg.Done()
			if err := plugin.BenchmarkRunStream(runCtx, ep, s.query, s.path, sender); err != nil && runCtx.Err() == nil {
				errCh <- fmt.Errorf("stream %s: %w", s.path, err)
			}
		}(s, senders[i])
	}

	stop := func() error {
		cancel()
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
			return nil
		case <-time.After(60 * time.Second):
			return errors.New("streams did not stop within 60s; refusing to contaminate the next scenario")
		}
	}
	checkErr := func() error {
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	}

	if err := sleepCtx(ctx, o.warmup); err != nil {
		return nil, 0, 0, errors.Join(err, stop())
	}
	if err := checkErr(); err != nil {
		return nil, 0, 0, errors.Join(err, stop())
	}
	totals := func() (packets, bytes int64) {
		for _, sender := range senders {
			packets += sender.packets.Load()
			bytes += sender.bytes.Load()
		}
		return
	}
	startPackets, startBytes := totals()
	plugin.BenchmarkResetRunStreamWork()
	if err := sleepCtx(ctx, o.duration); err != nil {
		return nil, 0, 0, errors.Join(err, stop())
	}
	snap := plugin.BenchmarkRunStreamSamplesByPath()
	packets, bytes := totals()
	packets -= startPackets
	bytes -= startBytes
	if err := stop(); err != nil {
		return nil, 0, 0, err
	}
	if err := checkErr(); err != nil {
		return nil, 0, 0, err
	}

	series := make([][]int64, len(plan.Groups))
	for g := range plan.Groups {
		var groupSeries []int64
		if grouped {
			groupSeries = snap[groupPaths[g]]
		} else {
			members := make([][]int64, 0, len(memberPaths[g]))
			for _, mp := range memberPaths[g] {
				m := snap[mp]
				if len(m) < o.minTicks {
					return nil, 0, 0, fmt.Errorf("group %d member %s recorded %d ticks, need %d", g, mp, len(m), o.minTicks)
				}
				members = append(members, m)
			}
			expected := int(o.duration / o.tick)
			sums, sumErr := SumCompleteTicks(members, expected)
			if sumErr != nil {
				return nil, 0, 0, fmt.Errorf("group %d: %w", g, sumErr)
			}
			groupSeries = sums
		}
		if len(groupSeries) < o.minTicks {
			return nil, 0, 0, fmt.Errorf("group %d recorded %d ticks, need %d", g, len(groupSeries), o.minTicks)
		}
		series[g] = groupSeries
	}
	return series, packets, bytes, nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func metaFromPlan(plan GroupPlan, poolSize int) scenarioMeta {
	starts := make([]int, len(plan.Groups))
	for g := range plan.Groups {
		starts[g] = (g * plan.Stride) % poolSize
	}
	return scenarioMeta{
		GroupSize:              plan.Size,
		Groups:                 len(plan.Groups),
		PoolSize:               poolSize,
		Stride:                 plan.Stride,
		GroupStartIndex:        starts,
		OverlapTarget:          plan.OverlapTarget,
		OverlapCountTarget:     plan.OverlapCountWant,
		OverlapAchieved:        plan.OverlapAchieved,
		OverlapPerAdjacentPair: plan.AchievedPerPair,
		DistinctGroups:         plan.DistinctGroups,
		FullPoolWindow:         plan.FullPoolOverlapAt,
	}
}

func buildScenario(meta scenarioMeta, repIdx []int, raw [][][]int64, packets, bytes int64) scenarioResult {
	var all []int64
	var observed int
	per := make([]repSummary, 0, len(raw))
	for i, rep := range raw {
		var repVals []int64
		ticks := 0
		for _, series := range rep {
			ticks += len(series)
			repVals = append(repVals, series...)
		}
		observed += ticks
		all = append(all, repVals...)
		per = append(per, repSummary{
			Repetition: repIdx[i],
			GroupTicks: ticks,
			MedianNS:   ComputeDistribution(repVals).Median,
		})
	}
	dist := ComputeDistribution(all)
	return scenarioResult{
		scenarioMeta:                meta,
		Repetitions:                 len(raw),
		RepetitionIndices:           repIdx,
		ObservedGroupTicks:          observed,
		NonemptyGroupTicks:          observed,
		SerializedPackets:           packets,
		SerializedBytes:             bytes,
		GroupTickBusyNS:             dist.Median,
		GroupTickBusyNSDistribution: dist,
		PerRepetition:               per,
		RawGroupTickSeriesNS:        raw,
	}
}

// mergeOutputs pools repetitions from several outputs of the same revision.
// Inputs must agree on everything that defines the comparison.
func mergeOutputs(paths []string, out string) error {
	if out == "" {
		return errors.New("-out is required with -merge")
	}
	var merged *benchOutput
	bySize := map[int]*scenarioResult{}
	seenRep := map[int]map[int]bool{}
	for _, path := range paths {
		raw, err := os.ReadFile(strings.TrimSpace(path))
		if err != nil {
			return err
		}
		var in benchOutput
		if err := json.Unmarshal(raw, &in); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if len(in.Scenarios) == 0 {
			return fmt.Errorf("%s: no completed scenarios", path)
		}
		if merged == nil {
			copied := in
			copied.Scenarios = nil
			copied.Invocation = []string{"merge"}
			merged = &copied
		} else if err := sameDefinition(*merged, in, path); err != nil {
			return err
		}
		for _, sc := range in.Scenarios {
			if len(sc.RepetitionIndices) == 0 || len(sc.RepetitionIndices) != len(sc.RawGroupTickSeriesNS) {
				return fmt.Errorf("%s: K=%d has incomplete repetition data", path, sc.GroupSize)
			}
			for _, rep := range sc.RawGroupTickSeriesNS {
				if len(rep) != sc.Groups {
					return fmt.Errorf("%s: K=%d has incomplete group data", path, sc.GroupSize)
				}
				for _, series := range rep {
					if len(series) < in.MinTicksPerGroup {
						return fmt.Errorf("%s: K=%d has too few group ticks", path, sc.GroupSize)
					}
				}
			}
			acc, ok := bySize[sc.GroupSize]
			if !ok {
				bySize[sc.GroupSize] = &scenarioResult{scenarioMeta: sc.scenarioMeta}
				acc = bySize[sc.GroupSize]
				seenRep[sc.GroupSize] = map[int]bool{}
			}
			if acc.Groups != sc.Groups {
				return fmt.Errorf("%s: K=%d has %d groups, expected %d", path, sc.GroupSize, sc.Groups, acc.Groups)
			}
			if !sameInts(acc.GroupStartIndex, sc.GroupStartIndex) || acc.OverlapAchieved != sc.OverlapAchieved {
				return fmt.Errorf("%s: K=%d group plan differs from earlier input", path, sc.GroupSize)
			}
			for i, r := range sc.RepetitionIndices {
				if seenRep[sc.GroupSize][r] {
					return fmt.Errorf("%s: repetition %d for K=%d appears twice", path, r, sc.GroupSize)
				}
				seenRep[sc.GroupSize][r] = true
				acc.RepetitionIndices = append(acc.RepetitionIndices, r)
				acc.RawGroupTickSeriesNS = append(acc.RawGroupTickSeriesNS, sc.RawGroupTickSeriesNS[i])
			}
			acc.SerializedPackets += sc.SerializedPackets
			acc.SerializedBytes += sc.SerializedBytes
		}
	}
	if merged == nil {
		return errors.New("no inputs")
	}
	sizes := make([]int, 0, len(bySize))
	for k := range bySize {
		sizes = append(sizes, k)
	}
	sort.Ints(sizes)
	merged.Scenarios = nil
	for _, k := range sizes {
		acc := bySize[k]
		merged.Scenarios = append(merged.Scenarios, buildScenario(acc.scenarioMeta, acc.RepetitionIndices, acc.RawGroupTickSeriesNS, acc.SerializedPackets, acc.SerializedBytes))
	}
	merged.Repetitions = 0
	for _, sc := range merged.Scenarios {
		if sc.Repetitions > merged.Repetitions {
			merged.Repetitions = sc.Repetitions
		}
	}
	merged.Invocation = append([]string{"merge"}, paths...)
	merged.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSON(out, *merged)
}

func sameDefinition(a, b benchOutput, path string) error {
	switch {
	case a.BenchmarkSemver != b.BenchmarkSemver, a.MetricSemanticsVersion != b.MetricSemanticsVersion, a.Metric != b.Metric:
		return fmt.Errorf("%s: benchmark semver or metric differs", path)
	case a.Revision.PluginCommit != b.Revision.PluginCommit:
		return fmt.Errorf("%s: plugin commit %s differs from %s", path, b.Revision.PluginCommit, a.Revision.PluginCommit)
	case a.Mode != b.Mode:
		return fmt.Errorf("%s: mode differs", path)
	case !reflect.DeepEqual(a.Revision.OverlaySHA256, b.Revision.OverlaySHA256):
		return fmt.Errorf("%s: benchmark instrumentation overlays differ", path)
	case a.Environment != b.Environment, a.Protobuf != b.Protobuf:
		return fmt.Errorf("%s: runtime environment or transport differs", path)
	case a.FixtureCommit != b.FixtureCommit:
		return fmt.Errorf("%s: fixture commit differs", path)
	case a.TickSeconds != b.TickSeconds, a.WarmupSeconds != b.WarmupSeconds, a.DurationSeconds != b.DurationSeconds:
		return fmt.Errorf("%s: tick/warmup/duration differ", path)
	case a.OverlapTarget != b.OverlapTarget, a.MaxGroups != b.MaxGroups:
		return fmt.Errorf("%s: overlap target or max groups differ", path)
	case !sameStrings(a.Parameters, b.Parameters):
		return fmt.Errorf("%s: parameter list or order differs", path)
	}
	return nil
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if path == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// hostPort strips an optional http(s):// scheme and trailing slash from the address flag.
func hostPort(addr string) string {
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	return strings.TrimSuffix(addr, "/")
}
