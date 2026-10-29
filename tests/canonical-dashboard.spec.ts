import { expect, test, type APIRequestContext, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

// Requires the Yamcs Demo MDB (/myproject, /features and /images), not the drone demo.
// GRAFANA_URL may include a subpath, for example http://localhost:3000/grafana.
// These tests never provision dashboards, issue spacecraft commands or change data links.
// For an anonymous local demo without bundled Chromium:
// DEMO_DASHBOARDS_E2E=1 CANONICAL_BROWSER_CHANNEL=chrome CANONICAL_ANONYMOUS=1
// playwright test tests/canonical-dashboard.spec.ts --project=chromium --no-deps
test.use({
    ...(process.env.CANONICAL_BROWSER_CHANNEL ? { channel: process.env.CANONICAL_BROWSER_CHANNEL } : {}),
    ...(process.env.CANONICAL_ANONYMOUS === '1' ? { storageState: { cookies: [], origins: [] } } : {}),
});

const uid = '0708823c-c106-407b-9861-f64283746167';
const provisioningFile = resolve(
    __dirname,
    '../provisioning/dashboards/Example Demo/yamcs_quickstart_demo_dashboard.json'
);
const tabNames = ['Overview', 'Telemetry', 'Parameter types', 'Operations'];
const requiredPanelTypes = [
    'text',
    'stat',
    'gauge',
    'timeseries',
    'state-timeline',
    'canvas',
    'geomap',
    'bargauge',
    'table',
    'jaops-staticimage-panel',
    'jaops-telemetricimage-panel',
    'jaops-commanding-panel',
    'jaops-commandhistory-panel',
    'jaops-varcommanding-panel',
    'jaops-alarms-panel',
    'jaops-links-panel',
    'jaops-timesync-panel',
];
const requiredQueryTypes = [
    'single',
    'plot',
    'discrete',
    'time',
    'image',
    'commanding',
    'command-history',
    'events',
    'alarms',
    'links',
];
const pluginMarkers: Record<string, string> = {
    'jaops-staticimage-panel': 'jaops-static-image-panel',
    'jaops-telemetricimage-panel': 'jaops-telemetric-image-panel',
    'jaops-commanding-panel': 'jaops-commanding-panel',
    'jaops-commandhistory-panel': 'jaops-command-history-panel',
    'jaops-varcommanding-panel': 'jaops-variable-setting-panel',
    'jaops-alarms-panel': 'jaops-alarms-panel',
    'jaops-links-panel': 'jaops-links-panel',
    'jaops-timesync-panel': 'jaops-time-sync-panel',
};

interface Query {
    type: string;
    parameter?: string;
    parameters?: string[];
    command?: string;
    endpoint?: string;
    endpointVariable?: string;
    asVariable?: boolean;
}

interface Panel {
    id: number;
    title: string;
    data: {
        spec: { queries: Array<{ spec: { refId: string; hidden?: boolean; query: { group: string; spec: Query } } }> };
    };
    vizConfig: { group: string; spec: { options: Record<string, any>; fieldConfig: Record<string, any> } };
}

interface Layout {
    kind: string;
    spec: {
        title?: string;
        layout?: Layout;
        rows?: Layout[];
        tabs?: Layout[];
        items?: Layout[];
        element?: { name: string };
    };
}

interface Dashboard {
    apiVersion: string;
    kind: string;
    metadata: { name: string };
    spec: {
        title: string;
        elements: Record<string, { kind: string; spec: Panel }>;
        layout: Layout;
        variables: Array<{ spec: { name: string; current: { value: string } } }>;
    };
}

interface FrameField {
    name: string;
    type: string;
    values: any[];
    size: number;
    config: {
        unit?: string;
        thresholds?: { mode: string; steps: Array<{ value: number | null; color: string }> };
    };
}

interface Frame {
    refId?: string;
    length: number;
    fields: FrameField[];
}

interface Snapshot {
    state?: string;
    errors: string[];
    frames: Frame[];
    rawFrames: Frame[];
}

function url(baseURL: string | undefined, path: string) {
    return `${(baseURL || 'http://localhost:3000').replace(/\/$/, '')}/${path.replace(/^\//, '')}`;
}

function walkLayout(layout: Layout, visit: (node: Layout) => void) {
    visit(layout);
    for (const child of [...(layout.spec.rows || []), ...(layout.spec.tabs || []), ...(layout.spec.items || [])]) {
        walkLayout(child, visit);
    }
    if (layout.spec.layout) {
        walkLayout(layout.spec.layout, visit);
    }
}

function tabs(dashboard: Dashboard) {
    const result: Layout[] = [];
    walkLayout(dashboard.spec.layout, (node) => {
        if (node.kind === 'TabsLayoutTab') {
            result.push(node);
        }
    });
    return result;
}

function panelKeys(layout: Layout) {
    const keys: string[] = [];
    walkLayout(layout, (node) => {
        if (node.spec.element) {
            keys.push(node.spec.element.name);
        }
    });
    return keys;
}

function queries(panel: Panel) {
    return panel.data.spec.queries.filter((query) => !query.spec.hidden).map((query) => query.spec);
}

function parameters(query: Query) {
    return query.parameters?.length ? query.parameters : (query.parameter || '').split(/[\n,]+/).filter(Boolean);
}

async function loadDashboard(request: APIRequestContext, baseURL: string | undefined): Promise<Dashboard> {
    const response = await request.get(
        url(baseURL, `/apis/dashboard.grafana.app/v2/namespaces/default/dashboards/${uid}`)
    );
    expect(response.ok(), `Canonical v2 dashboard: ${response.status()} ${await response.text()}`).toBeTruthy();
    const dashboard = (await response.json()) as Dashboard;
    expect(dashboard.apiVersion).toBe('dashboard.grafana.app/v2');
    expect(dashboard.metadata.name).toBe(uid);
    expect(
        tabs(dashboard).map((tab) => tab.spec.title),
        'Deploy the Yamcs Demo tabbed dashboard, not the legacy quickstart'
    ).toEqual(tabNames);
    return dashboard;
}

// Scenes is the data source actually consumed by the panels; a 200 response or a
// successful Live subscription alone does not establish that telemetry arrived.
async function snapshot(page: Page, key: string): Promise<Snapshot | null> {
    return page.evaluate((panelKey) => {
        const seen = new Set<object>();
        let panel: any;
        function walk(object: any) {
            if (!object || typeof object !== 'object' || seen.has(object)) {
                return;
            }
            seen.add(object);
            if (object.state?.key === panelKey && object.state?.pluginId) {
                panel = object;
                return;
            }
            for (const value of Object.values(object.state || {}) as any[]) {
                if (Array.isArray(value)) {
                    value.forEach(walk);
                } else if (value?.state) {
                    walk(value);
                }
            }
        }
        walk((window as any).__grafanaSceneContext);
        if (!panel) {
            return null;
        }
        const providers: any[] = [];
        for (let provider = panel.state.$data; provider; provider = provider.state?.$data) {
            if (provider.state?.data) {
                providers.push(provider.state.data);
            }
        }
        const serialize = (frames: any[] = []) =>
            frames.map((frame) => ({
                refId: frame.refId,
                length: frame.length,
                fields: frame.fields.map((field: any) => ({
                    name: field.name,
                    type: field.type,
                    size: field.values.length,
                    values: Array.from(field.values).slice(-256),
                    config: JSON.parse(JSON.stringify(field.config || {})),
                })),
            }));
        return {
            state: providers[0]?.state,
            errors: providers.flatMap((data) =>
                [...(data.errors || []), ...(data.error ? [data.error] : [])].map(
                    (error) => error.message || String(error)
                )
            ),
            frames: serialize(providers[0]?.series),
            rawFrames: serialize(providers[providers.length - 1]?.series),
        };
    }, key);
}

function queryProblems(panel: Panel, data: Snapshot | null): string[] {
    if (!data) {
        return ['Panel scene not found'];
    }
    const problems = [...data.errors];
    if (queries(panel).length && !['Done', 'Streaming'].includes(data.state || '')) {
        problems.push(`Query state: ${data.state}`);
    }
    const fields = data.rawFrames.flatMap((frame) => frame.fields);
    for (const {
        refId,
        query: { spec: query },
    } of queries(panel)) {
        const prefix = `${refId} ${query.type}`;
        if (query.type === 'commanding') {
            // This query intentionally returns no frames; command metadata is verified separately.
            continue;
        }
        const names = parameters(query);
        const expected = names.length
            ? names
            : (
                  {
                      time: ['time'],
                      events: ['time', 'message', 'severity'],
                      alarms: ['alarms'],
                      'command-history': ['commands'],
                      links: ['name', 'status', 'disabled', 'dataInCount', 'dataOutCount'],
                  } as Record<string, string[]>
              )[query.type];
        if (!expected) {
            problems.push(`${prefix}: no value assertion implemented for this query type`);
            continue;
        }
        for (const name of expected) {
            const field = fields.find((candidate) => candidate.name === name);
            if (!field) {
                problems.push(`${prefix}: missing field ${name}`);
                continue;
            }
            // An empty alarm/history collection is valid; a missing collection schema is not.
            if (
                !['alarms', 'command-history'].includes(query.type) &&
                !field.values.some((value) => value !== null && value !== undefined)
            ) {
                problems.push(`${prefix}: ${name} has no values`);
            }
            if (field.type === 'number' && field.values.some((value) => value !== null && !Number.isFinite(value))) {
                problems.push(`${prefix}: ${name} contains non-finite numeric values`);
            }
            if (
                field.type === 'time' &&
                field.values.some((value) => value !== null && (!Number.isFinite(value) || value <= 0))
            ) {
                problems.push(`${prefix}: ${name} contains invalid timestamps`);
            }
        }
    }
    return problems;
}

function latestTime(data: Snapshot | null) {
    return Math.max(
        0,
        ...(data?.rawFrames || []).flatMap((frame) =>
            frame.fields
                .filter((field) => field.type === 'time')
                .flatMap((field) => field.values.filter((value) => Number.isFinite(value)))
        )
    );
}

function checkFieldMetadata(field: FrameField) {
    const name = field.name;
    const expectedUnit = /Voltage|CellVoltages/.test(name)
        ? 'volt'
        : /DemoCurrent|PowerMatrix|PowerChannels\[\d+\]\.current/.test(name)
          ? 'amp'
          : /Battery[12]_Temp$|DemoTemperature/.test(name)
            ? 'celsius'
            : /\.stateOfCharge$/.test(name)
              ? 'percent'
              : /DemoBundle\.attitude\.|^\/myproject\/(Latitude|Longitude)$/.test(name)
                ? 'degree'
                : name === '/myproject/Height'
                  ? 'lengthm'
                  : undefined;
    if (expectedUnit) {
        expect.soft(field.type, `${name}: scalar/member numeric field`).toBe('number');
        expect
            .soft(field.config.unit, `${name}: datasource unit, including aggregate/array member inheritance`)
            .toBe(expectedUnit);
    }
    // Independent MDB expectations, not values copied from the dashboard defaults.
    const bounds = /^\/features\/(DemoVoltage|DemoCellVoltages)|^\/myproject\/Battery[12]_Voltage$/.test(name)
        ? [25.6, 27.2, 34, 34.4]
        : /^\/features\/(DemoCurrent|DemoPowerMatrix|DemoPowerChannels\[\d+\]\.current)/.test(name)
          ? [-8, -4, 20, 28]
          : name === '/features/DemoTemperature' || /^\/myproject\/Battery[12]_Temp$/.test(name)
            ? [-20, -10, 55, 70]
            : name === '/features/DemoBundle.battery.stateOfCharge'
              ? [15, 25, 100]
              : undefined;
    if (bounds) {
        expect.soft(field.config.thresholds?.mode, `${name}: absolute MDB alarm thresholds`).toBe('absolute');
        const steps = field.config.thresholds?.steps || [];
        expect
            .soft(
                steps.map((step) => step.value),
                `${name}: exact MDB alarm boundaries`
            )
            .toEqual([null, ...bounds]);
        expect
            .soft(
                steps.map((step) => step.color),
                `${name}: critical/warning/nominal alarm colors`
            )
            .toEqual(
                bounds.length === 3 ? ['red', 'yellow', 'green', 'red'] : ['red', 'yellow', 'green', 'yellow', 'red']
            );
    }
    if (/^\/myproject\/(Detector_Temp|Gyro\.[xyz]|Magnetometer\.[xyz])$/.test(name)) {
        expect.soft(field.config.unit || '', `${name}: MDB declares no unit; do not invent one`).toBe('');
    }
}

async function checkRendering(content: Locator, panel: Panel, data: Snapshot) {
    const type = panel.vizConfig.group;
    await expect(content).toBeVisible();
    await expect(
        content.getByText(
            /Plugin unavailable|Panel plugin not found|Error loading panel|No data|No image received|No command configured|No variable command configured/i
        )
    ).toHaveCount(0);
    if (pluginMarkers[type]) {
        await expect(content.getByTestId(pluginMarkers[type])).toBeVisible();
    }
    if (['timeseries', 'state-timeline', 'geomap'].includes(type)) {
        await expect(content.locator('canvas').first()).toBeVisible();
        const dimensions = await content
            .locator('canvas')
            .first()
            .evaluate((canvas: HTMLCanvasElement) => [canvas.width, canvas.height]);
        expect(
            dimensions.every((dimension) => dimension > 0),
            `${type}: nonzero drawing surface`
        ).toBe(true);
    } else if (['gauge', 'bargauge', 'stat', 'canvas'].includes(type)) {
        expect((await content.innerText()).trim(), `${type}: rendered values, not just a panel title`).not.toBe('');
        await expect(content).not.toContainText(/\b(?:NaN|undefined)\b/);
    } else if (type === 'table') {
        await expect(content.getByRole('columnheader').first()).toBeVisible();
        await expect(content.getByRole('gridcell').first()).toBeVisible();
    } else if (type === 'text') {
        expect((await content.innerText()).trim()).not.toBe('');
    } else if (type.endsWith('image-panel') || type === 'jaops-telemetricimage-panel') {
        const images = content.locator('img');
        await expect(images.first()).toBeVisible();
        await expect
            .poll(
                () =>
                    images.evaluateAll(
                        (elements: HTMLImageElement[]) =>
                            elements.length > 0 &&
                            elements.every(
                                (image) => image.complete && image.naturalWidth > 0 && image.naturalHeight > 0
                            )
                    ),
                { message: `${panel.title}: image URLs must decode, not merely render an img element` }
            )
            .toBe(true);
        if (type === 'jaops-telemetricimage-panel') {
            const values = data.rawFrames
                .flatMap((frame) => frame.fields)
                .filter((field) => field.type === 'string')
                .flatMap((field) => field.values);
            expect(
                values.some((value) => typeof value === 'string' && /^(https?:|data:image\/|\/)/.test(value)),
                'Actual image URL telemetry'
            ).toBe(true);
        }
    }
}

async function inspectPanel(page: Page, dashboardURL: string, key: string, panel: Panel) {
    const address = new URL(dashboardURL);
    address.searchParams.set('viewPanel', key);
    await page.goto(address.toString());
    const content = page.getByTestId('data-testid panel content');
    await expect(content).toHaveCount(1, { timeout: 20000 });
    try {
        await expect
            .poll(async () => queryProblems(panel, await snapshot(page, key)), {
                message: `${key} "${panel.title}" (${panel.vizConfig.group}): every query must deliver valid frames`,
                timeout: 30000,
                intervals: [250, 500, 1000],
            })
            .toEqual([]);
    } finally {
        await test.info().attach(`${key}-frames`, {
            body: JSON.stringify(await snapshot(page, key), null, 2),
            contentType: 'application/json',
        });
    }
    const data = (await snapshot(page, key))!;
    for (const { query } of queries(panel)) {
        for (const parameter of parameters(query.spec)) {
            const field = data.frames.flatMap((frame) => frame.fields).find((field) => field.name === parameter);
            expect
                .soft(
                    field?.values.some((value) => value !== null && value !== undefined),
                    `${key}: ${parameter} survives transformations with actual values`
                )
                .toBe(true);
        }
    }
    for (const frame of data.frames) {
        for (const field of frame.fields) {
            expect.soft(field.size, `${key}/${field.name}: rectangular frame`).toBe(frame.length);
        }
    }
    for (const field of data.rawFrames.flatMap((frame) => frame.fields)) {
        checkFieldMetadata(field);
        if (field.name === '/features/DemoMode') {
            expect
                .soft(
                    field.values.every(
                        (value) =>
                            value === null ||
                            ['BOOT', 'SAFE', 'NOMINAL', 'PAYLOAD', 'DOWNLINK', 'FAULT'].includes(value)
                    )
                )
                .toBe(true);
        }
        if (field.name === '/features/DemoEnabled') {
            // Single queries stringify BOOLEAN in extractValue; discrete frames preserve booleans.
            expect.soft(['boolean', 'string']).toContain(field.type);
            expect
                .soft(field.values.every((value) => value === null || ['true', 'false'].includes(String(value))))
                .toBe(true);
        }
        if (field.name === '/features/DemoMessage') {
            expect.soft(field.values.some((value) => typeof value === 'string' && value.length > 0)).toBe(true);
        }
        if (field.name === '/features/DemoCounter') {
            expect
                .soft(field.values.every((value) => value === null || (Number.isFinite(value) && value >= 0)))
                .toBe(true);
        }
    }
    await checkRendering(content, panel, data);
    if (queries(panel).some(({ query }) => parameters(query.spec).includes('/features/DemoEnabled'))) {
        await expect(content).toContainText(/Enabled|Disabled/);
    }
    await expect(page.getByTestId('data-testid Panel status error')).toHaveCount(0);
}

test.describe('Yamcs Demo dashboard', () => {
    test.skip(
        process.env.DEMO_DASHBOARDS_E2E !== '1',
        'Opt-in demo suite: set DEMO_DASHBOARDS_E2E=1 and provision Yamcs Demo /myproject, /features and /images telemetry'
    );

    test.beforeEach(async ({ page }) => {
        await page.setViewportSize({ width: 1920, height: 1080 });
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

    test('provisioned and deployed v2 contracts cover every Yamcs Demo tab, panel type and query', async ({
        request,
        baseURL,
    }) => {
        const provisioned = JSON.parse(readFileSync(provisioningFile, 'utf8')) as Dashboard;
        expect(provisioned.apiVersion).toBe('dashboard.grafana.app/v2');
        expect(provisioned.spec.title).toBe('Yamcs Demo');
        expect(tabs(provisioned).map((tab) => tab.spec.title)).toEqual(tabNames);
        const deployed = await loadDashboard(request, baseURL);
        expect(deployed.spec.title).toBe(provisioned.spec.title);
        expect(deployed.spec.layout).toEqual(provisioned.spec.layout);
        expect(deployed.spec.elements).toEqual(provisioned.spec.elements);
        const referenced = panelKeys(deployed.spec.layout);
        expect([...referenced].sort(), 'No untested/orphaned panels or duplicated references').toEqual(
            Object.keys(deployed.spec.elements).sort()
        );
        const panels = Object.values(deployed.spec.elements).map((element) => element.spec);
        expect([...new Set(panels.map((panel) => panel.vizConfig.group))].sort()).toEqual(
            [...requiredPanelTypes].sort()
        );
        expect(
            [...new Set(panels.flatMap((panel) => queries(panel).map((query) => query.query.spec.type)))].sort()
        ).toEqual([...requiredQueryTypes].sort());
        for (const tab of tabs(deployed)) {
            expect(panelKeys(tab).length, `${tab.spec.title}: nonempty tab`).toBeGreaterThan(0);
        }
        for (const panel of panels) {
            for (const { query } of queries(panel)) {
                expect(query.group).toBe('jaops-yamcs-datasource');
                if (['single', 'plot', 'discrete', 'image'].includes(query.spec.type)) {
                    expect(parameters(query.spec).length, `${panel.title}: configured parameters`).toBeGreaterThan(0);
                }
            }
        }
    });

    for (const tabName of ['Header', ...tabNames]) {
        test(`${tabName}: every panel renders real query values and correct field metadata`, async ({
            page,
            request,
            baseURL,
        }) => {
            const dashboard = await loadDashboard(request, baseURL);
            const allTabs = tabs(dashboard);
            const tab = allTabs.find((candidate) => candidate.spec.title === tabName);
            const tabPanels = new Set(allTabs.flatMap(panelKeys));
            const keys = tab ? panelKeys(tab) : panelKeys(dashboard.spec.layout).filter((key) => !tabPanels.has(key));
            test.setTimeout(60000 + keys.length * 45000);
            expect(keys.length, `${tabName}: panels to validate`).toBeGreaterThan(0);
            const errors: string[] = [];
            page.on('pageerror', (error) => errors.push(error.message));
            page.on('console', (message) => {
                if (
                    message.type() === 'error' &&
                    /streaming channel|invalid channel|plugin|query|TypeError/i.test(message.text())
                ) {
                    errors.push(message.text());
                }
            });
            await page.goto(url(baseURL, `/d/${uid}?from=now-5m&to=now`));
            for (const name of tabNames) {
                await expect(page.getByRole('tab', { name, exact: true })).toBeVisible();
            }
            if (tab) {
                await page.getByRole('tab', { name: tabName, exact: true }).click();
                await expect(page.getByRole('tab', { name: tabName, exact: true })).toHaveAttribute(
                    'aria-selected',
                    'true'
                );
            }
            const dashboardURL = page.url();
            for (const key of keys) {
                const panel = dashboard.spec.elements[key].spec;
                await test.step(`${key}: ${panel.title || panel.vizConfig.group}`, async () => {
                    try {
                        await inspectPanel(page, dashboardURL, key, panel);
                    } catch (error) {
                        // Retain a failing assertion while still inspecting every remaining panel.
                        expect.soft(String(error), `${key}: ${panel.title}`).toBe('');
                    }
                });
            }
            expect(errors, 'Runtime and Grafana Live errors must not be hidden by an otherwise rendered panel').toEqual(
                []
            );
        });
    }

    test('realtime telemetry advances after leaving and returning to a tab', async ({ page, request, baseURL }) => {
        test.setTimeout(90000);
        const dashboard = await loadDashboard(request, baseURL);
        const overview = tabs(dashboard).find((tab) => tab.spec.title === 'Overview')!;
        const key = panelKeys(overview).find(
            (key) => dashboard.spec.elements[key].spec.vizConfig.group === 'timeseries'
        );
        expect(key, 'Overview must exercise a realtime trend').toBeTruthy();
        const panel = dashboard.spec.elements[key!].spec;
        await page.goto(url(baseURL, `/d/${uid}?from=now-5m&to=now`));
        await page.getByRole('tab', { name: 'Overview', exact: true }).click();
        await page.getByTestId(`data-testid Panel header ${panel.title}`).scrollIntoViewIfNeeded();
        await expect.poll(async () => queryProblems(panel, await snapshot(page, key!)), { timeout: 30000 }).toEqual([]);
        const beforeData = await snapshot(page, key!);
        const before = latestTime(beforeData);
        expect(before).toBeGreaterThan(0);
        await page.getByRole('tab', { name: 'Telemetry', exact: true }).click();
        await expect(page.getByRole('tab', { name: 'Telemetry', exact: true })).toHaveAttribute(
            'aria-selected',
            'true'
        );
        await page.getByRole('tab', { name: 'Overview', exact: true }).click();
        await page.getByTestId(`data-testid Panel header ${panel.title}`).scrollIntoViewIfNeeded();
        try {
            await expect
                .poll(async () => latestTime(await snapshot(page, key!)), {
                    timeout: 20000,
                    message:
                        'A visible trend must receive new mission timestamps, not remain frozen on a cached Live buffer',
                })
                .toBeGreaterThan(before);
        } finally {
            await test.info().attach('tab-resumption-frames', {
                body: JSON.stringify({ key, before: beforeData, after: await snapshot(page, key!) }, null, 2),
                contentType: 'application/json',
            });
        }
        expect(queryProblems(panel, await snapshot(page, key!))).toEqual([]);
    });

    test('operations commands resolve metadata and variable buttons change only dashboard state', async ({
        page,
        request,
        baseURL,
    }) => {
        test.setTimeout(90000);
        const dashboard = await loadDashboard(request, baseURL);
        const sourceResponse = await request.get(url(baseURL, '/api/datasources'));
        expect(sourceResponse.ok()).toBe(true);
        const sources = await sourceResponse.json();
        const endpoint = dashboard.spec.variables.find((variable) => variable.spec.name === 'endpoint')?.spec.current
            .value;
        expect(endpoint).toBe('demo_realtime');
        const source = sources.find(
            (candidate: any) =>
                candidate.uid === 'jaops-yamcs-main' &&
                candidate.type === 'jaops-yamcs-datasource' &&
                candidate.jsonData.endpoints?.[endpoint!]
        );
        expect(source, 'Canonical datasource with demo_realtime must be provisioned').toBeTruthy();
        const commandNames = new Set<string>();
        for (const panel of Object.values(dashboard.spec.elements).map((element) => element.spec)) {
            for (const { query } of queries(panel).filter((query) => query.query.spec.type === 'commanding')) {
                commandNames.add(query.spec.command!);
            }
            if (panel.vizConfig.group === 'jaops-commanding-panel') {
                for (const form of Object.values(panel.vizConfig.spec.options.commandForms || {}) as any[]) {
                    for (const command of [
                        form.commandName,
                        form.onCommand?.commandName,
                        form.offCommand?.commandName,
                    ]) {
                        if (command) {
                            commandNames.add(command);
                        }
                    }
                }
            }
        }
        expect(commandNames.size).toBeGreaterThan(0);
        for (const command of commandNames) {
            const response = await request.get(
                url(
                    baseURL,
                    `/api/datasources/uid/${source.uid}/resources/endpoint/${endpoint}/command/info?name=${encodeURIComponent(command)}`
                )
            );
            expect(response.ok(), `${command}: command metadata ${response.status()} ${await response.text()}`).toBe(
                true
            );
            const metadata = await response.json();
            expect(metadata.qualifiedName || metadata.name).toBe(command);
        }
        const mutations: string[] = [];
        page.on('request', (request) => {
            if (request.method() === 'POST' && /\/command(?:\/|$)|\/links\/|\/alarms\//.test(request.url())) {
                mutations.push(request.url());
            }
        });
        await page.goto(url(baseURL, `/d/${uid}?var-counter=0&var-rotation=0`));
        await page.getByRole('tab', { name: 'Operations', exact: true }).click();
        const commandPanel = page.getByTestId('jaops-commanding-panel');
        for (const label of ['Battery 1 on', 'Battery 1 off', 'Configure payload', 'Reboot']) {
            await expect(commandPanel.getByRole('button', { name: label, exact: true })).toBeVisible();
            await expect(commandPanel.getByRole('button', { name: label, exact: true })).toBeEnabled();
        }
        const forms = Object.values(dashboard.spec.elements)
            .filter((element) => element.spec.vizConfig.group === 'jaops-varcommanding-panel')
            .flatMap((element) => Object.values(element.spec.vizConfig.spec.options.commandForms || {}) as any[]);
        const variableButton = (variable: string, mode: string, value: string) => {
            const matches = forms.filter(
                (form) => form.variableToSet === variable && form.changeMode === mode && form.valueToSet === value
            );
            expect(matches, `${variable}: exactly one ${mode} ${value} control`).toHaveLength(1);
            return page.getByRole('button', { name: matches[0].label, exact: true });
        };
        await variableButton('counter', 'add', '1').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-counter')).toBe('1');
        const counterInput = page.getByTestId('jaops-variable-setting-input');
        await counterInput.fill('7');
        await counterInput.press('Enter');
        await expect.poll(() => new URL(page.url()).searchParams.get('var-counter')).toBe('7');
        await variableButton('counter', 'change', '0').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-counter')).toBe('0');
        await variableButton('rotation', 'add', '90').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-rotation')).toBe('90');
        const image = page.getByTestId('jaops-telemetric-image-panel').locator('img').first();
        await expect(image).toBeVisible();
        await expect(image).toHaveCSS('transform', 'matrix(0, 1, -1, 0, 0, 0)');
        await variableButton('rotation', 'add', '-90').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-rotation')).toBe('0');
        await variableButton('rotation', 'add', '-90').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-rotation')).toBe('-90');
        await variableButton('rotation', 'change', '0').click();
        await expect.poll(() => new URL(page.url()).searchParams.get('var-rotation')).toBe('0');
        await expect(image).toHaveCSS('transform', 'matrix(1, 0, 0, 1, 0, 0)');
        expect(mutations, 'Variable setters must not issue spacecraft commands').toEqual([]);
    });
});
