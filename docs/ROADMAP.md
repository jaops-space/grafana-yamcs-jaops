# Roadmap

## Partner with JAOPS

JAOPS welcomes customers who want to extend the integration around their mission workflows. The current plugin provides Yamcs telemetry visualization, replay-aware timing, and operational actions in Grafana. The opportunities below build on those capabilities; **proposed work is not a released feature or a delivery commitment**.

A useful starting point is a representative dashboard and an agreed operating envelope: parameter count and rates, replay speeds, required timestamp fidelity, retention, operator roles, and supported Grafana/Yamcs versions. Together we can scope a prototype, measure it against customer-defined acceptance criteria, and agree implementation and support priorities. The example dashboards demonstrate integration patterns; they are not evidence of mission qualification. Operational deployment requires validation against the customer's procedures and assurance requirements.

## Streaming: choose the right balance of fidelity and throughput

### Current behavior

Plot, Single Value, and Discrete queries can request multiple parameters through one Grafana Live channel per query. This reduces channel and stream-handler overhead compared with separate parameter queries; it does not combine every panel on a dashboard automatically.

For a grouped live query, each backend tick produces at most one shared row:

- Each parameter's drained batch is converted using its query type, then reduced to the conversion's last row. For Graph batches with more than three values, the conversion averages the batch and uses the **last buffered value's Yamcs `GenerationTime`**; smaller Graph batches and Single Value/Discrete batches retain the last converted value when grouped.
- The shared timestamp is the **maximum representative `GenerationTime` across all included parameter states, including carried-forward states**, not the backend clock and not an average timestamp. This describes the executable loop in `buildMultiParameterFrame`; its nearby “fresh data this tick” comment is narrower than the actual comparison.
- Parameters without fresh data contribute their last known value. A parameter that has never reported is omitted; a tick with no fresh values emits nothing. A shared row therefore does not mean every parameter was measured simultaneously.

A single-parameter query avoids the additional grouped last-row reduction, but is not an unconditional raw-sample mode: Graph batches over three values are still averaged. Initial grouped Graph history also differs from the live tick policy: it joins the sorted union of historical sample timestamps, carries values forward, and fills rows before a series' first sample with that first value. These are visualization policies, not a lossless reconstruction of event ordering.

Sources: [stream dispatch and tick cadence](../pkg/plugin/datasource_run.go), [grouped reduction, timestamp selection, and historical join](../pkg/plugin/datasource_run_multiparameter.go), [frame conversion](../pkg/utils/tools/buffer_frame.go), and [frontend query/channel construction](../src/datasource/datasource.ts).

### Partnership opportunities

Customers could help define a configuration-page grouping on/off switch or explicit **grouped / split** mode, with per-query overrides and clear migration behavior. Today grouping follows the query's parameter list; separate one-parameter targets are the existing alternative. There is no dedicated grouping or timestamp-policy selector in the [configuration editor](../src/datasource/config/ConfigEditor.tsx).

Possible options include preserving separate per-parameter time axes, exposing original generation times alongside grouped values, configurable reduction, and explicit freshness/gap indicators. Preserving more samples and timestamps can increase payload size, serialization, browser memory, and rendering work; splitting also increases Live channel demand. Those costs should be measured against the actual mission workload, not assumed to be negligible.

Grafana is the visualization layer; the configured Yamcs archive should remain the authoritative source for recorded telemetry, subject to its recording and retention settings. Timestamp precision **does matter** for correlation, event sequencing, transient analysis, and some operator decisions. A sampled or aligned dashboard view should not silently substitute for that evidence. A joint fidelity review can establish which views need exact timing and which benefit from lower-overhead summaries.

## Time synchronization and accelerated replay

### Current behavior

The [Time Sync panel](../src/time-sync-panel/TimeSyncPanel.tsx) reads the latest Yamcs time, compares it with browser time, and writes dashboard-wide relative `from`/`to` expressions through `locationService.partial`, preserving the selected duration to second-level granularity. It does not replace Grafana's clock. Its [defaults](../src/time-sync-panel/types.ts) enable synchronization for relative ranges, quantize the offset in 15-second steps, set a nominal 10-second minimum write interval, and normalize small offsets within 1.5 seconds. Duplicate-write checks and throttling reduce updates; the code exempts detected user range changes, so the interval is not an unconditional rate guarantee.

Changing the dashboard range can rerun range-dependent queries rather than merely moving a visual time axis. In this plugin, changed relative expressions change the Live path for Plot, Discrete, and Single Value queries. A new stream can therefore fetch initial history again, while the previous path remains briefly retained. An unchanged path can reuse the existing retained stream instead; it is not correct to assume every clock sample causes an archive fetch. See [query paths and retention](../src/datasource/datasource.ts) and [initial archive sampling](../pkg/plugin/datasource_subscribe.go).

At 2× or 3× Yamcs time, the offset from the browser clock keeps changing, so periodic correction remains necessary. The displayed replay-speed badge is informational; the panel aligns to received timestamps rather than extrapolating time from that speed. The [backend time stream](../pkg/plugin/datasource_run.go) emits cached Yamcs time only while its update is at most 15 seconds old. For Yamcs processors reporting a realtime replay multiplier above one, stream tick intervals are divided by that multiplier, with a 200 ms floor, when the stream starts. That mechanism is separate from dashboard range updates and does not establish continuous retiming after a speed change.

### Partnership opportunities

JAOPS and customers could benchmark range changes, archive requests, retained channel counts, and browser responsiveness at 1×, 2×, and 3×, including pause, seek, and long-running replay. A scoped Grafana Scenes integration could explore a mission-time range for selected panels and longer-lived shared query runners, separating display movement from historical refetch where supported. This would be a version-tested integration project, not an existing fix or a promise that adopting Scenes alone removes query churn. The current dashboard-wide panel remains useful for dashboards that intentionally share one timeline.

## Continuity when switching tabs or editing panels

### Current behavior

A dashboard tab change, panel edit, or query replacement can alter subscription lifetimes. In the inspected Grafana source, dashboard tabs unmount their previous content, and query-runner replacement unsubscribes the previous query. This is distinct from a background **browser** tab being throttled or suspended: not every invisible panel loses its subscription, and a stopped repaint does not by itself prove telemetry loss. See [Grafana tab lifecycle][grafana-tabs] and [query replacement][grafana-query-runner].

The plugin already mitigates short interruptions. Its [frontend retention layer](../src/datasource/datasource.ts) keeps an unused Live path subscribed for **two minutes**, shares it within the data-source instance, and replays its latest buffered response to returning subscribers. The Grafana streaming buffer is bounded by the configured `bufferMaxLength` (default 5,000); this is not unlimited history or durable storage.

On the backend, [parameter streams share a ring buffer](../pkg/source/endpoint_parameter.go) with individual read cursors. A new stream starts at the ring's current position, not its old backlog. A reader that falls behind capacity loses the oldest pending values and logs a warning; the last withdrawn demand can remove the Yamcs subscription. Retention expiry, browser reload/suspension, disconnection, or a changed query path can therefore require recovery beyond simply redrawing the panel.

Graph subscriptions already seed from Yamcs historical samples. Grouped Graph queries use the join described above; grouped Single Value and Discrete queries seed from current values instead. Existing backfill is consequently query-type dependent and not a guarantee that every missed live value is recovered. Carry-forward can also conceal an interval without fresh measurements. Sources: [initial frames](../pkg/plugin/datasource_subscribe.go) and [grouped history/current-value selection](../pkg/plugin/datasource_run_multiparameter.go).

### Partnership opportunities

Customer-supported work could add configurable inactive retention, bounded shared scene data runners or caches, explicit discontinuity indicators, and archive backfill after reconnect or reactivation. A robust backfill design would define overlap/deduplication, archive availability, sampled versus raw recovery, and behavior across replay seeks.

Keeping more streams warm consumes memory, bandwidth, backend work, and Live channel capacity even when no panel is visible. Shared runners can reduce duplicate work, but require compatible queries, explicit ownership and teardown, and isolation between data sources and sessions. Acceptance tests should distinguish values never received, values evicted from buffers, archive gaps, and values received but not displayed.

## Canvas and specialized mission panels

Grafana Canvas already provides a strong foundation for telemetry schematics, data-driven styling, links, actions, and API buttons. JAOPS can partner with customers on the workflow-specific integration around these features rather than treating all customization as a missing Grafana capability.

| Opportunity | Current foundation | Proposed customer-facing outcome |
| --- | --- | --- |
| Conditional line animation | Canvas supports animated connections. In the inspected implementation, color, width, and direction can be data-driven, while animation reads the configured `lineStyle.animate` flag. | Evaluate telemetry-conditioned start/stop, speed, and direction with clear stale/unknown states. Determine whether configuration, an upstream contribution, or a specialized panel is the maintainable route. |
| Embedded sparklines and custom graphs | The inspected built-in Canvas element registry has no general sparkline or embedded time-series element; Grafana already offers separate graph panels. | Place local trends beside subsystem indicators, with agreed time alignment, retention, interaction, and rendering budgets. Evaluate supported extension options rather than promising arbitrary panel embedding. |
| Canvas command controls | Canvas already has action editing, including one-click actions, and an API Button element. JAOPS already issues Yamcs commands through its data-source resource endpoint and dedicated Commanding panel. | Develop and validate a supported Canvas-to-JAOPS command integration: argument handling, confirmation policy, permissions, and command-history feedback. This is integration work, not a claim that Grafana cannot issue actions. An HTTP success is not proof of spacecraft execution. |
| Specialized JAOPS panels | Existing telemetry, commanding, alarms, links, and timing integrations provide reusable building blocks. | Co-design mission-specific views such as subsystem synoptics, correlated event timelines, or procedure-oriented controls when ordinary dashboard composition is insufficient. Define usability, accessibility, failure behavior, and mission-assurance acceptance criteria before operational use. |

Sources: Grafana [connection styling/animation][grafana-connections], [Canvas element registry][grafana-registry], [API Button][grafana-button], and [action editor][grafana-actions]; JAOPS [command submission](../src/commanding-panel/hooks/useCommandSubmit.ts) and [resource routes](../pkg/plugin/resources.go).

The Grafana references below are pinned to the inspected source revision, not a compatibility guarantee for every installed release. Deployment-version checks and end-to-end testing are part of scoping these opportunities.

## Authorization for Yamcs actions

Current behavior is unchanged: anyone who can use the data source, including Grafana Viewers, can invoke telecommand, alarm, and link actions. The actions execute through the data source's configured Yamcs credentials; the backend does not currently enforce a Grafana role. This keeps shared and view-oriented dashboards working, but means dashboard access can include operational control.

Revisit whether to support a configurable minimum Grafana role or separate permissions for command issuance and alarm/link actions. Any change should make the effective permissions clear to dashboard users, account for shared dashboards, and preserve an explicit compatibility path for existing deployments. No role restriction is being introduced in this release.

[grafana-tabs]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/features/dashboard-scene/scene/layout-tabs/TabItemRenderer.tsx
[grafana-query-runner]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/features/query/state/PanelQueryRunner.ts
[grafana-connections]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/plugins/panel/canvas/utils.ts
[grafana-registry]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/features/canvas/registry.ts
[grafana-button]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/features/canvas/elements/button.tsx
[grafana-actions]: https://github.com/grafana/grafana/blob/9479ec6b92b1c583f7f126b747cb28c4b1c253be/public/app/plugins/panel/canvas/editor/element/ActionsEditor.tsx
