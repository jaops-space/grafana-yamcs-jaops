# Benchmarking

This repository has three benchmark layers:

- Microbenchmarks for isolated frame/buffer processing helpers.
- Yamcs simulator scenarios for backend streaming behavior against Yamcs Demo.
- Grafana panel scenarios for the full browser + Grafana Live + plugin backend path.

Run the layers with:

```bash
pnpm run bench:micro
pnpm run bench:simulator
pnpm run bench:grafana
```

Build one local report from the raw layer outputs with:

```bash
pnpm run bench:report
```

Refresh the checked-in long-term baseline with the manual GitHub Actions workflow named `Refresh benchmark long-term baseline`.

The simulator benchmark targets Yamcs Demo (`demo/realtime`) on `localhost:8093`. By default it starts `simulator.py` from `/tmp/yamcs-demo` at `1 Hz`; pass `--no-simulator` when the demo Compose stack is already publishing telemetry.

The Grafana panel benchmark assumes:

- Grafana is running on `http://localhost:3000` (the benchmark harness uses the root-served Compose configuration, not the demo dashboard's `/grafana` subpath);
- the Linux backend binary in `dist` is built with `go build -tags "arrow_json_stdlib benchmark"` so benchmark-only resource endpoints are compiled in;
- the provisioned datasource UID is `jaops-yamcs-main`;
- Yamcs Demo and its simulator are running on port `8093`;
- the datasource has a `demo_realtime` endpoint pointing at that demo.

## Files

Benchmark code is organized by layer:

- `scripts/benchmarks/micro/run.py`: isolated Go microbenchmarks.
- `scripts/benchmarks/simulator/run.py`: Yamcs Demo scenario runner.
- `scripts/benchmarks/simulator/scenario.go`: Go workload used by the simulator runner.
- `scripts/benchmarks/workload.json`: shared benchmark semver, endpoint, and parameter coverage.
- `tests/benchmark/streaming.benchmark.spec.ts`: browser/Grafana benchmark.
- `scripts/benchmarks/report.py`: the single report builder. It reads raw layer JSON outputs directly and writes the PR comment, status, and plots.
- `scripts/benchmarks/common/`: shared implementation helpers and the central metric catalog. This keeps each layer runner small and avoids copying plot/report/baseline logic between layers.

## Simulator scenario

Each scenario runs `N` concurrent Grafana stream demands against Yamcs Demo's `demo/realtime` processor.

For every value of `N`, the benchmark:

1. Runs one discarded warmup scenario so Yamcs/plugin paths are warmed before measured scenarios.
2. Creates `N` Grafana stream paths distributed across the canonical `/myproject`, `/features`, and `/images` parameters.
3. Lets Yamcs warm up for `3s`.
4. Runs the measured workload for `10s`.
5. Runs one goroutine per Grafana stream.
6. Reads and clears each stream buffer every `1s`.
7. Converts read values into Grafana data frames, matching the normal RunStream read/frame/send path.
8. Records throughput, freshness, memory, setup time, and median RunStream busy time per tick.

- Yamcs simulator rate: `1 Hz`
- Grafana stream read ticker: `1s`
- Freshness window: `1s`
- Discarded warmup scenario: `25` streams for `3s`

## Outputs

Layer runners write raw outputs:

- `benchmark-output/microbenchmarks/microbenchmarks.json`
- `benchmark-output/simulator/simulator.json`
- `benchmark-output/simulator/simulator.csv`
- `benchmark-output/grafana/grafana.json`

`pnpm run bench:report` reads those raw files and writes:

- `benchmark-output/benchmark-status.json`
- `benchmark-output/benchmark-comment.md`
- `benchmark-output/plots/*.png`
- `benchmark-output/regression-plots/*.png`

`plots/` contains all generated benchmark plots. `regression-plots/` is retained as a small machine-readable subset for metrics that crossed a warn or fail threshold.

The blue shaded band on plots is the measured HEAD min-to-max range for that X value when the raw benchmark output contains repeated samples. If a metric has only one measured value for that X, the band collapses to the line.

## Baselines

Benchmark plots can show three curves:

- Blue: the current benchmark result.
- Slate: the PR base commit before the PR changes, when CI can benchmark it.
- Green dashed: the checked-in long-term baseline from `scripts/benchmarks/baselines/long-term/simulator.json`, `scripts/benchmarks/baselines/long-term/microbenchmarks.json`, and `scripts/benchmarks/baselines/long-term/grafana.json`.

The long-term baseline is intentionally committed to the repository so performance drift remains visible even when a PR base benchmark is unavailable or changes too often to be a useful historical reference. It should be refreshed only from GitHub Actions so the baseline is tied to the CI environment, not a developer workstation. Its metadata lives next to the result file in `scripts/benchmarks/baselines/long-term/metadata.json`.

Simulator and Grafana benchmark results include the workload's `benchmark_semver`, currently `3.0.0`. Comparisons against their old quickstart baselines are omitted until the canonical workload has a matching baseline; refresh that baseline through the manual workflow. `metric_semantics_version` remains at `2` because the metric calculations themselves did not change, so the independent microbenchmark baseline remains comparable. Change the benchmark semver whenever the fixture, parameter mix, or endpoint changes; comparison requires matching workload semver and metric semantics.

The report also shows informational median, maximum negative, and maximum positive percentage change for each plotted metric against each compatible baseline. These percentage changes do not affect WARN or FAIL status.

## Metrics by layer

### Microbenchmarks

Microbenchmarks isolate small backend helpers without Yamcs, Grafana, browser rendering, network traffic, or scheduler-heavy stream timing. They are cheap enough to use large sample counts and report median time, with min/max bands on plots to show spread.

They currently cover:

- numeric buffer to full frame conversion;
- numeric buffer to average frame conversion;
- numeric buffer to average/min/max frame conversion;
- discrete buffer to frame conversion;
- processing 10 Yamcs parameter values into active stream buffers.

These metrics are mainly for detecting low-level allocation or frame-building regressions. They are not intended to represent full user-visible Grafana performance.

### Simulator scenario

The simulator scenario measures the plugin backend against live Yamcs Demo data, but without rendering Grafana panels in a browser. It uses the real stream buffers and RunStream read/frame/send path.

The PR comment currently plots:

- live memory used during the run;
- total memory allocated during the run;
- values read per second from stream buffers;
- percentage of values read within the same 1 second simulator tick;
- median total RunStream busy time per 1 second tick;
- stream setup time.

The most useful signals here are throughput, freshness, memory, and RunStream busy time. RunStream busy time sums actual backend read/frame/send work per tick, so it avoids treating goroutine ticker scheduling delay as plugin work.

### Grafana scenario

The Grafana scenario measures the full browser + Grafana dashboard + Grafana Live + plugin backend path. It creates temporary dashboards with Timeseries panels over Yamcs Demo parameters, warms each panel count once, then records the measured run.

The PR comment currently plots:

- time to panels ready;
- browser heap after browser garbage collection;
- total backend RunStream runtime for the fixed sample window;
- median backend live heap while panels stream;
- backend datapoints produced per second (per unique backend stream - stays flat when panels share a stream);
- datapoints received per second across all panels (frontend, measured in the browser - scales with real panel count);
- live streams opened.

The Grafana layer is the closest to user-visible behavior. Browser heap and time-to-ready cover frontend cost. Backend RunStream runtime and backend heap cover plugin backend work while Grafana is actively streaming.

## Grafana panel benchmark

The Grafana benchmark creates real temporary Grafana dashboards with Timeseries panels and measures the full browser + Grafana Live + plugin backend path.

The one-parameter-per-query control cycles through `scripts/benchmarks/workload.json`, so small dashboards mostly use distinct parameters and large dashboards naturally contain repeated parameters. A second scenario arranges the same 72 parameters into 12 queries of six parameters each. It exercises the multi-parameter query path that batches Grafana Live streams onto one channel per query, while the control protects single-parameter behavior. Both cover `/myproject`, feature arrays, nested values, and the image counter. The grouped scenario has its own `grouped_scenario_semver`; it does not invalidate the shared canonical workload's `3.0.0` semver.

On the stacked dashboard PR, CI runs the same grouped browser harness once against the stacked plugin and once against the last pre-batching #90 revision. That revision receives only the later subscription-locking fix (otherwise 72 concurrent subscriptions crash) and the missing SingleValue benchmark counter; its per-parameter Live stream behavior remains unchanged. The comparison reports streams seen across all panels, streams active in the measured window, backend values processed, backend runtime per input value, and browser heap. The stacked run also verifies that all 72 parameter fields reach the browser. Shared timestamp rows intentionally differ from the per-parameter baseline; frontend row counts must not be interpreted as equal telemetry values. The pre-batching revision is fixed so later changes to its branch cannot silently alter this comparison.

By default it tests `1, 5, 10, 25, 50, 100` panels. Each panel count first runs a discarded warmup dashboard with the same shape, then records the measured dashboard. This reduces first-touch variation from Grafana app caches, datasource initialization, panel rendering paths, and Yamcs/plugin query warmup without bypassing the real Grafana path. Each measured panel count records `active_backend_streams × 15` non-empty RunStream samples by default. The dashboard time range is `now-5m` to `now`.

The harness releases its own Live subscriptions when it closes each temporary dashboard, instead of using the normal two-minute hidden-tab retention. Active-panel streaming is unchanged; this prevents one panel count's idle streams from contaminating the next measurement.

It scrolls through every panel and verifies each distinct parameter stream opened. The result records `parameter_streams_seen` separately from `backend_unique_stream_paths`: Grafana may unmount off-screen panels, so the fixed sample window is sized from streams still active after scrolling, not paths that were opened earlier and have since closed.

The backend exposes benchmark-only resource endpoints:

- `POST /api/datasources/uid/<uid>/resources/benchmark/reset`
- `GET /api/datasources/uid/<uid>/resources/benchmark/stats`

These endpoints report only aggregate benchmark counters. They are used by Playwright through authenticated Grafana datasource resource requests.

They exist only in benchmark-tagged backend builds. Normal plugin builds compile a no-op implementation that returns 404.

The Grafana benchmark CI first runs the normal plugin build, then overwrites `dist/gpx_grafana_yamcs_jaops_linux_amd64` with a benchmark-tagged backend binary. This keeps regular release builds free of benchmark endpoints while giving Grafana/Playwright a static compile-time switch.

## CI Behavior

The benchmark workflow is conditional on pull requests. Add the `run/benchmarks` label to run it and publish or update the benchmark PR comment. Unlabeled PRs skip only the base-resolution gate; downstream benchmark jobs depend on it so they are not listed individually as skipped checks.

Pushes to `main` run the benchmark jobs without PR-base comparison or PR comments. This keeps the README benchmark badge tied to a real default-branch benchmark result.

On pull requests with the `run/benchmarks` label:

- The workflow runs microbenchmarks, simulator scenarios, and Grafana benchmarks in separate jobs.
- A final report job downloads those artifacts, builds one combined report, and writes one PR comment.
- The workflow uploads the full `benchmark-output` artifact.
- The workflow creates or updates one benchmark PR comment on every run, including clean runs.
- The PR comment includes all plots in collapsible sections.
- Warn thresholds leave CI green.
- Fail thresholds fail the benchmark job.
- The workflow uploads all PNG plots to a PR artifact branch named `benchmark-artifacts-pr-<number>` so they can render in the comment.
- When available on the PR base commit, each benchmark layer also runs on the PR base commit and plots PR base curves next to HEAD curves.
- The long-term baseline refresh workflow refreshes the same three layers and opens a PR with updated checked-in baseline files.
