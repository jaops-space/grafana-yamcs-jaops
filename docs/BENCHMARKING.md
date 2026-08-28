# Benchmarking

This repository has three benchmark layers:

- Microbenchmarks for isolated frame/buffer processing helpers.
- Yamcs simulator scenarios for backend streaming behavior against live quickstart data.
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

The simulator benchmark assumes Yamcs quickstart is running on `localhost:8090`. By default it also starts `simulator.py` from `/tmp/yamcs-quickstart` at `1 Hz`.

The Grafana panel benchmark assumes:

- Grafana is running on `http://localhost:3000`;
- the Linux backend binary in `dist` is built with `go build -tags "arrow_json_stdlib benchmark"` so benchmark-only resource endpoints are compiled in;
- the provisioned datasource UID is `jaops-yamcs-main`;
- Yamcs quickstart and its simulator are running.

## Files

Benchmark code is organized by layer:

- `scripts/benchmarks/micro/run.py`: isolated Go microbenchmarks.
- `scripts/benchmarks/simulator/run.py`: Yamcs quickstart simulator scenario runner.
- `scripts/benchmarks/simulator/scenario.go`: Go workload used by the simulator runner.
- `tests/benchmark/streaming.benchmark.spec.ts`: browser/Grafana benchmark.
- `scripts/benchmarks/report.py`: the single report builder. It reads raw layer JSON outputs directly and writes the PR comment, status, and plots.
- `scripts/benchmarks/common/`: shared implementation helpers and the central metric catalog. This keeps each layer runner small and avoids copying plot/report/baseline logic between layers.

## Simulator scenario

Each scenario runs `N` concurrent Grafana stream demands against the Yamcs quickstart `myproject/realtime` processor.

For every value of `N`, the benchmark:

1. Runs one discarded warmup scenario so Yamcs/plugin paths are warmed before measured scenarios.
2. Creates `N` Grafana stream paths distributed across the default quickstart parameters.
3. Lets Yamcs quickstart warm up for `3s`.
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

Benchmark result files include a `metric_semantics_version`. Baseline percentage comparisons are only computed when the current run and baseline use the same metric semantics version. This prevents misleading comparisons when a metric definition changes, such as changing from averages to medians or replacing a wall-clock span with busy work.

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

The simulator scenario measures the plugin backend against live Yamcs quickstart data, but without rendering Grafana panels in a browser. It uses the real stream buffers and RunStream read/frame/send path.

The PR comment currently plots:

- live memory used during the run;
- total memory allocated during the run;
- values read per second from stream buffers;
- percentage of values read within the same 1 second simulator tick;
- median total RunStream busy time per 1 second tick;
- stream setup time.

The most useful signals here are throughput, freshness, memory, and RunStream busy time. RunStream busy time sums actual backend read/frame/send work per tick, so it avoids treating goroutine ticker scheduling delay as plugin work.

### Grafana scenario

The Grafana scenario measures the full browser + Grafana dashboard + Grafana Live + plugin backend path. It creates temporary dashboards with Timeseries panels over Yamcs quickstart parameters, warms each panel count once, then records the measured run.

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

It uses one mixed-parameter scenario. Panels cycle through the known Yamcs quickstart parameters, so small dashboards mostly use distinct parameters and large dashboards naturally contain repeated parameters. This keeps the CI comment simple while still allowing shared-stream changes to show up in HEAD-vs-PR-base comparisons.

By default it tests `1, 5, 10, 25, 50, 100` panels. Each panel count first runs a discarded warmup dashboard with the same shape, then records the measured dashboard. This reduces first-touch variation from Grafana app caches, datasource initialization, panel rendering paths, and Yamcs/plugin query warmup without bypassing the real Grafana path. Each measured panel count records a fixed number of non-empty backend RunStream samples: `panel_count × 15` samples by default. The dashboard time range is `now-5m` to `now`.

The backend exposes benchmark-only resource endpoints:

- `POST /api/datasources/uid/<uid>/resources/benchmark/reset`
- `GET /api/datasources/uid/<uid>/resources/benchmark/stats`

These endpoints report only aggregate benchmark counters. They are used by Playwright through authenticated Grafana datasource resource requests.

They exist only in benchmark-tagged backend builds. Normal plugin builds compile a no-op implementation that returns 404.

The Grafana benchmark CI first runs the normal plugin build, then overwrites `dist/gpx_grafana_yamcs_jaops_linux_amd64` with a benchmark-tagged backend binary. This keeps regular release builds free of benchmark endpoints while giving Grafana/Playwright a static compile-time switch.

## CI Behavior

The benchmark workflow is conditional. Add the `run/benchmarks` label to a pull request to run it and publish or update the benchmark PR comment, or start it manually with `workflow_dispatch`.

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
