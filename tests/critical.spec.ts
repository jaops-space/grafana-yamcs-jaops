import { expect, test } from '@grafana/plugin-e2e';
import type { Page } from '@playwright/test';

const appPluginId = 'jaops-yamcs-app';
const datasourcePluginId = 'jaops-yamcs-datasource';
const datasourceUid = 'jaops-yamcs-main';
const datasourceName = 'JAOPS Yamcs Datasource';
const quickstartEndpoint = 'myproject_realtime';

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
    'JAOPS Commanding Panel',
    'JAOPS Command History Panel',
    'JAOPS Telemetric Image Panel',
    'JAOPS Static Image Panel',
    'JAOPS Variable Setting Panel',
    'JAOPS Alarms Panel',
    'JAOPS Links Panel',
    'JAOPS Yamcs Time Sync',
];

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

    test('datasource query editor is configurable in a Grafana panel editor', async ({ panelEditPage, page }) => {
        test.setTimeout(60000);

        await panelEditPage.datasource.set(datasourceName);
        const queryEditor = page.getByTestId('jaops-query-editor').first();
        await expect(queryEditor).toBeVisible({ timeout: 20000 });
        await expect(queryEditor.getByTestId('jaops-query-type-select')).toBeVisible();
        await expect(queryEditor.getByTestId('jaops-parameter-select')).toBeVisible();

        const endpointsResponse = page.waitForResponse(
            (response) => response.url().includes('/resources/fetch/endpoints') && response.ok(),
            { timeout: 30000 }
        );
        await queryEditor.getByTestId('jaops-query-editor-fetch-endpoints').click();
        await endpointsResponse;

        await queryEditor.locator('label').filter({ hasText: 'As variable' }).click();
        await expect(queryEditor.getByLabel('Custom string')).toBeVisible();
        await queryEditor.locator('label').filter({ hasText: 'Custom string' }).click();
        await expect(queryEditor.getByText('Endpoint Variable')).toBeVisible();

        await queryEditor.getByTestId('jaops-query-editor-run-query').click();
        await expect(queryEditor).toBeVisible();
    });

    for (const panelName of panelRenderChecks) {
        test(`${panelName} renders in Grafana panel editor`, async ({ panelEditPage }) => {
            test.setTimeout(60000);

            await panelEditPage.setVisualization(panelName);
            await expect(panelEditPage.getVisualizationName()).toContainText(panelName);
            await expect(panelEditPage.panel.locator).toBeVisible({ timeout: 20000 });
            await expect(panelEditPage.panel.getErrorIcon()).toHaveCount(0);
        });
    }
});
