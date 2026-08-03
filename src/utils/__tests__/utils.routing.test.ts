import { of } from 'rxjs';

import { getDatasourceResource, postDatasourceResource, prefixRoute } from '../utils.routing';

const fetchMock = jest.fn();

jest.mock('@grafana/runtime', () => {
  const actual = jest.requireActual('@grafana/runtime');
  return {
    ...actual,
    getBackendSrv: () => ({
      fetch: (...args: unknown[]) => fetchMock(...args),
    }),
  };
});

describe('utils.routing', () => {
  beforeEach(() => {
    fetchMock.mockReset();
  });

  it('prefixes plugin routes consistently', () => {
    expect(prefixRoute('how-to-use')).toBe('/a/jaops-yamcs-app/how-to-use');
    expect(prefixRoute('commanding-setup')).toBe('/a/jaops-yamcs-app/commanding-setup');
  });

  it('builds datasource GET resource URLs', async () => {
    fetchMock.mockReturnValue(of({ data: { ok: true } }));

    await getDatasourceResource('jaops-yamcs-main', 'fetch/endpoints');

    expect(fetchMock).toHaveBeenCalledWith({
      url: '/api/datasources/uid/jaops-yamcs-main/resources/fetch/endpoints',
      method: 'GET',
    });
  });

  it('builds datasource POST resource URLs with payload', async () => {
    fetchMock.mockReturnValue(of({ data: { ok: true } }));

    await postDatasourceResource('jaops-yamcs-main', 'endpoint/myproject_realtime/command/issue', { name: '/TEST/CMD' });

    expect(fetchMock).toHaveBeenCalledWith({
      url: '/api/datasources/uid/jaops-yamcs-main/resources/endpoint/myproject_realtime/command/issue',
      method: 'POST',
      data: { name: '/TEST/CMD' },
    });
  });
});
