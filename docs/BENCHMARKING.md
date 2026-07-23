# Benchmarking

This repository has one Yamcs stream workload benchmark:

```bash
pnpm run bench
```

The benchmark assumes Yamcs quickstart is running on `localhost:8090`. By default it also starts `simulator.py` from `/tmp/yamcs-quickstart` at `1 Hz`.

## Scenario

Each scenario runs `N` concurrent Grafana stream demands against the Yamcs quickstart `myproject/realtime` processor.

For every value of `N`, the benchmark:

1. Creates `N` Grafana stream paths distributed across the default quickstart parameters.
2. Lets Yamcs quickstart warm up for `3s`.
3. Runs the measured workload for `10s`.
4. Runs one goroutine per Grafana stream.
5. Reads and clears each stream buffer every `1s`.
6. Converts read values into Grafana data frames, matching the normal RunStream read/frame/send path.
7. Records processing time, read/clear time, value freshness, memory, setup time, and RunStream per-tick wall time.

- Yamcs simulator rate: `1 Hz`
- Grafana stream read ticker: `1s`
- Freshness window: `1s`

## Outputs

`pnpm run bench` writes:

- `benchmark-output/yamcs-stream-results.json`
- `benchmark-output/yamcs-stream-results.csv`
- `benchmark-output/plots/*.png`

`pnpm run bench:report` also writes:

- `benchmark-output/benchmark-status.json`
- `benchmark-output/benchmark-comment.md`
- `benchmark-output/regression-plots/*.png`

`regression-plots/` contains only plots for metrics that crossed a warn or fail threshold. `plots/` contains all generated benchmark plots.

## Metrics

### Average read and clear time

The average wall-clock time for one Grafana stream goroutine to call `GetAndClearParameterStreamBuffer`, convert the returned values into a Grafana data frame, and finish that read/send unit of work.

This is a per-stream operation average. It should stay small as `N` grows.

### Average Yamcs listener processing time

The average time spent in the Yamcs parameter listener when a Yamcs parameter update is received and copied into the active Grafana stream buffers that requested that parameter.

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

### Average value age when read

The average age of values when a Grafana stream reads them from its buffer.

Lower is better. Values near or above `1s` mean reads are close to missing the simulator tick in which the value arrived.
But high doesn't mean lower performance, it might just be de-sync with Yamcs simulator ticks, but it should never be above `1s`.

### Average RunStream wall time Per 1s Tick

For each 1 second stream ticker interval, the benchmark measures the wall-clock span from the first RunStream read/frame/send unit starting to the last RunStream read/frame/send unit finishing across all streams.

Ideally it stays below `1s`, otherwise it might be falling behind. This might highly depend on hardware because Grafana streams are concurrent and their wall-clock span might depend on how parallel they run. 

### Stream setup time

The time to create the Grafana stream demand state and Yamcs subscriptions for `N` streams before the measured run begins.

The threshold uses setup time per stream.

## CI Behavior

The benchmark workflow is conditional. Add the `run-benchmark` label to a pull request to run it, or start it manually with `workflow_dispatch`.

When a warn or fail threshold is crossed:

- The workflow uploads the full `benchmark-output` artifact.
- The workflow uploads all PNG plots to a PR artifact branch named `benchmark-artifacts-pr-<number>`.
- The PR comment embeds only regression plots from `regression-plots/`.
- Warn thresholds leave CI green.
- Fail thresholds fail the benchmark job.

When the PR closes, the workflow deletes the `benchmark-artifacts-pr-<number>` branch. Missing branches are ignored.
