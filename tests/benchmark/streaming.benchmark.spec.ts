import fs from 'node:fs';
import path from 'node:path';

import { expect, test } from '@grafana/plugin-e2e';

type BenchmarkStats = {
    run_stream_runtime_ns: number;
    run_stream_median_runtime_ns: number;
    run_stream_samples: number;
    run_stream_calls: number;
    frames_sent: number;
    values_sent: number;
    unique_stream_paths: number;
    window_seconds: number;
    backend_alloc_bytes: number;
    backend_median_heap_alloc_bytes: number;
    backend_median_heap_alloc_bytes_min: number;
    backend_median_heap_alloc_bytes_max: number;
    backend_heap_alloc_growth_bytes: number;
    backend_heap_inuse_bytes: number;
    backend_heap_objects: number;
    backend_sys_bytes: number;
};

type BenchmarkResult = {
    panels: number;
    time_to_panels_ready_ms: number;
    browser_heap_after_gc_bytes: number;
    browser_task_duration_ms: number;
    browser_script_duration_ms: number;
    browser_layout_duration_ms: number;
    live_streams_opened: number;
    backend_run_stream_runtime_ns: number;
    backend_run_stream_samples: number;
    backend_run_stream_target_samples: number;
    backend_run_stream_calls: number;
    backend_frames_sent: number;
    backend_values_sent: number;
    backend_datapoints_per_second: number;
    backend_unique_stream_paths: number;
    backend_heap_alloc_bytes: number;
    backend_heap_alloc_growth_bytes: number;
    backend_heap_inuse_bytes: number;
    backend_heap_objects: number;
    backend_sys_bytes: number;
};

const datasourceUid = process.env.GRAFANA_BENCHMARK_DATASOURCE_UID ?? 'jaops-yamcs-main';
const datasourceType = 'jaops-yamcs-datasource';
const endpoint = process.env.GRAFANA_BENCHMARK_ENDPOINT ?? 'myproject_realtime';
const benchmarkDurationMs = Number(process.env.GRAFANA_BENCHMARK_DURATION_MS ?? '15000');
const sampleTicks = Number(process.env.GRAFANA_BENCHMARK_SAMPLE_TICKS ?? '15');
const streamSettleMs = Number(process.env.GRAFANA_BENCHMARK_SETTLE_MS ?? '4000');
const outputPath = process.env.GRAFANA_BENCHMARK_OUTPUT ?? 'benchmark-output/grafana/grafana.json';
const panelCounts = (process.env.GRAFANA_BENCHMARK_PANEL_COUNTS ?? '1,5,10,25,50,100')
    .split(',')
    .map((value) => Number(value.trim()))
    .filter((value) => Number.isFinite(value) && value > 0);
const parameters = JSON.parse(
    fs.readFileSync(path.resolve('scripts/benchmarks/grafana/parameters.json'), 'utf8')
) as string[];

function maxDataPointsForPanel(index: number): number {
    return 325 + (index % 7) * 37;
}

function panelParameter(index: number): string {
    return parameters[index % parameters.length];
}

function median(values: number[]): number {
    if (values.length === 0) {
        return 0;
    }
    const sorted = [...values].sort((left, right) => left - right);
    const middle = Math.floor(sorted.length / 2);
    if (sorted.length % 2 === 1) {
        return sorted[middle];
    }
    return Math.round((sorted[middle - 1] + sorted[middle]) / 2);
}

function buildPanel(index: number) {
    const columns = 4;
    const width = 6;
    const height = 8;
    const id = index + 1;
    const parameter = panelParameter(index);

    return {
        id,
        title: `Parameter ${id}`,
        type: 'timeseries',
        datasource: { type: datasourceType, uid: datasourceUid },
        gridPos: {
            h: height,
            w: width,
            x: (index % columns) * width,
            y: Math.floor(index / columns) * height,
        },
        maxDataPoints: maxDataPointsForPanel(index),
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
        options: {
            legend: { displayMode: 'hidden', placement: 'bottom', showLegend: false },
            tooltip: { mode: 'single', sort: 'none' },
        },
        fieldConfig: { defaults: {}, overrides: [] },
    };
}

function buildDashboard(panelCount: number) {
    const uid = `jybench-mixed-${panelCount}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    return {
        dashboard: {
            id: null,
            uid,
            title: `JAOPS benchmark ${panelCount} panels`,
            tags: ['jaops-benchmark'],
            timezone: 'browser',
            schemaVersion: 39,
            version: 0,
            refresh: '',
            time: { from: 'now-5m', to: 'now' },
            panels: Array.from({ length: panelCount }, (_, index) => buildPanel(index)),
        },
        folderId: 0,
        overwrite: true,
    };
}

async function createDashboard(request: any, panelCount: number): Promise<string> {
    const dashboardPayload = buildDashboard(panelCount);
    const createResponse = await request.post('/api/dashboards/db', { data: dashboardPayload });
    expect(
        createResponse.ok(),
        `dashboard create failed: ${createResponse.status()} ${await createResponse.text()}`
    ).toBeTruthy();
    const createBody = await createResponse.json();
    return createBody.uid ?? dashboardPayload.dashboard.uid;
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

async function collectBrowserMetrics(page: any) {
    const client = await page.context().newCDPSession(page);
    await client.send('Performance.enable');
    await client.send('HeapProfiler.collectGarbage');
    const heap = await client.send('Runtime.getHeapUsage');
    const performanceMetrics = await client.send('Performance.getMetrics');
    const metricByName = new Map<string, number>(
        performanceMetrics.metrics.map((metric: { name: string; value: number }) => [metric.name, metric.value])
    );
    await client.detach();

    return {
        browser_heap_after_gc_bytes: Math.round(heap.usedSize),
        browser_task_duration_ms: Math.round((metricByName.get('TaskDuration') ?? 0) * 1000),
        browser_script_duration_ms: Math.round((metricByName.get('ScriptDuration') ?? 0) * 1000),
        browser_layout_duration_ms: Math.round((metricByName.get('LayoutDuration') ?? 0) * 1000),
    };
}

async function visiblePanelIds(page: any): Promise<string[]> {
    return page.evaluate(() => {
        const ids = new Set<string>();
        for (const element of document.querySelectorAll('[data-panelid]')) {
            const panelId = element.getAttribute('data-panelid');
            if (panelId) {
                ids.add(panelId);
            }
        }
        for (const element of document.querySelectorAll('[data-viz-panel-key^="panel-"]')) {
            const panelId = element.getAttribute('data-viz-panel-key')?.replace(/^panel-/, '');
            if (panelId) {
                ids.add(panelId);
            }
        }
        return [...ids];
    });
}

async function waitForAllPanelsSeen(page: any, panelCount: number): Promise<void> {
    const seen = new Set<string>();
    const deadline = Date.now() + 60_000;

    while (Date.now() < deadline && seen.size < panelCount) {
        await page.evaluate(() => window.scrollTo(0, 0));
        let lastScrollY = -1;

        for (let i = 0; i < Math.max(80, panelCount * 2) && seen.size < panelCount; i++) {
            for (const panelId of await visiblePanelIds(page)) {
                seen.add(panelId);
            }

            const state = await page.evaluate(() => {
                window.scrollBy(0, Math.floor(window.innerHeight * 0.85));
                return {
                    scrollY: window.scrollY,
                    scrollHeight: document.documentElement.scrollHeight,
                    innerHeight: window.innerHeight,
                };
            });
            await page.waitForTimeout(100);

            if (state.scrollY === lastScrollY || state.scrollY + state.innerHeight >= state.scrollHeight) {
                for (const panelId of await visiblePanelIds(page)) {
                    seen.add(panelId);
                }
                break;
            }
            lastScrollY = state.scrollY;
        }
    }

    expect(seen.size, `expected to see ${panelCount} dashboard panels while scrolling`).toBeGreaterThanOrEqual(panelCount);
    await page.evaluate(() => window.scrollTo(0, 0));
}

async function waitForPanelsReady(page: any, request: any, panelCount: number): Promise<number> {
    const started = Date.now();
    await waitForAllPanelsSeen(page, panelCount);
    await expect.poll(async () => (await readBackendStats(request)).frames_sent, { timeout: 60_000 }).toBeGreaterThan(0);
    await expect(page.locator('[data-testid="data-testid Panel status error"]')).toHaveCount(0);
    return Date.now() - started;
}

async function drainBufferedStreamingValues(request: any): Promise<void> {
    await resetBackendStats(request);
    await expect
        .poll(async () => (await readBackendStats(request)).frames_sent, { timeout: 60_000 })
        .toBeGreaterThan(0);
}

async function waitForBackendStreamsQuiet(request: any): Promise<void> {
    await expect
        .poll(
            async () => {
                await resetBackendStats(request);
                await new Promise((resolve) => setTimeout(resolve, 1500));
                const stats = await readBackendStats(request);
                return stats.frames_sent;
            },
            { timeout: 30_000 }
        )
        .toBe(0);
}

async function warmPanelCount(page: any, request: any, panelCount: number): Promise<void> {
    const dashboardUid = await createDashboard(request, panelCount);
    try {
        await page.goto('about:blank');
        await page.waitForTimeout(streamSettleMs);
        await resetBackendStats(request);
        await page.goto(`/d/${dashboardUid}?from=now-5m&to=now&kiosk`);
        await waitForPanelsReady(page, request, panelCount);
    } finally {
        await page.goto('about:blank').catch(() => undefined);
        await page.waitForTimeout(streamSettleMs).catch(() => undefined);
        await request.delete(`/api/dashboards/uid/${dashboardUid}`).catch(() => undefined);
        await waitForBackendStreamsQuiet(request);
    }
}

test.describe('Grafana panel streaming benchmark', () => {
    test.describe.configure({ mode: 'serial' });

    test.beforeEach(async ({ page }) => {
        await page.addInitScript(() => {
            window.localStorage.setItem('grafana.whatsNew.dashboardShown', 'true');
            window.localStorage.setItem('grafana.whatsNew.datasourceShown', 'true');
            window.localStorage.setItem('grafana.whatsNewShown', 'true');
        });
    });

    test('measures panel streaming runtime', { tag: ['@performance', '@benchmark'] }, async ({ page, request }) => {
        test.setTimeout(Math.max(600_000, panelCounts.length * (benchmarkDurationMs + streamSettleMs * 8 + 60_000)));
        const results: BenchmarkResult[] = [];

        await page.goto('about:blank');
        await page.waitForTimeout(streamSettleMs);
        await resetBackendStats(request);

        for (const panelCount of panelCounts) {
            await warmPanelCount(page, request, panelCount);
            const dashboardUid = await createDashboard(request, panelCount);

            try {
                await page.goto('about:blank');
                await page.waitForTimeout(streamSettleMs);
                await resetBackendStats(request);
                await page.goto(`/d/${dashboardUid}?from=now-5m&to=now&kiosk`);
                const timeToPanelsReady = await waitForPanelsReady(page, request, panelCount);
                const targetSamples = panelCount * sampleTicks;
                await drainBufferedStreamingValues(request);
                await resetBackendStats(request, targetSamples);
                let backend = await readBackendStats(request);
                const backendHeapAllocSamples = [backend.backend_heap_alloc_bytes];
                const sampleDeadline = Date.now() + benchmarkDurationMs + 30_000;
                while (backend.run_stream_samples < targetSamples && Date.now() < sampleDeadline) {
                    await page.waitForTimeout(250);
                    backend = await readBackendStats(request);
                    backendHeapAllocSamples.push(backend.backend_heap_alloc_bytes);
                }
                expect(backend.run_stream_samples).toBeGreaterThanOrEqual(targetSamples);
                backend = await readBackendStats(request);
                backendHeapAllocSamples.push(backend.backend_heap_alloc_bytes);
                const medianBackendHeapAllocBytes = median(backendHeapAllocSamples);
                const browser = await collectBrowserMetrics(page);

                results.push({
                    panels: panelCount,
                    time_to_panels_ready_ms: timeToPanelsReady,
                    live_streams_opened: backend.unique_stream_paths,
                    backend_run_stream_runtime_ns: backend.run_stream_runtime_ns,
                    backend_run_stream_samples: backend.run_stream_samples,
                    backend_run_stream_target_samples: targetSamples,
                    backend_run_stream_calls: backend.run_stream_calls,
                    backend_frames_sent: backend.frames_sent,
                    backend_values_sent: backend.values_sent,
                    backend_datapoints_per_second: backend.window_seconds > 0 ? backend.values_sent / backend.window_seconds : 0,
                    backend_unique_stream_paths: backend.unique_stream_paths,
                    backend_median_heap_alloc_bytes: medianBackendHeapAllocBytes,
                    backend_median_heap_alloc_bytes_min: Math.min(...backendHeapAllocSamples),
                    backend_median_heap_alloc_bytes_max: Math.max(...backendHeapAllocSamples),
                    backend_heap_alloc_growth_bytes: backend.backend_heap_alloc_growth_bytes,
                    backend_heap_inuse_bytes: backend.backend_heap_inuse_bytes,
                    backend_heap_objects: backend.backend_heap_objects,
                    backend_sys_bytes: backend.backend_sys_bytes,
                    ...browser,
                });
            } finally {
                await page.goto('about:blank').catch(() => undefined);
                await page.waitForTimeout(streamSettleMs).catch(() => undefined);
                await request.delete(`/api/dashboards/uid/${dashboardUid}`).catch(() => undefined);
                await waitForBackendStreamsQuiet(request);
            }
        }

        await page.goto('about:blank');
        await page.waitForTimeout(streamSettleMs);

        fs.mkdirSync(path.dirname(outputPath), { recursive: true });
        fs.writeFileSync(
            outputPath,
            JSON.stringify(
                {
                    started_at: new Date().toISOString(),
                    datasource_uid: datasourceUid,
                    endpoint,
                    time_range: 'now-5m-now',
                    benchmark_duration_ms: benchmarkDurationMs,
                    sample_ticks: sampleTicks,
                    panel_counts: panelCounts,
                    scenario: 'mixed-parameters',
                    parameters: parameters.length,
                    results,
                },
                null,
                2
            )
        );
    });
});
