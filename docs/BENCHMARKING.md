# Benchmarking

This repository has one Yamcs stream workload benchmark:

```bash
pnpm run bench
```

Refresh the checked-in long-term baseline with the manual GitHub Actions workflow named `Refresh benchmark long-term baseline`.

`pnpm run bench:baseline` writes the checked-in baseline path, but it is intentionally guarded so it fails outside CI unless `--allow-local-baseline` is passed for a local diagnostic run.

The benchmark assumes Yamcs quickstart is running on `localhost:8090`. By default it also starts `simulator.py` from `/tmp/yamcs-quickstart` at `1 Hz`.

## Scenario

Each scenario runs `N` concurrent Grafana stream demands against the Yamcs quickstart `myproject/realtime` processor.

For every value of `N`, the benchmark:

1. Runs one discarded warmup scenario so Yamcs/plugin paths are warmed before measured scenarios.
2. Creates `N` Grafana stream paths distributed across the default quickstart parameters.
3. Lets Yamcs quickstart warm up for `3s`.
4. Runs the measured workload for `10s`.
5. Runs one goroutine per Grafana stream.
6. Reads and clears each stream buffer every `1s`.
7. Converts read values into Grafana data frames, matching the normal RunStream read/frame/send path.
8. Records median processing time, median read/clear time, freshness, memory, setup time, empty-read validity data, and median RunStream busy time per tick.

- Yamcs simulator rate: `1 Hz`
- Grafana stream read ticker: `1s`
- Freshness window: `1s`
- Discarded warmup scenario: `25` streams for `3s`

## Outputs

`pnpm run bench` writes:

- `benchmark-output/yamcs-stream-results.json`
- `benchmark-output/yamcs-stream-results.csv`
- `benchmark-output/plots/*.png`

`pnpm run bench:report` also writes:

- `benchmark-output/benchmark-status.json`
- `benchmark-output/benchmark-comment.md`
- `benchmark-output/regression-plots/*.png`

`plots/` contains all generated benchmark plots. `regression-plots/` is retained as a small machine-readable subset for metrics that crossed a warn or fail threshold.

## Baselines

Benchmark plots can show three curves:

- Blue: the current benchmark result.
- Slate: the PR base commit before the PR changes, when CI can benchmark it.
- Green dashed: the checked-in long-term baseline from `scripts/benchmarks/baselines/long-term/yamcs-stream-results.json`.

The long-term baseline is intentionally committed to the repository so performance drift remains visible even when a PR base benchmark is unavailable or changes too often to be a useful historical reference. It should be refreshed only from GitHub Actions so the baseline is tied to the CI environment, not a developer workstation. Its metadata lives next to the result file in `scripts/benchmarks/baselines/long-term/metadata.json`.

Benchmark result files include a `metric_semantics_version`. Baseline percentage comparisons are only computed when the current run and baseline use the same metric semantics version. This prevents misleading comparisons when a metric definition changes, such as changing from averages to medians or replacing a wall-clock span with busy work.

The report also shows informational median, maximum negative, and maximum positive percentage change for each plotted metric against each compatible baseline. These percentage changes do not affect WARN or FAIL status.

## Metrics

### Implementation notes

The benchmark intentionally measures the normal backend work path rather than a synthetic no-op path:

- read/clear uses the same endpoint stream buffers as plugin streaming;
- frame conversion uses preallocated value/time slices to avoid measuring avoidable slice growth overhead;
- listener processing measures Yamcs parameter fan-out into active stream buffers;
- RunStream busy time measures actual accumulated read/frame/send work, while avoiding the scheduler-sensitive wall-span artifact caused by many independent 1s tickers.

### Median read and clear time

The median wall-clock time for one Grafana stream goroutine to call `GetAndClearParameterStreamBuffer`, convert the returned values into a Grafana data frame, and finish that read/send unit of work.

This is a per-stream operation median. It should stay small as `N` grows. The endpoint lookup uses a read lock and each stream buffer uses its own stream lock, so concurrent reads of different stream buffers do not need the endpoint-wide write lock.

### Median Yamcs listener processing time

The median time spent in the Yamcs parameter listener when a Yamcs parameter update is received and copied into the active Grafana stream buffers that requested that parameter.

This measures the backend fan-out cost of incoming Yamcs data.

### Live memory used during run

The difference in live heap allocation between the start and end of the measured scenario.

This is the memory still retained after the scenario, not the total amount allocated over time.

### Total memory allocated during the run

The total bytes allocated during the measured scenario according to Go runtime memory stats.

This can grow even when live memory stays flat, because short-lived allocations are counted too.

### Values read per second from buffers

The number of parameter values read from all Grafana stream buffers per second.

The plot title is:
`Values read per second from buffers by N Grafana streams`

Because the default simulator runs at `1 Hz`, this value should scale with the number of active streams until backend work starts delaying stream reads.

### Values read within the same 1s tick

The percentage of values read before the next 1 second simulator update.

This is the main stalling signal. If this drops, Grafana stream reads are falling behind the 1 Hz Yamcs simulator cadence.

### Empty read percentage

The percentage of read/clear operations that found no buffered values.

This is a validity signal, not a primary performance plot. With the default 1 Hz simulator and 1s read ticker, valid runs should normally have few or no empty reads. A high empty-read percentage usually means the simulator/Yamcs data path is not feeding values as expected, or the benchmark is reading more often than Yamcs produces updates.

### Median RunStream busy time per 1s tick

For each 1 second stream ticker interval, the benchmark sums the actual read/frame/send durations performed by all RunStream goroutines in that tick bucket.

The reported value is the median of those per-tick busy totals. This measures total backend work per tick without treating unsynchronized goroutine ticker phase spread as plugin work.

The raw legacy wall-span value is still present in JSON as `avg_tick_runstream` for diagnostics, but the CI plot and threshold use `median_tick_runstream_busy`.

### Stream setup time

The time to create the Grafana stream demand state and Yamcs subscriptions for `N` streams before the measured run begins.

The threshold uses setup time per stream. Setup time is useful diagnostic data, but it can be affected by Yamcs JVM warmup, parameter metadata cache state, and subscription path warmup. The benchmark runs a discarded warmup scenario before measured scenarios to reduce this bias.

## CI Behavior

The benchmark workflow is conditional. Add the `run-benchmark` label to a pull request to run it and publish or update the benchmark PR comment, or start it manually with `workflow_dispatch`.

On pull requests with the `run-benchmark` label:

- The workflow uploads the full `benchmark-output` artifact.
- The workflow creates or updates one benchmark PR comment on every run, including clean runs.
- The PR comment includes all plots in collapsible sections.
- Warn thresholds leave CI green.
- Fail thresholds fail the benchmark job.
- The workflow uploads all PNG plots to a PR artifact branch named `benchmark-artifacts-pr-<number>` so they can render in the comment.
