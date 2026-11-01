# Yamcs Demo

This directory is mounted by `.config/docker-compose-base.yaml`. The canonical source is `Example Demo/yamcs_quickstart_demo_dashboard.json`, provisioned by `yamcs-dashboards.yaml`. Save changes to the JSON: Grafana UI saves do not update that file and can temporarily override it.

Open [Yamcs Demo](http://localhost:3000/d/0708823c-c106-407b-9861-f64283746167/yamcs-demo) or its [FHD presentation view](http://localhost:3000/d/0708823c-c106-407b-9861-f64283746167/yamcs-demo?kiosk=true). The existing UID is preserved.

## Mission views

The compact header combines the JAOPS static-image panel, an HTML navigation bar and Yamcs time synchronization. Mission status is inside Overview rather than repeated above every tab. Six native Grafana tabs contain 65 panels.

| View         | Demonstration                                                                                                                                                     |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Overview     | Rebuilt spacecraft Canvas, telemetry-bound fault borders and readings, batteries, gyro history, state timelines, alarms, mission clock and packet reception.      |
| Telemetry    | Ground track, orbit coordinates, contact/eclipse history, MDB alarm limits, aggregate members and battery selection.                                              |
| Mission data | Integer, boolean, string, enumeration, engineering units, arrays, arrays of aggregates, nested aggregates, aggregate-array members, histogram and XY correlation. |
| Commanding   | Live image telemetry, dual battery commands, typed payload commands, editable gain variable, command history and queue counters.                                  |
| Operations   | Link enable/disable/reset controls, JVM telemetry, events and alarm acknowledgement/shelving controls.                                                            |
| Diagnostics  | Plugin stream demands, Yamcs subscriptions, read-only link table and explanations of data flow.                                                                   |

All **eight JAOPS panel plugins** are represented: commanding, variable setting, command history, alarms, links, time sync, telemetric images and static images. All **12 datasource query modes** are configured: `single`, `plot`, `discrete`, `time`, `image`, `commanding`, `command-history`, `events`, `alarms`, `links`, `demands`, `subscriptions`.

Native Grafana coverage includes Canvas, stats/sparklines, gauges, bar gauges, time series, state timelines, tables, Geomap, histogram, XY chart, text/HTML, tabs, variables, data links, transformations, units/value mappings, thresholds, shared cursor, relative time and panel time overrides. The navbar opens Yamcs, Explore and the JAOPS app. This is not a claim that every Grafana or Yamcs subsystem is implemented in the dashboard: alert-rule administration, access control, CFDP, replay creation, timeline authoring and mission-specific automation require their own workflows and configuration. See the [roadmap](../../docs/ROADMAP.md).

## Simulator controls and interpretation

**This is an interactive software demonstrator, not a flight-qualified control console.** Command buttons issue real Yamcs commands to the configured `demo_realtime` endpoint. Link and alarm actions also change Yamcs state. Do not point this demonstration at operational hardware without configuring permissions and appropriate safeguards.

- The Battery variable selects both telemetry and battery-command arguments. Changing it alone sends nothing.
- The JAOPS variable-setting panel edits payload gain. Apply resolves that variable and the selected enum mode into typed MDB arguments.
- `ConfigurePayload` has a five-second transmission constraint requiring `Payload_Status = True`, and an MDB completion verifier. A false status produces a rejection visible in history. Do not bypass the constraint to make a green demonstration.
- Queued, released, sent and completed are distinct stages. A successful submission response is not proof of delivery or execution. The simulator receives UDP commands but replays spacecraft telemetry independently; battery commands do not change the replay.
- Camera frames are generated test images, explicitly labeled by the simulator. They demonstrate parameter-driven image updates, not a physical or physics-simulated camera.
- Canvas connections illustrate power topology, not measured flow. Fault colors, readings, units and alarm limits come from actual fields. JVM memory keeps its reported KB unit.
- The default range is 15 minutes with a 30-second refresh plus live streaming. Enum panels use a 30-second override; command history uses seven days. Command-history rows currently use browser-local time; the mission clock and other time panels use UTC.
- Esri basemap tiles need network access. The plotted coordinates are recorded telemetry, not an orbit prediction.

The fixture's `DemoPowerMatrix` root is a flattened array despite its 2×4 declaration; its multidimensional leaf reads do not reliably return engineering values. The dashboard uses verified cell-array and channel-aggregate paths instead of displaying fabricated zeros.

## Local fixture

See the [setup instructions](../../docs/SETUP_INSTRUCTIONS.md#run-the-provisioned-yamcs-demo). The local layout uses Grafana `:3000`, Yamcs `:8093`, image HTTP `:8092` and simulator UDP commands `:10026`:

```sh
docker compose \
  -f ../grafana-yamcs-demo/docker/docker-compose.yml \
  -f .config/demo-ports.yml up -d --build
```

The matching demo fixture must include [de2feb0](https://github.com/jaops-space/grafana-yamcs-demo/commit/de2feb0), which adds `YAMCS_TC_PORT` support. The local override routes commands through `host.docker.internal:10026`; this avoids a startup failure when the simulator's Docker DNS name is not yet available. Its advertised image URLs use the actual `:8092` mapping. It does not change the fixture's default ports. The updated simulator also retries logged telemetry-send failures so a temporary Docker DNS outage during a Yamcs restart does not permanently stop packet replay.

If an older running Yamcs uplink already failed during startup, an **udp-out Disable → Enable** cycle reopens it. An `OK` label alone is insufficient: verify that its sent counter advances after a battery command.

## Streaming limits

`Streaming channel error: limit exceeded` is Centrifuge's default **128 active channels per browser client** in Grafana Live, not `[live] max_connections`. Unique parameter/query/range combinations can create separate channels. Reduce simultaneous subscriptions, share identical queries and use supported grouped queries; increasing the connection limit does not raise the per-client channel cap.

Inactive tabs or panel editing can interrupt subscriptions and produce visible gaps. An `expired` error is a separate token/subscription lifecycle issue; refreshing can restore data but is not a backend fix. Proposed continuity, timestamp-fidelity and Scenes work is described in the [roadmap](../../docs/ROADMAP.md).

## End-to-end contract

`tests/canonical-dashboard.spec.ts` compares the deployed/source contracts, inventories every panel/query type, inspects real Grafana Scene frames for every panel, verifies values after transformations, validates field types/units/alarm limits, checks image decoding and containment, tests tab resumption and variables, and checks all six layouts at 1920×1080.

The ordinary tests block all Yamcs mutations, including accidental initialization requests. The explicit simulator-command test verifies the local endpoint, changes gain through the variable panel, submits typed commands, checks history and requires the UDP link's sent counter to advance. It never disables a live telemetry link or acknowledges alarms automatically.

```sh
mkdir -p benchmark-output/canonical-e2e/runtime
DEMO_DASHBOARDS_E2E=1 CANONICAL_BROWSER_CHANNEL=chrome CANONICAL_ANONYMOUS=1 \
  CANONICAL_SIMULATOR_COMMANDS=1 GRAFANA_URL=http://localhost:3000 \
  TMPDIR="$PWD/benchmark-output/canonical-e2e/runtime" \
  pnpm exec playwright test tests/canonical-dashboard.spec.ts \
  --project=chromium --no-deps --workers=1 --reporter=line \
  --output=benchmark-output/canonical-e2e/test-results
```

Omit `CANONICAL_SIMULATOR_COMMANDS` for a mutation-blocked run. Screenshots are attached to the FHD test; presentation captures are in the ignored `benchmark-output/screenshots/` directory.

The aggregate/member metadata regression remains covered by `pkg/utils/tools/parameter_path_test.go`: combined tokens such as `temperatures[0]` preserve leaf units and alarm thresholds. E2E expectations inspect datasource metadata independently of display overrides.
