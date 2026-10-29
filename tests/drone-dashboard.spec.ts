import { expect, test } from '@playwright/test';
import type { Locator, Page } from '@playwright/test';

// Requires the running drone simulator and the deployed v2 drone dashboard; never issues commands.
// DEMO_DASHBOARDS_E2E=1 GRAFANA_URL=http://localhost:3000/grafana pnpm exec playwright test tests/drone-dashboard.spec.ts --project=chromium --workers=1
// For local anonymous Grafana/installed Chrome, add DRONE_ANONYMOUS=1 DRONE_CHROMIUM_CHANNEL=chrome and --no-deps.
// Set --output to a task-specific artifact folder when other suites are running concurrently.
const grafana = (process.env.DRONE_GRAFANA_URL || process.env.GRAFANA_URL || 'http://localhost:3000/grafana').replace(
    /\/$/,
    ''
);
const dashboardUid = 'yamcs-grafana-drone';
const tabs = {
    Flight: {
        Vehicle: 'canvas',
        'Flight path': 'xychart',
        Camera: 'jaops-telemetricimage-panel',
        'Flight state machine': 'canvas',
        Altitude: 'timeseries',
        Attitude: 'canvas',
        SoC: 'gauge',
        'Ground speed': 'stat',
        'Flight ready': 'stat',
    },
    Power: {
        'Battery pack': 'gauge',
        'Power bus': 'stat',
        'Cell voltages': 'bargauge',
        'Pack voltage': 'timeseries',
        'Pack current': 'timeseries',
        'Branch current': 'bargauge',
        'Branch switches': 'state-timeline',
        'Power faults': 'state-timeline',
    },
    Propulsion: {
        'Motor speed': 'timeseries',
        'Motor temperature': 'timeseries',
        Throttle: 'bargauge',
        'Motor current': 'bargauge',
        'Motor status': 'state-timeline',
        'Vibration spectrum': 'bargauge',
        'Angular rates': 'timeseries',
    },
    Operations: {
        Commands: 'jaops-commanding-panel',
        'Last command': 'stat',
        'Data links': 'jaops-links-panel',
        'Command history': 'jaops-commandhistory-panel',
        Events: 'table',
    },
};

type PanelDefinition = {
    key: string;
    spec: {
        title: string;
        vizConfig: { group: string; spec: any };
        data: { spec: { queries: Array<{ spec: { query: { spec: any } } }> } };
    };
};
type Field = {
    name: string;
    type: string;
    values: any[];
    unit?: string;
    thresholds?: { mode: string; steps: Array<{ value: number | null; color: string }> };
    lastDisplay?: { text: string; suffix?: string; color?: string };
};
type Rendered = { state: string; errors: string[]; fields: Field[] };

function panel(page: Page, title: string) {
    return page.getByTestId(`data-testid Panel header ${title}`);
}

// Inspect the data passed to the actual renderer, AFTER field overrides. Raw query frames miss
// regressions where datasource metadata defeats panel defaults. Missing instrumentation fails closed.
async function rendered(locator: Locator): Promise<Rendered | null> {
    return locator.evaluate((element) => {
        const key = Object.keys(element).find((key) => key.startsWith('__reactFiber'));
        if (!key) {
            throw new Error('Grafana React panel instrumentation is unavailable');
        }
        const candidates: any[] = [];
        function visit(fiber: any) {
            if (!fiber) {
                return;
            }
            const data = fiber.memoizedProps?.data;
            if (data?.series) {
                candidates.push(data);
            }
            for (let child = fiber.child; child; child = child.sibling) {
                visit(child);
            }
        }
        const fiber = (element as any)[key];
        visit(fiber);
        visit(fiber.alternate);
        const newest = (data: any) =>
            Math.max(
                0,
                ...data.series.flatMap((frame: any) =>
                    frame.fields.filter((field: any) => field.type === 'time').flatMap((field: any) => field.values)
                )
            );
        candidates.sort((a, b) => newest(b) - newest(a));
        const data = candidates[0];
        if (!data) {
            return null;
        }
        return {
            state: data.state,
            errors: [data.error, ...(data.errors || [])].filter(Boolean).map((error: any) => error.message),
            fields: data.series.flatMap((frame: any) =>
                frame.fields.map((field: any) => {
                    const values = Array.from(field.values) as any[];
                    const value = values.filter((value) => value !== null && value !== undefined).at(-1);
                    return {
                        name: field.name,
                        type: field.type,
                        values,
                        unit: field.config?.unit,
                        thresholds: field.config?.thresholds,
                        lastDisplay: field.display && value !== undefined ? field.display(value) : undefined,
                    };
                })
            ),
        };
    });
}

function timestamp(data: Rendered | null) {
    return Math.max(
        0,
        ...(data?.fields.filter((field) => field.type === 'time').flatMap((field) => field.values) || [])
    );
}

function unitsFor(name: string): string | undefined {
    if (/Voltage|railVoltages/.test(name)) {
        return 'volt';
    }
    if (/Current$|branchCurrents|\.current$/.test(name)) {
        return 'amp';
    }
    if (/Temperature|\.temperature$/.test(name)) {
        return 'celsius';
    }
    if (/StateOfCharge|\.throttle$/.test(name)) {
        return 'percent';
    }
    if (/\.rpm$/.test(name)) {
        return 'rotrpm';
    }
    if (/VibrationSpectrum/.test(name)) {
        return 'accG';
    }
    if (/Attitude\.|Navigation\.(latitude|longitude)$/.test(name)) {
        return 'degree';
    }
    if (/AngularRates\./.test(name)) {
        return 'rotdegs';
    }
    if (/Navigation\.groundSpeed$/.test(name)) {
        return 'velocityms';
    }
    if (/Navigation\.altitude$|TargetAltitude$|^(East|North)$/.test(name)) {
        return 'lengthm';
    }
    if (/BatteryPower$/.test(name)) {
        return 'watt';
    }
    if (/BatteryEnergyRemaining$/.test(name)) {
        return 'watth';
    }
    if (/MissionElapsedTime$/.test(name)) {
        return 'dthms';
    }
    return undefined;
}

function alarmLimits(name: string): number[] | undefined {
    if (/BatteryPackVoltage$/.test(name)) {
        return [25.6, 27.2, 34, 34.4];
    }
    if (/BatteryPackCurrent$/.test(name)) {
        return [-25, -22, 55, 65];
    }
    if (/BatteryCellVoltages/.test(name)) {
        return [3.2, 3.4, 4.25, 4.3];
    }
    if (/Motor\dTemperature$|Motors\[\d\]\.temperature$/.test(name)) {
        return [-20, -10, 75, 90];
    }
    if (/BatteryTemperature$/.test(name)) {
        return [-20, -10, 55, 70];
    }
    if (/BatteryStateOfCharge$/.test(name)) {
        return [15, 25, 100];
    }
    if (/Motors\[\d\]\.current$/.test(name)) {
        return [0, 18, 24];
    }
    if (/Motors\[\d\]\.rpm$/.test(name)) {
        return [0, 14000];
    }
    if (/Navigation\.altitude$|TargetAltitude$/.test(name)) {
        return [-5, -2, 110, 120];
    }
    return undefined;
}

function checkFieldMetadata(data: Rendered, title: string) {
    for (const field of data.fields.filter((field) => field.type === 'number')) {
        const context = `${title}: ${field.name}`;
        const unit = unitsFor(field.name);
        if (unit) {
            expect.soft(field.unit, `${context} rendered unit`).toBe(unit);
        }
        const limits = alarmLimits(field.name);
        if (limits) {
            expect.soft(field.thresholds?.mode, `${context} alarm mode`).toBe('absolute');
            expect
                .soft(
                    field.thresholds?.steps
                        .filter((step) => step.value !== null && Number.isFinite(step.value))
                        .map((step) => step.value),
                    `${context} MDB alarm boundaries`
                )
                .toEqual(limits);
            const steps = field.thresholds?.steps || [];
            const colors =
                limits.length === 4
                    ? ['red', 'yellow', 'green', 'yellow', 'red']
                    : limits.length === 2
                      ? ['red', 'green', 'red']
                      : /StateOfCharge$/.test(field.name)
                        ? ['red', 'yellow', 'green', 'red']
                        : ['red', 'green', 'yellow', 'red'];
            expect
                .soft(
                    steps.map((step) => step.color),
                    `${context} critical/warning/normal bands`
                )
                .toEqual(colors);
        }
    }
}

async function definitions(request: any) {
    const response = await request.get(
        `${grafana}/apis/dashboard.grafana.app/v2/namespaces/default/dashboards/${dashboardUid}`
    );
    expect(response.ok(), `Deploy the drone v2 dashboard first: ${response.status()} ${await response.text()}`).toBe(
        true
    );
    const dashboard = await response.json();
    expect(dashboard.apiVersion).toBe('dashboard.grafana.app/v2');
    return dashboard.spec;
}

function references(layout: any): string[] {
    if (!layout || typeof layout !== 'object') {
        return [];
    }
    if (layout.kind === 'ElementReference') {
        return [layout.name];
    }
    return Object.values(layout).flatMap(references);
}

function tabLayout(layout: any, title: string): any {
    if (!layout || typeof layout !== 'object') {
        return undefined;
    }
    if (layout.kind === 'TabsLayoutTab' && layout.spec.title === title) {
        return layout.spec.layout;
    }
    return Object.values(layout)
        .map((value) => tabLayout(value, title))
        .find(Boolean);
}

async function openTab(page: Page, title: string) {
    await page.goto(`${grafana}/d/${dashboardUid}?kiosk&from=now-15m&to=now&var-endpoint=drone_realtime`);
    const tab = page.getByRole('tab', { name: title, exact: true });
    await expect(tab).toBeVisible({ timeout: 30000 });
    await tab.click();
    await expect(tab).toHaveAttribute('aria-selected', 'true');
}

test.use({
    viewport: { width: 1920, height: 1080 },
    ...(process.env.DRONE_CHROMIUM_CHANNEL ? { channel: process.env.DRONE_CHROMIUM_CHANNEL } : {}),
    ...(process.env.DRONE_ANONYMOUS === '1' ? { storageState: { cookies: [], origins: [] } } : {}),
});

test.describe('drone dashboard @drone', () => {
    test.skip(
        process.env.DEMO_DASHBOARDS_E2E !== '1',
        'Opt-in deployment suite: set DEMO_DASHBOARDS_E2E=1 after provisioning drone Yamcs, its datasource and v2 dashboard.'
    );
    test.beforeEach(async ({ page }) => {
        await page.addInitScript(() => {
            for (const key of [
                'grafana.whatsNew.dashboardShown',
                'grafana.whatsNew.datasourceShown',
                'grafana.whatsNewShown',
            ]) {
                localStorage.setItem(key, 'true');
            }
        });
    });

    for (const [tab, expected] of Object.entries(tabs)) {
        test(`${tab}: every panel renders live with correct types, units and alarm thresholds`, async ({
            page,
            request,
        }) => {
            test.setTimeout(180000);
            const spec = await definitions(request);
            const layout = tabLayout(spec.layout, tab);
            expect(layout, `v2 tab ${tab}`).toBeTruthy();
            const panels: PanelDefinition[] = references(layout).map((key) => ({ key, ...spec.elements[key] }));
            expect(Object.fromEntries(panels.map(({ spec }) => [spec.title, spec.vizConfig.group]))).toEqual(expected);
            const tabKeys = new Set(Object.keys(tabs).flatMap((title) => references(tabLayout(spec.layout, title))));
            const header: PanelDefinition[] = references(spec.layout)
                .filter((key) => !tabKeys.has(key))
                .map((key) => ({ key, ...spec.elements[key] }));
            expect(header.map(({ spec }) => [spec.title, spec.vizConfig.group])).toEqual([
                ['', 'text'],
                ['', 'text'],
                ['Callsign', 'stat'],
                ['Flight mode', 'stat'],
                ['Motors', 'stat'],
                ['Failsafe', 'stat'],
                ['Mission elapsed', 'stat'],
            ]);
            await openTab(page, tab);
            const advancing: Array<{ title: string; initial: number }> = [];
            for (const definition of [...header, ...panels]) {
                const title = definition.spec.title;
                await test.step(title || definition.spec.vizConfig.group, async () => {
                    const container = page.locator(`[data-viz-panel-key="${definition.key}"]`);
                    await container.scrollIntoViewIfNeeded();
                    await expect(container).toBeVisible();
                    const queries = definition.spec.data.spec.queries.map((query) => query.spec.query.spec);
                    if (definition.key === header[0].key) {
                        await expect(container.locator('img')).toHaveCount(1);
                        await expect
                            .poll(
                                () =>
                                    container
                                        .locator('img')
                                        .evaluate(
                                            (image: HTMLImageElement) => image.complete && image.naturalWidth > 0
                                        ),
                                { message: 'Header logo must decode', timeout: 15000 }
                            )
                            .toBe(true);
                        return;
                    }
                    if (definition.spec.vizConfig.group === 'text') {
                        await expect(container.getByTestId('TextPanel-converted-content')).not.toBeEmpty();
                        return;
                    }
                    const parameters = queries.map((query) => query.parameter).filter(Boolean);
                    await expect
                        .poll(async () => (await rendered(container))?.state, {
                            message: `${title} must finish loading or stream`,
                            timeout: 45000,
                        })
                        .toMatch(/^(Done|Streaming)$/);
                    const data = (await rendered(container))!;
                    expect(data.errors, `${title} query errors`).toEqual([]);
                    const markers: Record<string, string> = {
                        'jaops-commanding-panel': 'jaops-commanding-panel',
                        'jaops-links-panel': 'jaops-links-panel',
                        'jaops-commandhistory-panel': 'jaops-command-history-panel',
                        'jaops-telemetricimage-panel': 'jaops-telemetric-image-panel',
                    };
                    const marker = markers[definition.spec.vizConfig.group];
                    if (marker) {
                        await expect(container.getByTestId(marker)).toBeVisible();
                    }
                    await expect(
                        container.getByText(/Field not found|Plugin not found|Streaming channel error/, {
                            exact: false,
                        })
                    ).toHaveCount(0);
                    if (parameters.length && definition.spec.vizConfig.group !== 'jaops-telemetricimage-panel') {
                        await expect
                            .poll(
                                async () => {
                                    const current = await rendered(container);
                                    return parameters.filter(
                                        (parameter) =>
                                            !current?.fields.some(
                                                (field) =>
                                                    field.name === parameter &&
                                                    field.values.some((value) => value !== null && value !== undefined)
                                            )
                                    );
                                },
                                {
                                    message: `${title} must have live data for every requested parameter`,
                                    timeout: 45000,
                                }
                            )
                            .toEqual([]);
                        const current = (await rendered(container))!;
                        checkFieldMetadata(current, title);
                        advancing.push({ title, initial: timestamp(current) });
                    }
                });
            }
            // Prove these are active subscriptions, not a successful historical response or static placeholders.
            await expect
                .poll(
                    async () => {
                        const stalled = [];
                        for (const { title, initial } of advancing) {
                            if (timestamp(await rendered(panel(page, title))) <= initial) {
                                stalled.push(title);
                            }
                        }
                        return stalled;
                    },
                    { message: `${tab} telemetry timestamps must advance`, timeout: 30000 }
                )
                .toEqual([]);
        });
    }

    test('Flight shows a live camera and exactly one telemetry-driven state', async ({ page, request }) => {
        test.setTimeout(90000);
        const spec = await definitions(request);
        const flight = references(tabLayout(spec.layout, 'Flight')).map((key) => spec.elements[key].spec);
        expect(flight.map((definition) => definition.title)).toEqual(
            expect.arrayContaining(['Camera', 'Flight state machine'])
        );
        const state = flight.find((definition) => definition.title === 'Flight state machine');
        expect(state.data.spec.queries.map((query: any) => query.spec.query.spec.parameter)).toEqual([
            '/drone/CurrentState',
        ]);
        await openTab(page, 'Flight');
        const camera = panel(page, 'Camera');
        await camera.scrollIntoViewIfNeeded();
        await expect(camera.getByTestId('jaops-telemetric-image-panel')).toBeVisible({ timeout: 30000 });
        const image = camera.locator('img');
        await expect(image).toHaveCount(1);
        await expect
            .poll(
                () =>
                    image.evaluate(
                        (img: HTMLImageElement) => img.complete && img.naturalWidth > 100 && img.naturalHeight > 100
                    ),
                { message: 'Camera must decode a real image, not show a broken URL', timeout: 30000 }
            )
            .toBe(true);
        const firstImage = await image.getAttribute('src');
        await expect
            .poll(() => image.getAttribute('src'), { message: 'Camera must refresh from telemetry', timeout: 15000 })
            .not.toBe(firstImage);
        const machine = panel(page, 'Flight state machine');
        await machine.scrollIntoViewIfNeeded();
        await expect
            .poll(
                async () => {
                    const data = await rendered(machine);
                    const current = data?.fields.find((field) => field.name === '/drone/CurrentState')?.values.at(-1);
                    if (!Number.isInteger(current) || current < 0 || current > 7) {
                        return false;
                    }
                    return Array.from(
                        { length: 8 },
                        (_, index) =>
                            data?.fields.find((field) => field.name === `S${index}`)?.values.at(-1) === current - index
                    ).every(Boolean);
                },
                { message: 'S0…S7 must derive from CurrentState with exactly one zero/active state', timeout: 30000 }
            )
            .toBe(true);
        const stateData = (await rendered(machine))!;
        const currentState = stateData.fields.find((field) => field.name === '/drone/CurrentState')!.values.at(-1);
        const active = stateData.fields.find((field) => field.name === `S${currentState}`)!;
        expect(active.lastDisplay?.color, 'Active state must have a mapped display color').toBeTruthy();
        for (const field of stateData.fields.filter(
            (field) => /^S[0-7]$/.test(field.name) && field.name !== active.name
        )) {
            expect(field.lastDisplay?.color, `${field.name} must not appear active`).not.toBe(
                active.lastDisplay?.color
            );
        }
        await expect(machine.getByText(/Field not found/)).toHaveCount(0);
    });

    test('Flight trajectory renders the physical circular survey with equal X/Y metre scale', async ({
        page,
        request,
    }) => {
        test.setTimeout(120000);
        await definitions(request);
        await openTab(page, 'Flight');
        const trajectory = panel(page, 'Flight path');
        await trajectory.scrollIntoViewIfNeeded();
        await expect(trajectory.locator('.uplot canvas')).toBeVisible({ timeout: 45000 });
        // uPlot is found through React refs, not axis-label OCR; assert the actual drawn scale and samples.
        const chart = () =>
            trajectory.evaluate((element) => {
                const key = Object.keys(element).find((key) => key.startsWith('__reactFiber'));
                if (!key) {
                    throw new Error('Grafana React chart instrumentation is unavailable');
                }
                let plot: any;
                const seen = new Set<any>();
                function find(value: any, depth = 0) {
                    if (!value || typeof value !== 'object' || seen.has(value) || depth > 5 || plot) {
                        return;
                    }
                    seen.add(value);
                    if (typeof value.valToPos === 'function' && value.scales && value.bbox && value.data) {
                        plot = value;
                        return;
                    }
                    for (const name of ['current', 'plot', 'uPlot', 'uplot', 'memoizedState', 'next']) {
                        find(value[name], depth + 1);
                    }
                }
                function visit(fiber: any) {
                    if (!fiber || plot) {
                        return;
                    }
                    find(fiber.stateNode);
                    let hook = fiber.memoizedState;
                    for (let count = 0; hook && count < 100; count++, hook = hook.next) {
                        find(hook.memoizedState);
                    }
                    for (let child = fiber.child; child; child = child.sibling) {
                        visit(child);
                    }
                }
                visit((element as any)[key]);
                if (!plot) {
                    return null;
                }
                const series = plot.series[1];
                const xScale = series?.facets?.[0]?.scale || 'x';
                const yScale = series?.facets?.[1]?.scale || series?.scale || 'y';
                return {
                    xPixelsPerMetre: Math.abs(plot.valToPos(1, xScale) - plot.valToPos(0, xScale)),
                    yPixelsPerMetre: Math.abs(plot.valToPos(1, yScale) - plot.valToPos(0, yScale)),
                    eastDirection: plot.valToPos(1, xScale) - plot.valToPos(0, xScale),
                    northDirection: plot.valToPos(1, yScale) - plot.valToPos(0, yScale),
                    data: plot.data,
                };
            });
        await expect
            .poll(chart, { message: 'The rendered native XY chart must expose its uPlot instance', timeout: 30000 })
            .not.toBeNull();
        const plot = (await chart())!;
        expect(plot.xPixelsPerMetre).toBeGreaterThan(0);
        expect(plot.yPixelsPerMetre).toBeGreaterThan(0);
        expect(plot.eastDirection, 'East must point right').toBeGreaterThan(0);
        expect(plot.northDirection, 'North must point up').toBeLessThan(0);
        expect
            .soft(
                Math.abs(plot.xPixelsPerMetre / plot.yPixelsPerMetre - 1),
                'Equal physical metres per pixel (within 1%)'
            )
            .toBeLessThan(0.01);
        const [east, north] = plot.data[1] as [number[], number[]];
        const points = east
            .map((x, index) => [x, north[index]])
            .filter(([x, y]) => Number.isFinite(x) && Number.isFinite(y));
        expect(points.length, 'Archive must include a complete survey, not only the current point').toBeGreaterThan(
            100
        );
        const xs = points.map(([x]) => x);
        const ys = points.map(([, y]) => y);
        expect(Math.max(...xs) - Math.min(...xs), '80 m radius circle east/west span').toBeGreaterThan(140);
        expect(Math.max(...ys) - Math.min(...ys), '80 m radius circle north/south span').toBeGreaterThan(140);
        expect(Math.max(...xs)).toBeLessThan(95);
        expect(Math.min(...xs)).toBeGreaterThan(-95);
        expect(Math.max(...ys)).toBeLessThan(175);
        expect(Math.min(...ys)).toBeGreaterThan(-15);
        // drone_model.py: radius 80 m, centre 80 m north of home. Return-to-home chords are valid.
        const survey = points.filter(([x, y]) => Math.abs(Math.hypot(x, y - 80) - 80) < 12);
        expect(
            survey.length / points.length,
            'Most of the actual route must lie on the physical survey circle'
        ).toBeGreaterThan(0.7);
        const sectors = new Set(
            survey.map(([x, y]) => Math.floor((Math.atan2(y - 80, x) + Math.PI) / (Math.PI / 4)) % 8)
        );
        expect(sectors.size, 'The path must cover the whole circle, not a line or a short arc').toBe(8);
        const fields = (await rendered(trajectory))!.fields;
        const field = (name: string) => {
            const result = fields.find((field) => field.name === name);
            expect(result, `Trajectory must retain ${name}`).toBeDefined();
            return result!;
        };
        const eastField = field('East');
        const northField = field('North');
        expect(eastField.unit).toBe('lengthm');
        expect(northField.unit).toBe('lengthm');
        const latitude = field('/drone/Navigation.latitude');
        const longitude = field('/drone/Navigation.longitude');
        const errors = eastField.values.flatMap((east, index) => {
            const north = northField.values[index];
            const lat = latitude.values[index];
            const lon = longitude.values[index];
            if (![east, north, lat, lon].every((value) => typeof value === 'number' && Number.isFinite(value))) {
                return [];
            }
            return [
                Math.abs(east - (lon - 2.3522) * 111320 * Math.cos((48.8566 * Math.PI) / 180)),
                Math.abs(north - (lat - 48.8566) * 111320),
            ];
        });
        expect(errors.length).toBeGreaterThan(200);
        expect(
            Math.max(...errors),
            'XY metres must be projected from actual telemetry, not a synthetic shape'
        ).toBeLessThan(0.1);
    });
});
