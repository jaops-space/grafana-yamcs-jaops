import { expect, test } from '@grafana/plugin-e2e';
import type { Page } from '@playwright/test';

const appPluginId = 'jaops-yamcs-app';
const datasourcePluginId = 'jaops-yamcs-datasource';
const datasourceUid = 'jaops-yamcs-main';
const datasourceName = 'JAOPS Yamcs Datasource';
const quickstartEndpoint = 'myproject_realtime';

type DashboardPanel = {
    id: number;
    title: string;
    type: string;
    datasource?: {
        type: string;
        uid: string;
    };
    targets?: Array<Record<string, unknown>>;
};

const pluginIds = [
    appPluginId,
    datasourcePluginId,
    'jaops-commanding-panel',
    'jaops-commandhistory-panel',
    'jaops-telemetricimage-panel',
    'jaops-staticimage-panel',
    'jaops-variables-panel',
    'jaops-alarms-panel',
    'jaops-links-panel',
    'jaops-timesync-panel',
];

const panelRenderChecks = [
    { name: 'JAOPS Commanding Panel', type: 'jaops-commanding-panel', marker: 'jaops-commanding-panel' },
    {
        name: 'JAOPS Command History Panel',
        type: 'jaops-commandhistory-panel',
        marker: 'jaops-command-history-panel',
    },
    {
        name: 'JAOPS Telemetric Image Panel',
        type: 'jaops-telemetricimage-panel',
        marker: 'jaops-telemetric-image-panel',
    },
    { name: 'JAOPS Static Image Panel', type: 'jaops-staticimage-panel', marker: 'jaops-static-image-panel' },
    { name: 'JAOPS Variable Setting Panel', type: 'jaops-variables-panel', marker: 'jaops-variable-setting-panel' },
    { name: 'JAOPS Alarms Panel', type: 'jaops-alarms-panel', marker: 'jaops-alarms-panel' },
    { name: 'JAOPS Links Panel', type: 'jaops-links-panel', marker: 'jaops-links-panel' },
    { name: 'JAOPS Yamcs Time Sync', type: 'jaops-timesync-panel', marker: 'jaops-time-sync-panel' },
];

async function createDashboardWithPanel(request: any, panel: DashboardPanel) {
    const uid = `jaops-e2e-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
    const response = await request.post('/api/dashboards/db', {
        data: {
            dashboard: {
                uid,
                title: `JAOPS E2E ${panel.title}`,
                schemaVersion: 41,
                version: 0,
                refresh: false,
                time: {
                    from: 'now-5m',
                    to: 'now',
                },
                panels: [
                    {
                        gridPos: { h: 8, w: 12, x: 0, y: 0 },
                        ...panel,
                    },
                ],
            },
            overwrite: true,
        },
    });

    expect(
        response.ok(),
        `dashboard setup for ${panel.title} should succeed: ${response.status()} ${await response.text()}`
    ).toBeTruthy();

    return { uid };
}

async function deleteDashboard(request: any, dashboard: { uid: string }) {
    await request.delete(`/api/dashboards/uid/${dashboard.uid}`, { timeout: 5000 }).catch(() => undefined);
}

async function dismissGrafanaModals(page: Page) {
    for (let attempt = 0; attempt < 4; attempt++) {
        await page.waitForTimeout(500);
        await page.keyboard.press('Escape').catch(() => undefined);

        const dialog = page.getByRole('dialog').filter({ hasText: "What's new in Grafana" });
        const closeButton = dialog.getByRole('button', { name: 'Close' }).first();
        if (await closeButton.isVisible().catch(() => false)) {
            await closeButton.click({ force: true });
            await expect(dialog)
                .toBeHidden({ timeout: 5000 })
                .catch(() => undefined);
            continue;
        }

        if (!(await dialog.isVisible().catch(() => false))) {
            return;
        }
    }
}

test.describe('critical plugin paths', () => {
    test.describe.configure({ mode: 'serial' });

    test.beforeEach(async ({ page }) => {
        await page.addInitScript(() => {
            window.localStorage.setItem('grafana.whatsNew.dashboardShown', 'true');
            window.localStorage.setItem('grafana.whatsNew.datasourceShown', 'true');
            window.localStorage.setItem('grafana.whatsNewShown', 'true');
        });
    });

    test('plugin app, datasource, and panels are registered in Grafana', async ({ request }) => {
        for (const pluginId of pluginIds) {
            const response = await request.get(`/api/plugins/${pluginId}/settings`);
            expect(response.ok(), `${pluginId} should be readable from Grafana plugin settings`).toBeTruthy();

            const body = await response.json();
            expect(body.id).toBe(pluginId);
        }
    });

    test('all app setup pages load through stable plugin markers', async ({ page }) => {
        test.setTimeout(60000);
        const pages = [
            { route: '/a/jaops-yamcs-app/', marker: 'jaops-setup-page-overview' },
            { route: '/a/jaops-yamcs-app/how-to-use', marker: 'jaops-setup-page-how-to-use' },
            { route: '/a/jaops-yamcs-app/commanding-setup', marker: 'jaops-setup-page-commanding' },
            { route: '/a/jaops-yamcs-app/image-panel-setup', marker: 'jaops-setup-page-image' },
            { route: '/a/jaops-yamcs-app/variable-setup', marker: 'jaops-setup-page-variable' },
            { route: '/a/jaops-yamcs-app/time-sync-setup', marker: 'jaops-setup-page-time-sync' },
        ];

        for (const setupPage of pages) {
            await page.goto(setupPage.route);
            await dismissGrafanaModals(page);
            await expect(page.getByTestId(setupPage.marker)).toBeVisible({ timeout: 15000 });
        }
    });

    test('provisioned datasource is readable and editable', async ({ gotoDataSourceConfigPage, page, request }) => {
        const response = await request.get(`/api/datasources/uid/${datasourceUid}`);
        expect(response.ok()).toBeTruthy();

        const body = await response.json();
        expect(body.uid).toBe(datasourceUid);
        expect(body.name).toBe(datasourceName);
        expect(body.type).toBe(datasourcePluginId);
        expect(['docker.gateway:8090', 'yamcs:8090']).toContain(body.jsonData.hosts['main-host'].path);
        expect(body.jsonData.endpoints[quickstartEndpoint].instance).toBe('myproject');
        expect(body.jsonData.endpoints[quickstartEndpoint].processor).toBe('realtime');

        const configPage = await gotoDataSourceConfigPage(datasourceUid);
        await dismissGrafanaModals(page);
        await expect(page.getByTestId('jaops-datasource-config-editor')).toBeVisible();
        await expect(page.getByRole('button', { name: 'Add host' })).toBeVisible();
        await expect(page.getByRole('button', { name: 'Add endpoint' })).toBeVisible();
    });

    test('datasource health succeeds for the Yamcs quickstart realtime endpoint', async ({ request }) => {
        const sourceResponse = await request.get(`/api/datasources/uid/${datasourceUid}`);
        expect(sourceResponse.ok()).toBeTruthy();
        const source = await sourceResponse.json();

        const healthDatasourceUid = 'jaops-yamcs-e2e-health';
        await request.delete(`/api/datasources/uid/${healthDatasourceUid}`);

        const createResponse = await request.post('/api/datasources', {
            data: {
                uid: healthDatasourceUid,
                name: 'JAOPS Yamcs E2E Health',
                type: datasourcePluginId,
                access: 'proxy',
                isDefault: false,
                jsonData: {
                    hosts: source.jsonData.hosts,
                    endpoints: {
                        [quickstartEndpoint]: source.jsonData.endpoints[quickstartEndpoint],
                    },
                },
                secureJsonData: {
                    'main-host-password': 'admin',
                },
            },
        });
        expect(createResponse.ok()).toBeTruthy();

        try {
            const healthResponse = await request.get(`/api/datasources/uid/${healthDatasourceUid}/health`);
            await expect(healthResponse).toBeOK();
        } finally {
            await request.delete(`/api/datasources/uid/${healthDatasourceUid}`);
        }
    });

    test('datasource backend resources can reach Yamcs quickstart', async ({ request }) => {
        const resources = [
            {
                url: `/api/datasources/uid/${datasourceUid}/resources/fetch/endpoints`,
                verify: async (body: any) => {
                    expect(body[quickstartEndpoint].name).toBe('Quickstart RT');
                    expect(body[quickstartEndpoint].online).toBe(true);
                },
            },
            {
                url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${quickstartEndpoint}/parameters?q=Battery`,
                verify: async (body: any) => {
                    expect(Array.isArray(body)).toBeTruthy();
                    expect(body.some((name: string) => name.includes('Battery'))).toBeTruthy();
                },
            },
            {
                url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${quickstartEndpoint}/commands?q=Reboot`,
                verify: async (body: any) => {
                    expect(Array.isArray(body)).toBeTruthy();
                    expect(body.some((command: { name: string }) => command.name.includes('Reboot'))).toBeTruthy();
                },
            },
            {
                url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${quickstartEndpoint}/command/info?name=%2Fmyproject%2FReboot`,
                verify: async (body: any) => {
                    expect(body.qualifiedName || body.name).toContain('Reboot');
                },
            },
            {
                url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${quickstartEndpoint}/links`,
                verify: async (body: any) => {
                    expect(Array.isArray(body) || typeof body === 'object').toBeTruthy();
                },
            },
        ];

        for (const resource of resources) {
            const response = await request.get(resource.url);
            expect(response.ok(), `${resource.url} should return 2xx`).toBeTruthy();
            await resource.verify(await response.json());
        }
    });

    test('datasource query editor is configurable in Grafana Explore', async ({ explorePage, page }) => {
        test.setTimeout(60000);

        await explorePage.goto();
        await dismissGrafanaModals(page);

        const queryEditor = explorePage.getQueryEditorRow('A').getByTestId('jaops-query-editor');
        await expect(queryEditor).toBeVisible({ timeout: 20000 });
        await expect(queryEditor.getByTestId('jaops-query-type-select')).toBeVisible();
        await expect(queryEditor.getByTestId('jaops-parameter-select')).toBeVisible();

        await queryEditor.getByText('As variable').click({ timeout: 5000 });
        await expect(queryEditor.getByLabel('Custom string')).toBeVisible();
        await queryEditor.getByText('Custom string').click({ timeout: 5000 });
        await expect(queryEditor.getByText('Endpoint Variable')).toBeVisible();
    });

    for (const panel of panelRenderChecks) {
        test(`${panel.name} renders in Grafana panel editor`, async ({ gotoPanelEditPage, page, request }) => {
            test.setTimeout(60000);

            const dashboard = await createDashboardWithPanel(request, {
                id: 1,
                title: panel.name,
                type: panel.type,
                datasource: { type: datasourcePluginId, uid: datasourceUid },
                targets: [
                    {
                        refId: 'A',
                        datasource: { type: datasourcePluginId, uid: datasourceUid },
                    },
                ],
            });

            try {
                await gotoPanelEditPage({ dashboard, id: '1' });
                await dismissGrafanaModals(page);
                await expect(page.getByTestId(panel.marker)).toBeVisible({ timeout: 20000 });
                await expect(
                    page.getByText(/Plugin unavailable|Panel plugin not found|Error loading panel/i)
                ).toHaveCount(0);
            } finally {
                await deleteDashboard(request, dashboard);
            }
        });
    }
});
