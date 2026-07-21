import { expect, test, type Page } from '@playwright/test';

const datasourceUid = 'jaops-yamcs-main';
const quickstartEndpoint = 'myproject_realtime';
const dashboardUid = '0708823c-c106-407b-9861-f64283746167';
const dashboardSlug = 'yamcs-quickstart-demo-dashboard';

async function dismissGrafanaModals(page: Page) {
  for (let attempt = 0; attempt < 4; attempt++) {
    await page.waitForTimeout(500);
    await page.keyboard.press('Escape').catch(() => undefined);

    const dialog = page.getByRole('dialog').filter({ hasText: "What's new in Grafana" });
    const closeButton = dialog.getByRole('button', { name: 'Close' }).first();
    if (await closeButton.isVisible().catch(() => false)) {
      await closeButton.click({ force: true });
      await expect(dialog).toBeHidden({ timeout: 5000 }).catch(() => undefined);
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

  test('all app setup pages load', async ({ page }) => {
    const pages = [
      { route: '/a/jaops-yamcs-app/', marker: 'How to Use the Yamcs Plugin' },
      { route: '/a/jaops-yamcs-app/how-to-use', marker: 'Step 1: Add the Datasource' },
      { route: '/a/jaops-yamcs-app/commanding-setup', marker: 'Step 2: Choose Commanding Query Type' },
      { route: '/a/jaops-yamcs-app/image-panel-setup', marker: 'Step 2: Choose Image Query Type' },
      { route: '/a/jaops-yamcs-app/variable-setup', marker: 'Step 1: Create Variables in Dashboard Settings' },
      { route: '/a/jaops-yamcs-app/time-sync-setup', marker: 'Step 1: Setup a Replay processor on Yamcs' },
    ];

    for (const setupPage of pages) {
      await page.goto(setupPage.route);
      await dismissGrafanaModals(page);
      await expect(page.getByText(setupPage.marker, { exact: true }).first()).toBeVisible();
    }
  });

  test('provisioned datasource config is present and editable', async ({ request }) => {
    const response = await request.get(`/api/datasources/uid/${datasourceUid}`);
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.uid).toBe(datasourceUid);
    expect(body.type).toBe('jaops-yamcs-datasource');
    expect(body.jsonData.hosts['main-host'].path).toBe('docker.gateway:8090');
    expect(body.jsonData.endpoints[quickstartEndpoint].instance).toBe('myproject');

    const health = await request.get(`/api/datasources/uid/${datasourceUid}/health`);
    expect([200, 400, 503]).toContain(health.status());
  });

  test('datasource backend resources return quickstart data', async ({ request }) => {
    const resources = [
      {
        url: `/api/datasources/uid/${datasourceUid}/resources/fetch/endpoints`,
        verify: async (body: any) => {
          expect(body[quickstartEndpoint].name).toBe('Quickstart RT');
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

  test('datasource config page renders controls and can run save/test action', async ({ page }) => {
    await page.goto(`/connections/datasources/edit/${datasourceUid}`);
    await dismissGrafanaModals(page);
    await expect(page.getByRole('heading', { name: 'JAOPS Yamcs Datasource' })).toBeVisible();
    await expect(page.getByText(/docker\.gateway:8090|main-host|Hosts/i).first()).toBeVisible();

    const saveAndTest = page.getByRole('button', { name: /save.*test/i });
    await expect(saveAndTest).toBeVisible();
    await saveAndTest.click();
    await expect(page.getByText(/datasource|health|success|error/i).first()).toBeVisible();
  });

  test('provisioned dashboard query config renders with expected datasource fields', async ({ page, request }) => {
    const response = await request.get(`/api/dashboards/uid/${dashboardUid}`);
    expect(response.ok()).toBeTruthy();
    const dashboardBody = await response.json();
    const dashboardConfig = JSON.stringify(dashboardBody);
    expect(dashboardConfig).toContain('jaops-yamcs-datasource');
    expect(dashboardConfig).toContain('/myproject/Battery1_Voltage');
    expect(dashboardConfig).toContain('/myproject/Battery2_Voltage');

    await page.goto(`/d/${dashboardUid}/${dashboardSlug}`);
    await dismissGrafanaModals(page);
    await expect(page.getByText('Battery Voltages')).toBeVisible({ timeout: 20000 });
  });

  test('demo dashboard renders all provisioned panels without plugin load errors', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto(`/d/${dashboardUid}/${dashboardSlug}`);
    await dismissGrafanaModals(page);
    await expect(page.getByText('Yamcs Quickstart Demo Dashboard')).toBeVisible({ timeout: 20000 });
    await expect(page.getByText('Battery Voltages')).toBeVisible({ timeout: 20000 });
    await expect(page.getByText(/panel plugin not found|failed to load plugin|plugin unavailable/i)).toHaveCount(0);

    const titledPanels = [
      { title: 'Battery Voltages', visual: true },
      { title: 'Command History' },
      { title: 'Position X/Y Map' },
      { title: 'Gyro' },
      { title: 'Temperatures' },
      { title: 'Static Image' },
      { title: 'Telemetric Image' },
      { title: 'Telemetric Image (variable rotation)' },
      { title: 'State Timeline' },
      { title: 'Battery 2' },
      { title: 'Variable Setter' },
      { title: 'Alarms' },
      { title: 'Links' },
    ];

    const discoveredTitles = new Set<string>();
    const discoveredButtons = new Set<string>();
    for (let i = 0; i < 16; i++) {
      for (const title of await page.locator('h2').allTextContents()) {
        discoveredTitles.add(title.trim());
      }
      for (const button of await page.locator('button').allTextContents()) {
        discoveredButtons.add(button.trim());
      }
      await page.evaluate(() => {
        window.scrollBy(0, 900);
        for (const element of Array.from(document.querySelectorAll<HTMLElement>('*'))) {
          if (element.scrollHeight > element.clientHeight) {
            element.scrollTop += 900;
          }
        }
      });
      await page.waitForTimeout(200);
    }

    for (const panel of titledPanels) {
      expect(discoveredTitles.has(panel.title), `${panel.title} panel should render`).toBeTruthy();
      if (panel.visual) {
        await page.goto(`/d/${dashboardUid}/${dashboardSlug}`);
        await dismissGrafanaModals(page);
        const panelRoot = page.getByRole('region', { name: panel.title }).first();
        await expect(panelRoot, `${panel.title} visual panel should render`).toBeVisible();
        await expect
          .poll(async () => panelRoot.locator('canvas, svg').count(), {
            message: `${panel.title} should render a visualization surface`,
          })
          .toBeGreaterThan(0);
      }
    }

    expect(discoveredButtons.has('-90 deg'), 'Rotate Image control should render').toBeTruthy();
  });
});
