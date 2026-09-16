// Measures the multi-parameter-stream architecture (RunMultiParameterStream)
// end-to-end, through a real browser + real Grafana Live + real gRPC plugin
// backend + real Yamcs, against the per-parameter fan-out (RunParameterStream)
// used for single-parameter queries - for the same 20 parameters, so the
// comparison is apples-to-apples on total parameter count rather than panel
// count. See src/datasource/datasource.ts (query()) and
// pkg/plugin/datasource_run_multiparameter.go for the implementation this
// exercises.
import fs from 'node:fs';
import path from 'node:path';

import { expect, test } from '@grafana/plugin-e2e';

const datasourceUid = process.env.GRAFANA_BENCHMARK_DATASOURCE_UID ?? 'yamcs-demo';
const datasourceType = 'jaops-yamcs-datasource';
const endpoint = process.env.GRAFANA_BENCHMARK_ENDPOINT ?? 'demo_realtime';
const benchmarkDurationMs = Number(process.env.GRAFANA_BENCHMARK_DURATION_MS ?? '10000');
const outputPath = process.env.GRAFANA_BENCHMARK_OUTPUT ?? 'benchmark-output/grafana/grafana-batched.json';

const parameters = [
    '/drone/Battery1_Voltage',
    '/drone/Battery2_Voltage',
    '/drone/Battery1_Temp',
    '/drone/Battery2_Temp',
    '/drone/Detector_Temp',
    '/drone/Latitude',
    '/drone/Longitude',
    '/drone/Height',
    '/drone/BatteryPackVoltage',
    '/drone/BatteryPackCurrent',
    '/drone/BatteryPower',
    '/drone/BatteryStateOfCharge',
    '/drone/BatteryTemperature',
    '/drone/ElapsedSeconds',
    '/drone/CCSDS_Packet_ID',
    '/drone/CCSDS_Packet_Sequence',
    '/drone/CCSDS_Packet_Length',
    '/drone/MissionElapsedTime',
    '/drone/BatteryEnergyRemaining',
    '/drone/Motor1Temperature',
];

type BenchmarkStats = {
    run_stream_runtime_ns: number;
    run_stream_median_runtime_ns: number;
    run_stream_samples: number;
    run_stream_calls: number;
    frames_sent: number;
    values_sent: number;
    unique_stream_paths: number;
    window_seconds: number;
    backend_heap_alloc_bytes: number;
};

function basePanelFields(id: number) {
    return {
        id,
        gridPos: { h: 8, w: 12, x: 0, y: 0 },
        options: {
            legend: { displayMode: 'hidden', placement: 'bottom', showLegend: false },
            tooltip: { mode: 'single', sort: 'none' },
        },
        fieldConfig: { defaults: {}, overrides: [] },
    };
}

// Multi-observer baseline: N panels, one parameter each - the shipped fan-out.
function buildObserverDashboard() {
    const uid = `jybench-observer-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    return {
        dashboard: {
            id: null,
            uid,
            title: 'JAOPS benchmark observer (N panels)',
            tags: ['jaops-benchmark'],
            timezone: 'browser',
            schemaVersion: 39,
            version: 0,
            refresh: '',
            time: { from: 'now-5m', to: 'now' },
            panels: parameters.map((parameter, index) => ({
                ...basePanelFields(index + 1),
                title: `Parameter ${index + 1}`,
                type: 'timeseries',
                datasource: { type: datasourceType, uid: datasourceUid },
                gridPos: { h: 8, w: 6, x: (index % 4) * 6, y: Math.floor(index / 4) * 8 },
                maxDataPoints: 200,
                targets: [
                    {
                        refId: 'A',
                        datasource: { type: datasourceType, uid: datasourceUid },
                        endpoint,
                        parameter,
                        type: 'plot',
                        fields: [],
                        asVariable: false,
                        customVariableString: false,
                        endpointVariable: '$endpoint',
                    },
                ],
            })),
        },
        folderId: 0,
        overwrite: true,
    };
}

// Multi-parameter-stream: 1 panel, all N parameters on a single query - the
// datasource batches this onto one Live channel/backend goroutine
// (RunMultiParameterStream) automatically since parameters.length > 1.
function buildBatchedDashboard() {
    const uid = `jybench-batched-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    return {
        dashboard: {
            id: null,
            uid,
            title: 'JAOPS benchmark batched (1 panel)',
            tags: ['jaops-benchmark'],
            timezone: 'browser',
            schemaVersion: 39,
            version: 0,
            refresh: '',
            time: { from: 'now-5m', to: 'now' },
            panels: [
                {
                    ...basePanelFields(1),
                    title: 'All parameters (batched)',
                    type: 'timeseries',
                    datasource: { type: datasourceType, uid: datasourceUid },
                    gridPos: { h: 16, w: 24, x: 0, y: 0 },
                    maxDataPoints: 200,
                    targets: [
                        {
                            refId: 'A',
                            datasource: { type: datasourceType, uid: datasourceUid },
                            endpoint,
                            parameter: parameters[0],
                            parameters,
                            type: 'plot',
                            fields: [],
                            asVariable: false,
                            customVariableString: false,
                            endpointVariable: '$endpoint',
                        },
                    ],
                },
            ],
        },
        folderId: 0,
        overwrite: true,
    };
}

async function createDashboard(request: any, payload: any): Promise<string> {
    const createResponse = await request.post('/api/dashboards/db', { data: payload });
    expect(createResponse.ok(), `dashboard create failed: ${createResponse.status()} ${await createResponse.text()}`).toBeTruthy();
    const createBody = await createResponse.json();
    return createBody.uid ?? payload.dashboard.uid;
}

async function resetBackendStats(request: any, targetSamples = 0): Promise<void> {
    const response = await request.post(`/api/datasources/uid/${datasourceUid}/resources/benchmark/reset`, {
        data: { target_samples: targetSamples },
    });
    await expect(response).toBeOK();
}

async function readBackendStats(request: any): Promise<BenchmarkStats> {
    const response = await request.get(`/api/datasources/uid/${datasourceUid}/resources/benchmark/stats`);
    await expect(response).toBeOK();
    return response.json();
}

async function waitForStreamsQuiet(request: any): Promise<void> {
    await expect
        .poll(
            async () => {
                await resetBackendStats(request);
                await new Promise((resolve) => setTimeout(resolve, 1500));
                return (await readBackendStats(request)).frames_sent;
            },
            { timeout: 30_000 }
        )
        .toBe(0);
}

// Grafana only mounts (and therefore only opens the Live subscription for)
// panels that have actually scrolled into the viewport, so the observer
// dashboard's 20 panels need to be scrolled through before its
// unique_stream_paths reflects all of them - otherwise this would silently
// undercount the multi-observer side and bias the comparison in its favor.
// The single-panel batched dashboard doesn't need this (it's always visible).
async function scrollThroughAllPanels(page: any, expectedPanelCount: number): Promise<void> {
    const seen = new Set<string>();
    const deadline = Date.now() + 30_000;

    const collect = async () => {
        const ids = await page.evaluate(() => {
            const found = new Set<string>();
            for (const element of document.querySelectorAll('[data-panelid]')) {
                const id = element.getAttribute('data-panelid');
                if (id) {
                    found.add(id);
                }
            }
            for (const element of document.querySelectorAll('[data-viz-panel-key^="panel-"]')) {
                const id = element.getAttribute('data-viz-panel-key')?.replace(/^panel-/, '');
                if (id) {
                    found.add(id);
                }
            }
            return [...found];
        });
        for (const id of ids) {
            seen.add(id);
        }
    };

    // Mirrors the main streaming.benchmark.spec.ts's waitForAllPanelsSeen: an
    // outer retry (the page may still be loading when the first pass starts,
    // so a single early "already at the bottom" reading is not trustworthy)
    // around an inner scroll-and-collect pass.
    while (Date.now() < deadline && seen.size < expectedPanelCount) {
        await page.evaluate(() => window.scrollTo(0, 0));
        let lastScrollY = -1;

        for (let i = 0; i < Math.max(80, expectedPanelCount * 2) && seen.size < expectedPanelCount; i++) {
            await collect();

            const state = await page.evaluate(() => {
                window.scrollBy(0, Math.floor(window.innerHeight * 0.85));
                return { scrollY: window.scrollY, scrollHeight: document.documentElement.scrollHeight, innerHeight: window.innerHeight };
            });
            await page.waitForTimeout(100);

            if (state.scrollY === lastScrollY || state.scrollY + state.innerHeight >= state.scrollHeight) {
                await collect();
                break;
            }
            lastScrollY = state.scrollY;
        }
    }
    await page.evaluate(() => window.scrollTo(0, 0));

    expect(seen.size, `expected to see ${expectedPanelCount} panels while scrolling`).toBeGreaterThanOrEqual(expectedPanelCount);
}

async function measureScenario(
    page: any,
    request: any,
    dashboardUid: string,
    expectedUniqueStreams: number,
    panelCount: number
) {
    await page.goto('about:blank');
    await page.waitForTimeout(1000);
    await resetBackendStats(request);
    await page.goto(`/d/${dashboardUid}?from=now-5m&to=now&kiosk`);
    await scrollThroughAllPanels(page, panelCount);

    // Let panels mount and the backend register its stream(s).
    await expect.poll(async () => (await readBackendStats(request)).frames_sent, { timeout: 60_000 }).toBeGreaterThan(0);
    await page.waitForTimeout(1000);

    const uniqueStreams = (await readBackendStats(request)).unique_stream_paths;
    const sampleTicks = 12;
    const targetSamples = Math.max(uniqueStreams, expectedUniqueStreams) * sampleTicks;

    // Drain any buffered warmup values, then measure a clean window.
    await resetBackendStats(request);
    await expect.poll(async () => (await readBackendStats(request)).frames_sent, { timeout: 60_000 }).toBeGreaterThan(0);
    await resetBackendStats(request, targetSamples);

    const windowStarted = Date.now();
    let backend = await readBackendStats(request);
    const deadline = Date.now() + benchmarkDurationMs + 30_000;
    while (backend.run_stream_samples < targetSamples && Date.now() < deadline) {
        await page.waitForTimeout(250);
        backend = await readBackendStats(request);
    }
    const windowSeconds = (Date.now() - windowStarted) / 1000;

    return {
        unique_stream_paths: backend.unique_stream_paths,
        run_stream_samples: backend.run_stream_samples,
        run_stream_target_samples: targetSamples,
        run_stream_calls: backend.run_stream_calls,
        frames_sent: backend.frames_sent,
        values_sent: backend.values_sent,
        run_stream_runtime_ns: backend.run_stream_runtime_ns,
        run_stream_median_runtime_ns: backend.run_stream_median_runtime_ns,
        backend_heap_alloc_bytes: backend.backend_heap_alloc_bytes,
        window_seconds: windowSeconds,
        datapoints_per_second: windowSeconds > 0 ? backend.values_sent / windowSeconds : 0,
        frames_per_second: windowSeconds > 0 ? backend.frames_sent / windowSeconds : 0,
    };
}

test.describe('Multi-observer vs multi-parameter-stream benchmark', () => {
    test.describe.configure({ mode: 'serial' });

    test.beforeEach(async ({ page }) => {
        await page.addInitScript(() => {
            window.localStorage.setItem('grafana.whatsNew.dashboardShown', 'true');
            window.localStorage.setItem('grafana.whatsNew.datasourceShown', 'true');
            window.localStorage.setItem('grafana.whatsNewShown', 'true');
        });
    });

    test(
        'measures observer vs batched streaming for the same parameter set',
        { tag: ['@performance', '@benchmark'] },
        async ({ page, request }) => {
            test.setTimeout(600_000);

            await page.goto('about:blank');
            await page.waitForTimeout(1000);
            await resetBackendStats(request);

            const observerPayload = buildObserverDashboard();
            const observerUid = await createDashboard(request, observerPayload);
            let observerResult;
            try {
                observerResult = await measureScenario(page, request, observerUid, parameters.length, parameters.length);
            } finally {
                await page.goto('about:blank').catch(() => undefined);
                await request.delete(`/api/dashboards/uid/${observerUid}`).catch(() => undefined);
                await waitForStreamsQuiet(request);
            }

            const batchedPayload = buildBatchedDashboard();
            const batchedUid = await createDashboard(request, batchedPayload);
            let batchedResult;
            try {
                batchedResult = await measureScenario(page, request, batchedUid, 1, 1);
            } finally {
                await page.goto('about:blank').catch(() => undefined);
                await request.delete(`/api/dashboards/uid/${batchedUid}`).catch(() => undefined);
                await waitForStreamsQuiet(request);
            }

            const output = {
                started_at: new Date().toISOString(),
                datasource_uid: datasourceUid,
                endpoint,
                parameters: parameters.length,
                observer: observerResult,
                batched: batchedResult,
            };

            fs.mkdirSync(path.dirname(outputPath), { recursive: true });
            fs.writeFileSync(outputPath, JSON.stringify(output, null, 2));

            console.log(JSON.stringify(output, null, 2));

            expect(observerResult.frames_sent).toBeGreaterThan(0);
            expect(batchedResult.frames_sent).toBeGreaterThan(0);
        }
    );
});
