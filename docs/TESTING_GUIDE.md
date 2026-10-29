# Testing Guide for Yamcs Plugin

This guide provides instructions for testing the Yamcs Grafana plugin functionality.

## CI Test Strategy

The repository validates core behavior across frontend, backend, integration, E2E, and benchmarks:

- Frontend core query, routing, and commanding logic
- Backend stream, frame conversion, and client behavior
- Yamcs Demo 5.13.7 integration (real HTTP + WebSocket)
- Critical frontend E2E paths across app pages and core datasource resources
- Yamcs-backed stream scenario benchmark with N concurrent Grafana stream demands

### Local commands

Run frontend unit tests:

    pnpm run test:ci

Run backend unit tests:

    pnpm run test:backend

Backend unit tests intentionally do not run Yamcs-dependent integration tests. Yamcs integration tests live behind the Go `integration` build tag and only run through `pnpm run test:integration:yamcs` or the matching CI job.

Run Yamcs integration tests against Yamcs Demo (Yamcs 5.13.7 on localhost:8093;
the CI integration job starts the same fixture on localhost:8090). Follow the
[Yamcs Demo setup](./SETUP_INSTRUCTIONS.md#run-the-provisioned-yamcs-demo)
to start it locally:

    YAMCS_ADDRESS=localhost:8093 pnpm run test:integration:yamcs

Run E2E tests:

    pnpm run e2e

The Yamcs Demo dashboard E2E suite covers every tab and panel, including live
data, units and alarm thresholds. It is opt-in because the standard plugin E2E
job runs a different, smaller Yamcs quickstart. This repository provisions
only the Yamcs Demo `demo_realtime` endpoint. Start Yamcs Demo and
Grafana at `http://localhost:3000/grafana` as described in the
[setup instructions](./SETUP_INSTRUCTIONS.md#run-the-provisioned-yamcs-demo),
then with local anonymous access and Chrome installed:

    DEMO_DASHBOARDS_E2E=1 CANONICAL_ANONYMOUS=1 CANONICAL_BROWSER_CHANNEL=chrome \
      GRAFANA_URL=http://localhost:3000/grafana \
      pnpm exec playwright test tests/canonical-dashboard.spec.ts \
      --project=chromium --no-deps --workers=1

Run microbenchmarks and a simulator scenario against the already-running
Yamcs Demo:

    pip install -r scripts/benchmarks/requirements.txt
    pnpm run bench:micro
    python3 scripts/benchmarks/simulator/run.py --no-simulator

The Grafana browser benchmark runs against root-served Grafana with a
benchmark-tagged backend, **not** the normal `/grafana` dashboard setup.
See [BENCHMARKING.md](./BENCHMARKING.md) for its setup, report command,
shared 72-parameter workload, thresholds, and `run/benchmarks` label-gated CI.

## Prerequisites

1. **Yamcs Server**: Run the [Yamcs Demo fixture](https://github.com/jaops-space/grafana-yamcs-demo/tree/chore/yamcs-5.13.7) with its simulator and `/myproject`, `/features`, `/images` MDB.
2. **Grafana Instance**: Use the [local plugin setup](./SETUP_INSTRUCTIONS.md) with its Yamcs Demo datasource and dashboard provisioning.

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

The Yamcs Demo repository includes:

- Sample Yamcs configuration
- Test telemetry, events and generated images from its simulator.

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
