import { LoadingState, LiveChannelScope, StreamingFrameAction } from '@grafana/data';
import { of } from 'rxjs';
import { firstValueFrom } from 'rxjs';

import { DataSource } from '../datasource';
import { QueryType } from '../types';

const getDataStreamMock = jest.fn();
const templateReplaceMock = jest.fn((value: string) => value);

jest.mock('@grafana/runtime', () => {
  const actual = jest.requireActual('@grafana/runtime');
  return {
    ...actual,
    getGrafanaLiveSrv: () => ({
      getDataStream: getDataStreamMock,
    }),
    getTemplateSrv: () => ({
      replace: templateReplaceMock,
    }),
  };
});

describe('DataSource.query', () => {
  beforeEach(() => {
    getDataStreamMock.mockReset();
    templateReplaceMock.mockClear();
    getDataStreamMock.mockReturnValue(of({ data: [], state: LoadingState.Done }));
  });

  const buildDatasource = () =>
    new DataSource({
      uid: 'jaops-yamcs-main',
      jsonData: {
        bufferMaxLength: 123,
      },
    } as any);

  const buildRequest = (type: QueryType, extra: Record<string, any> = {}) => ({
    targets: [
      {
        refId: 'A',
        endpoint: 'myproject_realtime',
        type,
        parameter: '/sim/temperature',
        command: '',
        asVariable: false,
        ...extra,
      },
    ],
    scopedVars: {},
    maxDataPoints: 321,
    range: {
      from: { unix: () => 1000 },
      to: { unix: () => 2000 },
      raw: { from: 'now-5m', to: 'now' },
    },
  });

  it('returns immediate done response for commanding type without streaming', async () => {
    const ds = buildDatasource();
    const response = await firstValueFrom(ds.query(buildRequest(QueryType.COMMANDING) as any));

    expect(response.state).toBe(LoadingState.Done);
    expect(response.data).toEqual([]);
    expect(getDataStreamMock).not.toHaveBeenCalled();
  });

  it('uses replace action for demands stream and omits range and max data points from live path', async () => {
    const ds = buildDatasource();

    await firstValueFrom(ds.query(buildRequest(QueryType.DEMANDS) as any));

    expect(getDataStreamMock).toHaveBeenCalledTimes(1);
    const streamArg = getDataStreamMock.mock.calls[0][0];

    expect(streamArg.buffer.action).toBe(StreamingFrameAction.Replace);
    expect(streamArg.buffer.maxLength).toBe(123);
    expect(streamArg.addr.scope).toBe(LiveChannelScope.DataSource);
    expect(streamArg.addr.stream).toBe('jaops-yamcs-main');
    expect(streamArg.addr.path).toBe('req/myproject_realtime/-sim-temperature');
    expect(streamArg.addr.data.from).toBe(1000);
    expect(streamArg.addr.data.to).toBe(2000);
    expect(streamArg.addr.data.realtime).toBe(true);
  });

  it('uses append action for plot queries', async () => {
    const ds = buildDatasource();

    await firstValueFrom(ds.query(buildRequest(QueryType.PLOT) as any));

    const streamArg = getDataStreamMock.mock.calls[0][0];
    expect(streamArg.buffer.action).toBe(StreamingFrameAction.Append);
  });

  it('includes range, max data points, and sorted min/max fields in plot live path', async () => {
    const ds = buildDatasource();

    await firstValueFrom(ds.query(buildRequest(QueryType.PLOT, { fields: ['max', 'min'] }) as any));

    const streamArg = getDataStreamMock.mock.calls[0][0];
    expect(streamArg.addr.path).toBe('req/myproject_realtime/-sim-temperature/now-5m-now/321/fields=max-min');
  });

  it('uses a stable field segment for plot queries without min or max', async () => {
    const ds = buildDatasource();

    await firstValueFrom(ds.query(buildRequest(QueryType.PLOT, { fields: [] }) as any));

    const streamArg = getDataStreamMock.mock.calls[0][0];
    expect(streamArg.addr.path).toBe('req/myproject_realtime/-sim-temperature/now-5m-now/321/fields=none');
  });

  it('maps core query types to expected stream paths and buffering actions', async () => {
    const ds = buildDatasource();
    const cases: Array<{ type: QueryType; expectedPath: string; expectedAction: StreamingFrameAction }> = [
      { type: QueryType.EVENTS, expectedPath: 'req/myproject_realtime/events', expectedAction: StreamingFrameAction.Append },
      { type: QueryType.DEMANDS, expectedPath: 'req/myproject_realtime/demands', expectedAction: StreamingFrameAction.Replace },
      { type: QueryType.SUBSCRIPTIONS, expectedPath: 'req/myproject_realtime/subscriptions', expectedAction: StreamingFrameAction.Replace },
      { type: QueryType.COMMAND_HISTORY, expectedPath: 'req/myproject_realtime/commands', expectedAction: StreamingFrameAction.Append },
      { type: QueryType.ALARMS, expectedPath: 'req/myproject_realtime/alarms', expectedAction: StreamingFrameAction.Replace },
      { type: QueryType.LINKS, expectedPath: 'req/myproject_realtime/links', expectedAction: StreamingFrameAction.Replace },
    ];

    for (const tc of cases) {
      getDataStreamMock.mockClear();
      await firstValueFrom(ds.query(buildRequest(tc.type, { parameter: '' }) as any));

      const streamArg = getDataStreamMock.mock.calls[0][0];
      expect(streamArg.addr.path).toBe(tc.expectedPath);
      expect(streamArg.buffer.action).toBe(tc.expectedAction);
    }
  });

  it('skips invalid targets and still streams valid ones', async () => {
    const ds = buildDatasource();
    const request = {
      ...buildRequest(QueryType.PLOT),
      targets: [
        { refId: 'A', endpoint: '', type: QueryType.PLOT, parameter: '', command: '', asVariable: false },
        { refId: 'B', endpoint: 'myproject_realtime', type: QueryType.PLOT, parameter: '/sim/temperature', command: '', asVariable: false },
      ],
    };

    await firstValueFrom(ds.query(request as any));
    expect(getDataStreamMock).toHaveBeenCalledTimes(1);
  });

  it('resolves endpoint from variable mode', async () => {
    const ds = buildDatasource();
    templateReplaceMock.mockImplementation((value: string) => (value === '$ENDPOINT' ? 'myproject_realtime' : value));

    await firstValueFrom(
      ds.query(
        buildRequest(QueryType.PLOT, {
          asVariable: true,
          endpoint: undefined,
          endpointVariable: '$ENDPOINT',
        }) as any
      )
    );

    const streamArg = getDataStreamMock.mock.calls[0][0];
    expect(streamArg.addr.path).toContain('req/myproject_realtime/');
  });
});
