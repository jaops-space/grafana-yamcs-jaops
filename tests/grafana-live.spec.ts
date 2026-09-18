import { expect, test } from '@grafana/plugin-e2e';

const datasourceName = 'JAOPS Yamcs Datasource';
const datasourceProvisioningFile = 'yamcs-jaops.yaml';
const quickstartEndpoint = 'myproject_realtime';

type LiveQueryCase = {
    title: string;
    queryType: string;
    panelType: string;
    parameter?: string;
    fields?: string[];
    automaticColors?: boolean;
};

type DashboardPanel = {
    id: number;
    title: string;
    type: string;
    datasource: {
        type: string;
        uid: string;
    };
    gridPos: {
        h: number;
        w: number;
        x: number;
        y: number;
    };
    targets: Array<Record<string, unknown>>;
    options?: Record<string, unknown>;
    fieldConfig?: Record<string, unknown>;
};

const liveQueryCases: LiveQueryCase[] = [
    {
        title: 'Graph realtime parameter',
        queryType: 'plot',
        panelType: 'timeseries',
        parameter: '/myproject/Battery1_Voltage',
        fields: [],
    },
    {
        title: 'Single realtime parameter',
        queryType: 'single',
        panelType: 'stat',
        parameter: '/myproject/Battery1_Voltage',
    },
    {
        title: 'Discrete realtime parameter',
        queryType: 'discrete',
        panelType: 'state-timeline',
        parameter: '/myproject/Mode_Safe',
        automaticColors: true,
    },
    {
        title: 'Image realtime parameter',
        queryType: 'image',
        panelType: 'table',
        parameter: '/myproject/Battery1_Voltage',
    },
    {
        title: 'Commanding query',
        queryType: 'commanding',
        panelType: 'table',
    },
    {
        title: 'Events stream',
        queryType: 'events',
        panelType: 'logs',
    },
    {
        title: 'Time stream',
        queryType: 'time',
        panelType: 'table',
    },
    {
        title: 'Command history stream',
        queryType: 'command-history',
        panelType: 'table',
    },
    {
        title: 'Alarms stream',
        queryType: 'alarms',
        panelType: 'table',
    },
    {
        title: 'Links stream',
        queryType: 'links',
        panelType: 'table',
    },
    {
        title: 'Endpoint stream demands',
        queryType: 'demands',
        panelType: 'table',
    },
    {
        title: 'Yamcs subscriptions',
        queryType: 'subscriptions',
        panelType: 'table',
    },
];

type ProvisionedDatasource = {
    name: string;
    uid: string;
    type: string;
};

function buildPanel(testCase: LiveQueryCase, index: number, datasource: ProvisionedDatasource): DashboardPanel {
    const columns = 3;
    const width = 8;
    const height = 8;
    const datasourceRef = { type: datasource.type, uid: datasource.uid };

    return {
        id: index + 1,
        title: testCase.title,
        type: testCase.panelType,
        datasource: datasourceRef,
        gridPos: {
            h: height,
            w: width,
            x: (index % columns) * width,
            y: Math.floor(index / columns) * height,
        },
        targets: [
            {
                refId: 'A',
                datasource: datasourceRef,
                endpoint: quickstartEndpoint,
                parameter: testCase.parameter ?? '',
                command: testCase.queryType === 'commanding' ? '/myproject/Reboot' : '',
                type: testCase.queryType,
                fields: testCase.fields ?? [],
                automaticColors: testCase.automaticColors ?? false,
                asVariable: false,
                customVariableString: false,
                endpointVariable: '$endpoint',
            },
        ],
        options: {},
        fieldConfig: { defaults: {}, overrides: [] },
    };
}

async function createDashboard(request: any, datasource: ProvisionedDatasource): Promise<{ uid: string }> {
    const uid = `jaops-live-e2e-${Date.now().toString(36)}`;
    const response = await request.post('/api/dashboards/db', {
        data: {
            dashboard: {
                uid,
                title: 'JAOPS Grafana Live query e2e',
                schemaVersion: 41,
                version: 0,
                refresh: '',
                time: {
                    from: 'now-5m',
                    to: 'now',
                },
                panels: liveQueryCases.map((queryCase, index) => buildPanel(queryCase, index, datasource)),
            },
            overwrite: true,
        },
    });

    expect(
        response.ok(),
        `dashboard setup should succeed: ${response.status()} ${await response.text()}`
    ).toBeTruthy();

    return { uid };
}

async function deleteDashboard(request: any, uid: string) {
    await request.delete(`/api/dashboards/uid/${uid}`, { timeout: 5000 }).catch(() => undefined);
}

test.describe('Grafana Live query paths', () => {
    test.beforeEach(async ({ page }) => {
        await page.addInitScript(() => {
            window.localStorage.setItem('grafana.whatsNew.dashboardShown', 'true');
            window.localStorage.setItem('grafana.whatsNew.datasourceShown', 'true');
            window.localStorage.setItem('grafana.whatsNewShown', 'true');
        });
    });

    test('all datasource query types run without Grafana Live channel address errors', async ({
        gotoDashboardPage,
        page,
        readProvisionedDataSource,
        request,
    }) => {
        test.setTimeout(90000);
        const datasource = await readProvisionedDataSource<ProvisionedDatasource>({
            fileName: datasourceProvisioningFile,
            name: datasourceName,
        });

        const consoleLiveErrors: string[] = [];
        page.on('console', (message) => {
            if (message.type() !== 'error') {
                return;
            }
            const text = message.text();
            if (/streaming channel error|invalid channel address/i.test(text)) {
                consoleLiveErrors.push(text);
            }
        });
        page.on('pageerror', (error) => {
            const text = error.message;
            if (/streaming channel error|invalid channel address/i.test(text)) {
                consoleLiveErrors.push(text);
            }
        });

        const dashboard = await createDashboard(request, datasource);

        try {
            const dashboardPage = await gotoDashboardPage({
                uid: dashboard.uid,
                timeRange: { from: 'now-5m', to: 'now' },
            });

            await dashboardPage.waitForPanelsQueriesToComplete({ scrollAll: true, timeout: 30000 });

            // Give Grafana enough time to render initial data, subscribe over
            // Grafana Live, and show the Live-channel error that regressed in
            // 1.1.1. Some query/panel combinations may legitimately show
            // domain-specific panel status (for example "No data" on an empty
            // event stream), so keep this test scoped to the Grafana Live
            // channel-address failure mode instead of asserting that every
            // representative panel is semantically populated.
            await page.waitForTimeout(10000);

            await expect(page.getByText(/Streaming channel error/i)).toHaveCount(0);
            await expect(page.getByText(/invalid channel address/i)).toHaveCount(0);
            expect(consoleLiveErrors).toEqual([]);
        } finally {
            await deleteDashboard(request, dashboard.uid);
        }
    });
});
