import { expect, test, type APIRequestContext, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

// Requires the software-only Yamcs Demo MDB (/myproject, /features, /images and /yamcs).
// GRAFANA_URL may include a subpath, for example http://localhost:3000/grafana.
// Mutation tests require CANONICAL_SIMULATOR_COMMANDS=1 and verify the local fixture target.
// All other tests block Yamcs mutations, including accidental panel-initialization actions.
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
const tabNames = ['Overview', 'Telemetry', 'Mission data', 'Commanding', 'Operations', 'Diagnostics'];
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
    'jaops-commandhistory-panel',
    'jaops-alarms-panel',
    'jaops-timesync-panel',
    'jaops-commanding-panel',
    'jaops-varcommanding-panel',
    'jaops-links-panel',
    'jaops-staticimage-panel',
    'jaops-telemetricimage-panel',
    'histogram',
    'xychart',
];
const requiredQueryTypes = [
    'single',
    'plot',
    'discrete',
    'time',
    'command-history',
    'events',
    'alarms',
    'links',
    'image',
    'commanding',
    'demands',
    'subscriptions',
];
const pluginMarkers: Record<string, string> = {
    'jaops-commandhistory-panel': 'jaops-command-history-panel',
    'jaops-alarms-panel': 'jaops-alarms-panel',
    'jaops-timesync-panel': 'jaops-time-sync-panel',
    'jaops-commanding-panel': 'jaops-commanding-panel',
    'jaops-varcommanding-panel': 'jaops-variable-setting-panel',
    'jaops-links-panel': 'jaops-links-panel',
    'jaops-staticimage-panel': 'jaops-static-image-panel',
    'jaops-telemetricimage-panel': 'jaops-telemetric-image-panel',
};

interface Query {
    type: string;
    parameter?: string;
    parameters?: string[];
    endpoint?: string;
}

interface Panel {
    id: number;
    title: string;
    description?: string;
    data: {
        spec: {
            queries: Array<{
                spec: {
                    refId: string;
                    hidden?: boolean;
                    query: { group: string; datasource?: { name: string }; spec: Query };
                };
            }>;
        };
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

function parameters(query: Query, battery = '1') {
    const names = query.parameters?.length ? query.parameters : (query.parameter || '').split(/[\n,]+/).filter(Boolean);
    return names.map((name) => name.replace(/\$\{battery\}|\$battery\b/g, battery));
}

function canvasElements(panel: Panel) {
    const root = panel.vizConfig.spec.options.root as { elements: Array<Record<string, any>> };
    return root.elements.map((element) => {
        const { connections = [], ...stableElement } = element;
        return {
            ...stableElement,
            connections: connections.map((connection: Record<string, any>) => {
                const { sourceOriginal, targetOriginal, ...stableConnection } = connection;
                return stableConnection;
            }),
        };
    });
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

function queryProblems(panel: Panel, data: Snapshot | null, battery = '1'): string[] {
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
            // Command metadata is fetched through resources, not telemetry frames.
            continue;
        }
        const names = parameters(query, battery);
        const expected = names.length
            ? names
            : (
                  {
                      time: ['time'],
                      events: ['time', 'message', 'severity'],
                      alarms: ['alarms'],
                      'command-history': ['commands'],
                      links: ['name', 'status', 'disabled', 'dataInCount', 'dataOutCount'],
                      demands: ['parameter', 'stream_path', 'last_value_at'],
                      subscriptions: ['parameter'],
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
          : /Battery[12]_Temp$|DemoTemperature|DemoBundle\.battery\.temperatures\[\d+\]/.test(name)
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
          : name === '/features/DemoTemperature' ||
              /^\/myproject\/Battery[12]_Temp$/.test(name) ||
              /^\/features\/DemoBundle\.battery\.temperatures\[\d+\]$/.test(name)
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

async function checkRendering(content: Locator, panel: Panel) {
    const type = panel.vizConfig.group;
    await expect(content).toBeVisible();
    await expect(
        content.getByText(/Plugin unavailable|Panel plugin not found|Error loading panel|No data|Field not found/i)
    ).toHaveCount(0);
    if (pluginMarkers[type]) {
        await expect(content.getByTestId(pluginMarkers[type])).toBeVisible();
    }
    if (['timeseries', 'state-timeline', 'geomap', 'histogram', 'xychart'].includes(type)) {
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
    } else if (['jaops-staticimage-panel', 'jaops-telemetricimage-panel'].includes(type)) {
        const image = content.locator('img').first();
        await expect(image).toBeVisible();
        await expect
            .poll(() => image.evaluate((node: HTMLImageElement) => node.complete && node.naturalWidth > 0))
            .toBe(true);
        const imageBox = await image.boundingBox();
        const panelBox = await content.boundingBox();
        expect(imageBox).not.toBeNull();
        expect(panelBox).not.toBeNull();
        expect(imageBox!.height).toBeLessThanOrEqual(panelBox!.height + 1);
        expect(imageBox!.width).toBeLessThanOrEqual(panelBox!.width + 1);
    } else if (type === 'jaops-commanding-panel') {
        await expect(content.getByTestId('jaops-commanding-panel')).toHaveAttribute('data-command-count', '1');
        await expect(content.getByRole('button').first()).toBeEnabled();
        await expect(content).not.toContainText('No command configured');
    } else if (type === 'jaops-varcommanding-panel') {
        await expect(content.getByRole('textbox')).toBeVisible();
    } else if (type === 'jaops-links-panel') {
        await expect(content.getByRole('button', { name: 'Disable', exact: true }).first()).toBeEnabled();
        await expect(content.getByRole('button', { name: 'Reset', exact: true }).first()).toBeEnabled();
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
        if (queries(panel).some(({ query: candidate }) => ['demands', 'subscriptions'].includes(candidate.spec.type))) {
            // Diagnostic panels keep their sample subscription active but display only diagnostic rows.
            continue;
        }
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
        if (
            field.name === '/features/DemoEnabled' ||
            /^\/features\/DemoPowerChannels\[\d+\]\.healthy$/.test(field.name)
        ) {
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
    await checkRendering(content, panel);
    if (queries(panel).some(({ query }) => parameters(query.spec).includes('/features/DemoEnabled'))) {
        await expect(content).toContainText(/Enabled|Disabled/);
    }
    await expect(page.getByTestId('data-testid Panel status error')).toHaveCount(0);
}

const mutationAttempts = new WeakMap<Page, string[]>();
const permittedCommands = new WeakMap<Page, Set<string>>();

test.describe('Yamcs Demo dashboard', () => {
    test.skip(
        process.env.DEMO_DASHBOARDS_E2E !== '1',
        'Opt-in demo suite: set DEMO_DASHBOARDS_E2E=1 and provision the Yamcs Demo dashboard'
    );

    test.beforeEach(async ({ page }) => {
        const attempts: string[] = [];
        mutationAttempts.set(page, attempts);
        await page.route('**/*', async (route) => {
            const request = route.request();
            const path = new URL(request.url()).pathname;
            if (
                !['GET', 'HEAD', 'OPTIONS'].includes(request.method()) &&
                (/\/resources\/endpoint\//.test(path) ||
                    /\/api\/(?:processors|links|archive|events|mdb|storage|instances|services|parameter-lists|timeline|activities)(?:\/|$)/.test(
                        path
                    ))
            ) {
                if (
                    request.method() === 'POST' &&
                    path.endsWith('/resources/endpoint/demo_realtime/command/issue') &&
                    permittedCommands.get(page)?.has(request.postDataJSON()?.name)
                ) {
                    await route.continue();
                    return;
                }
                attempts.push(`${request.method()} ${path}`);
                await route.abort('blockedbyclient');
                return;
            }
            await route.continue();
        });
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

    test.afterEach(async ({ page }) => {
        expect(
            mutationAttempts.get(page),
            'No Yamcs mutation may be attempted, even during panel initialization'
        ).toEqual([]);
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
        expect(deployed.spec.variables).toEqual(provisioned.spec.variables);
        expect(Object.keys(deployed.spec.elements).sort()).toEqual(Object.keys(provisioned.spec.elements).sort());
        // Grafana canonicalizes visualization metadata on read; verify stable panel data and rendered UI separately.
        for (const [key, provisionedElement] of Object.entries(provisioned.spec.elements)) {
            const deployedPanel = deployed.spec.elements[key].spec;
            const provisionedPanel = provisionedElement.spec;
            expect(deployedPanel.id).toBe(provisionedPanel.id);
            expect(deployedPanel.title).toBe(provisionedPanel.title);
            expect(deployedPanel.description).toBe(provisionedPanel.description);
            expect(deployedPanel.vizConfig.group).toBe(provisionedPanel.vizConfig.group);
            expect(deployedPanel.data).toEqual(provisionedPanel.data);
            if (provisionedPanel.vizConfig.group === 'canvas') {
                expect(canvasElements(deployedPanel)).toEqual(canvasElements(provisionedPanel));
            }
            if (provisionedPanel.vizConfig.group === 'text') {
                expect(deployedPanel.vizConfig.spec.options.content).toBe(
                    provisionedPanel.vizConfig.spec.options.content
                );
            }
        }
        const referenced = panelKeys(deployed.spec.layout);
        expect([...referenced].sort(), 'No untested/orphaned panels or duplicated references').toEqual(
            Object.keys(deployed.spec.elements).sort()
        );
        const panels = Object.values(provisioned.spec.elements).map((element) => element.spec);
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
            for (const {
                spec: { query },
            } of panel.data.spec.queries) {
                expect(query.group).toBe('jaops-yamcs-datasource');
                expect(query.datasource?.name).toBe('jaops-yamcs-main');
                expect(query.spec.endpoint).toBe('demo_realtime');
                if (['single', 'plot', 'discrete'].includes(query.spec.type)) {
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
            await page.goto(url(baseURL, `/d/${uid}?from=now-5m&to=now&var-battery=1`));
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

    test('navigation never issues commands and the battery selector changes telemetry queries', async ({
        page,
        request,
        baseURL,
    }) => {
        test.setTimeout(90000);
        const dashboard = await loadDashboard(request, baseURL);
        const sourceResponse = await request.get(url(baseURL, '/api/datasources'));
        expect(sourceResponse.ok()).toBe(true);
        const sources = await sourceResponse.json();
        const source = sources.find(
            (candidate: any) =>
                candidate.uid === 'jaops-yamcs-main' &&
                candidate.type === 'jaops-yamcs-datasource' &&
                candidate.jsonData.endpoints?.demo_realtime
        );
        expect(source, 'Canonical datasource with demo_realtime must be provisioned').toBeTruthy();
        expect(dashboard.spec.variables.map((variable) => variable.spec.name)).toEqual([
            'battery',
            'payload_mode',
            'payload_gain',
        ]);
        await page.goto(url(baseURL, `/d/${uid}?from=now-5m&to=now&var-battery=1`));
        for (const name of tabNames) {
            await page.getByRole('tab', { name, exact: true }).click();
            await expect(page.getByRole('tab', { name, exact: true })).toHaveAttribute('aria-selected', 'true');
        }

        const telemetry = tabs(dashboard).find((tab) => tab.spec.title === 'Telemetry')!;
        const keys = panelKeys(telemetry);
        const selected = keys.filter((key) =>
            queries(dashboard.spec.elements[key].spec).some(({ query }) => query.spec.parameter?.includes('${battery}'))
        );
        expect(selected, 'Both latest-value battery panels exercise the viewer-only variable').toHaveLength(2);
        await page.getByRole('tab', { name: 'Telemetry', exact: true }).click();
        for (const battery of ['1', '2']) {
            if (battery === '2') {
                const selector = page.getByRole('combobox', { name: /Battery/ });
                await selector.click();
                await selector.press('ArrowDown');
                await selector.press('Enter');
                await expect.poll(() => new URL(page.url()).searchParams.get('var-battery')).toBe('2');
            }
            for (const key of selected) {
                const panel = dashboard.spec.elements[key].spec;
                await expect
                    .poll(() => snapshot(page, key).then((data) => queryProblems(panel, data, battery)), {
                        timeout: 30000,
                    })
                    .toEqual([]);
                const data = (await snapshot(page, key))!;
                const fields = data.rawFrames.flatMap((frame) => frame.fields);
                for (const { query } of queries(panel)) {
                    const name = parameters(query.spec, battery)[0];
                    expect(fields.find((field) => field.name === name)?.values.some((value) => value !== null)).toBe(
                        true
                    );
                    expect(
                        fields.some(
                            (field) =>
                                field.name ===
                                name.replace(`Battery${battery}`, `Battery${battery === '1' ? '2' : '1'}`)
                        )
                    ).toBe(false);
                }
                fields.forEach(checkFieldMetadata);
            }
            const comparisons = keys.filter((key) => {
                const panel = dashboard.spec.elements[key].spec;
                return (
                    panel.vizConfig.group === 'timeseries' &&
                    queries(panel).some(
                        ({ query }) =>
                            query.spec.parameter === '/myproject/Battery1_Voltage' ||
                            query.spec.parameter === '/myproject/Battery1_Temp'
                    )
                );
            });
            expect(comparisons).toHaveLength(2);
            for (const key of comparisons) {
                await expect
                    .poll(
                        () =>
                            snapshot(page, key).then((data) => queryProblems(dashboard.spec.elements[key].spec, data)),
                        { timeout: 30000 }
                    )
                    .toEqual([]);
            }
        }
    });

    test('simulator commands submit typed arguments and appear in command history', async ({
        page,
        request,
        baseURL,
    }) => {
        test.skip(process.env.CANONICAL_SIMULATOR_COMMANDS !== '1', 'Explicit local simulator command opt-in required');
        const sourceResponse = await request.get(url(baseURL, '/api/datasources/uid/jaops-yamcs-main'));
        expect(sourceResponse.ok()).toBe(true);
        const source = await sourceResponse.json();
        const endpoint = source.jsonData.endpoints.demo_realtime;
        expect(endpoint.instance).toBe('demo');
        expect(endpoint.processor).toBe('realtime');
        expect(source.jsonData.hosts[endpoint.host].path).toBe('docker.gateway:8093');
        const linkURL = 'http://localhost:8093/api/links/demo/udp-out';
        const linkBeforeResponse = await request.get(linkURL);
        expect(linkBeforeResponse.ok()).toBe(true);
        const sentBefore = Number((await linkBeforeResponse.json()).dataOutCount);
        const commandNames = [
            '/myproject/SwitchVoltageOn',
            '/myproject/SwitchVoltageOff',
            '/myproject/ConfigurePayload',
        ];
        permittedCommands.set(page, new Set(commandNames));
        await page.goto(
            url(baseURL, `/d/${uid}?views-dtab=commanding&var-battery=1&var-payload_mode=IMAGING&var-payload_gain=2`)
        );
        const gainInput = page.getByTestId('jaops-variable-setting-panel').getByRole('textbox');
        await gainInput.fill('4');
        await gainInput.press('Enter');
        await expect.poll(() => new URL(page.url()).searchParams.get('var-payload_gain')).toBe('4');
        const buttons = ['Battery 1 ON', 'Battery 1 OFF', 'Apply IMAGING / gain 4'];
        for (const [index, name] of buttons.entries()) {
            const responsePromise = page.waitForResponse(
                (response) =>
                    response.url().endsWith('/resources/endpoint/demo_realtime/command/issue') &&
                    response.request().method() === 'POST'
            );
            await page.getByRole('button', { name, exact: true }).click();
            const response = await responsePromise;
            expect(response.ok(), await response.text()).toBe(true);
            const payload = response.request().postDataJSON();
            expect(payload.name).toBe(commandNames[index]);
            expect(payload.arguments).toEqual(index < 2 ? { Battery: 1 } : { Mode: 'IMAGING', Gain: 4 });
        }
        await page.reload();
        const history = page.getByTestId('jaops-command-history-panel');
        await expect(history).toContainText('ConfigurePayload', { timeout: 20000 });
        await expect(history).toContainText('SwitchVoltageOff');
        await expect(history).toContainText('SwitchVoltageOn');
        await expect
            .poll(
                async () => {
                    const response = await request.get(linkURL);
                    expect(response.ok()).toBe(true);
                    return Number((await response.json()).dataOutCount);
                },
                { timeout: 20000, message: 'Battery commands must actually leave the Yamcs UDP link' }
            )
            .toBeGreaterThanOrEqual(sentBefore + 2);
    });

    test('all six mission views fit the FHD presentation viewport', async ({ page, request, baseURL }) => {
        test.setTimeout(180000);
        const dashboard = await loadDashboard(request, baseURL);
        await page.goto(url(baseURL, `/d/${uid}?kiosk=true`));
        for (const name of tabNames) {
            await page.getByRole('tab', { name, exact: true }).click();
            const panels = page.getByTestId('data-testid panel content');
            const keys = panelKeys(tabs(dashboard).find((tab) => tab.spec.title === name)!);
            const expectedCount = keys.length + 3;
            await expect(panels).toHaveCount(expectedCount);
            await expect(panels.first()).toBeVisible();
            await expect
                .poll(
                    async () =>
                        (
                            await Promise.all(
                                keys.map(async (key) =>
                                    queryProblems(dashboard.spec.elements[key].spec, await snapshot(page, key))
                                )
                            )
                        ).flat(),
                    { timeout: 30000 }
                )
                .toEqual([]);
            const bounds = await panels.evaluateAll((nodes) =>
                nodes.map((node) => {
                    const rect = node.getBoundingClientRect();
                    return { left: rect.left, right: rect.right, bottom: rect.bottom };
                })
            );
            for (const rect of bounds) {
                expect.soft(rect.left, `${name}: panel left edge`).toBeGreaterThanOrEqual(0);
                expect.soft(rect.right, `${name}: panel right edge`).toBeLessThanOrEqual(1920);
                expect.soft(rect.bottom, `${name}: panel bottom edge`).toBeLessThanOrEqual(1060);
            }
            await test.info().attach(`${name}-fhd`, {
                body: await page.screenshot({ path: test.info().outputPath(`${name}-FHD.png`) }),
                contentType: 'image/png',
            });
        }
    });
});
