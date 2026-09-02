# Testing Guide for Yamcs Plugin

This guide provides instructions for testing the Yamcs Grafana plugin functionality.

## CI Test Strategy

The repository validates core behavior across frontend, backend, integration, E2E, and benchmarks:

- Frontend core query, routing, and commanding logic
- Backend stream, frame conversion, and client behavior
- Yamcs quickstart integration (real HTTP + WebSocket)
- Critical frontend E2E paths across app pages and core datasource resources
- Yamcs-backed stream scenario benchmark with N concurrent Grafana stream demands

### Local commands

Run frontend unit tests:

    pnpm run test:ci

Run backend unit tests:

    pnpm run test:backend

Backend unit tests intentionally do not run Yamcs-dependent integration tests. Yamcs integration tests live behind the Go `integration` build tag and only run through `pnpm run test:integration:yamcs` or the matching CI job.

Run Yamcs integration tests (requires Yamcs running on localhost:8090):

    pnpm run test:integration:yamcs

Run E2E tests:

    pnpm run e2e

Run Yamcs-backed stream scenario benchmarks:

    pip install -r scripts/benchmarks/requirements.txt
    pnpm run bench

Refresh the checked-in long-term benchmark baseline:

    pnpm run bench:baseline

Benchmark CI is conditional. On pull requests, add the `run/benchmarks` label to run the benchmark job and create or update the benchmark PR comment with all plots. The job also supports manual `workflow_dispatch` runs from GitHub Actions. Warn thresholds keep CI green; fail thresholds fail the benchmark job.

This starts Yamcs quickstart's `simulator.py` by default, opens N concurrent Grafana stream demands against the quickstart `myproject/realtime` processor, and reads each stream buffer from its own goroutine. Performance plots use concurrent Grafana streams (`N`) on the x-axis and one metric per file:

- live memory growth and total allocated bytes
- values read per second
- percentage of values read within the 1 second freshness window
- median RunStream busy work per 1s tick
- stream setup time

By default the benchmark matches Yamcs quickstart's simulator cadence: simulator rate `1 Hz`, stream read interval `1s`, and freshness window `1s`. A value counts as fresh when it is read from its Grafana stream buffer within one second of being placed there by the Yamcs listener.

See [BENCHMARKING.md](./BENCHMARKING.md) for the full scenario description, metric definitions, thresholds, and CI behavior.

Invariant and diagnostic metrics such as empty read percentage, legacy RunStream wall span, and scenario wall time remain in JSON/CSV for sanity checks, but are not primary plots. Plots show the current result, the PR base benchmark when available, and the checked-in long-term baseline when available. Baseline percentage changes are reported as median, maximum negative, and maximum positive informational values and do not affect WARN or FAIL status.

Thresholds are rendered directly on plots where they apply and are also included in the JSON output. Distant thresholds are hidden so plots can zoom into observed behavior. Use `--fail-on-threshold` to make failed thresholds exit non-zero:

    pnpm run bench -- --fail-on-threshold

Benchmark output files are written by layer:

- `microbenchmarks/microbenchmarks.json`
- `simulator/simulator.json`
- `simulator/simulator.csv`
- `grafana/grafana.json`
- `plots/*.png`
- `benchmark-status.json`
- `benchmark-comment.md`

## Prerequisites

1. **Yamcs Server**: You need a running Yamcs server instance
    - clone the quickstart from: https://github.com/jaops-space/yamcs-quickstart
    - start Yamcs server: `./mvnw yamcs:run`
    - run the simulator for basic spacecraft data: `./simulator.py`
2. **Grafana Instance**: Grafana 10.4.0 or later

## Quick Setup for Testing

### 1. Install the Plugin

1. Download the plugin ZIP from the release
2. Extract to your Grafana plugins directory
3. Restart Grafana
4. Enable the plugin in Configuration > Plugins

### 2. Configure Yamcs Connection

1. Go to Configuration > Data Sources
2. Add new Yamcs data source
3. Configure connection settings:
    - **URL**: Your Yamcs server URL (e.g., `http://localhost:8090`)
    - **Instance**: Yamcs instance name
    - **Credentials**: If authentication is required

### 3. Test Basic Functionality

## Sample Data

The yamcs-quickstart repository includes:

- Sample Yamcs configuration
- Test telemetry data and simulators (for telemetry, images, etc)

This grafana-yamcs-jaops repository includes:

- Example dashboards in `provisioning/`

#### Telemetry Display

1. Create a new dashboard
2. Add a graph panel
3. Select Yamcs as data source
4. Query telemetry parameters
5. Verify data display

#### Commanding

1. Navigate to the Commanding page in the plugin
2. Configure command buttons
3. Test command execution
4. Verify command history

#### Image Panels

1. Add an Image Panel
2. Configure with static image
3. Overlay telemetry data
4. Test real-time updates

## Troubleshooting

### Common Issues

1. **Connection Failed**: Check Yamcs server is running and accessible
2. **No Data**: Verify Yamcs instance name and parameter names
3. **Commands Not Working**: Check command definitions and permissions

### Logs

- Check Grafana logs for plugin errors
- Yamcs server logs for connection issues
- Browser console for frontend errors

## Expected Test Results

- Plugin loads without errors
- Data source connects to Yamcs
- Telemetry data displays correctly
- Commands execute successfully
- Image panels overlay data properly
- Real-time updates work smoothly
