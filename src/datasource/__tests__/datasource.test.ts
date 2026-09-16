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
                bufferMaxLength: 5000,
                dataPointsRounding: 500,
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
        expect(streamArg.buffer.maxLength).toBe(5000);
        expect(streamArg.addr.scope).toBe(LiveChannelScope.DataSource);
        expect(streamArg.addr.stream).toBe('jaops-yamcs-main');
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature');
        expect(streamArg.addr.data.from).toBe(1000);
        expect(streamArg.addr.data.to).toBe(2000);
    });

    it('uses append action for plot queries', async () => {
        const ds = buildDatasource();

        await firstValueFrom(ds.query(buildRequest(QueryType.PLOT) as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.buffer.action).toBe(StreamingFrameAction.Append);
    });

    it('uses relative range and rounded data points in realtime plot live path', async () => {
        const ds = buildDatasource();

        await firstValueFrom(ds.query(buildRequest(QueryType.PLOT, { fields: ['max', 'min'] }) as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature/now-5m-now/500/fields=max-min');
        expect(streamArg.addr.data.points).toBe(500);
        expect(streamArg.addr.data.from).toBe(1000);
        expect(streamArg.addr.data.to).toBe(2000);
    });

    it('uses raw Unix range in stream path when dashboard range is not relative', async () => {
        const ds = buildDatasource();
        const request = buildRequest(QueryType.PLOT, { fields: ['max', 'min'] });
        request.range.raw = { from: '2026-08-04T10:00:00Z', to: '2026-08-04T10:05:00Z' };

        await firstValueFrom(ds.query(request as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature/1000-2000/500/fields=max-min');
    });

    it('rounds data points to the nearest configured bucket and caps by buffer max length', async () => {
        const ds = new DataSource({
            uid: 'jaops-yamcs-main',
            jsonData: {
                bufferMaxLength: 1000,
                dataPointsRounding: 500,
            },
        } as any);

        const request = buildRequest(QueryType.PLOT, { fields: [] });
        request.maxDataPoints = 1600;

        await firstValueFrom(ds.query(request as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature/now-5m-now/1000/fields=none');
        expect(streamArg.addr.data.points).toBe(1000);
    });

    it('never rounds data points down to 0 for a narrow/shrunk panel', async () => {
        // A panel scaled down enough that Grafana resolves maxDataPoints to
        // less than half of the configured rounding bucket used to round
        // down to 0 datapoints, which the backend/Yamcs rejects outright
        // ("invalid point count 0, must be between 1 and 10000") and killed
        // the stream subscription instead of just showing a smaller plot.
        const ds = buildDatasource();
        const request = buildRequest(QueryType.PLOT, { fields: [] });
        request.maxDataPoints = 100;

        await firstValueFrom(ds.query(request as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.data.points).toBeGreaterThan(0);
        expect(streamArg.addr.data.points).toBe(500);
    });

    it('uses a stable field segment for plot queries without min or max', async () => {
        const ds = buildDatasource();

        await firstValueFrom(ds.query(buildRequest(QueryType.PLOT, { fields: [] }) as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature/now-5m-now/500/fields=none');
    });

    it('opens a single batched live stream for a multi-parameter plot query', async () => {
        const ds = buildDatasource();

        await firstValueFrom(
            ds.query(
                buildRequest(QueryType.PLOT, {
                    parameter: '/drone/BatteryPackVoltage',
                    parameters: ['/drone/BatteryPackVoltage', '/drone/Motors[0].rpm', '/drone/Attitude.yaw'],
                    fields: [],
                }) as any
            )
        );

        // One Live channel/backend goroutine for the whole query, not one per
        // parameter - see RunMultiParameterStream.
        expect(getDataStreamMock).toHaveBeenCalledTimes(1);
        const streamArg = getDataStreamMock.mock.calls[0][0];
        // A short hash, not the concatenated parameter names: Grafana Live
        // channel paths go through Centrifuge's channel length limit, so a
        // wide multi-parameter query must stay well under it regardless of
        // how many/how long its parameter names are - see hashParameterList.
        expect(streamArg.addr.path).toMatch(/^myproject_realtime\/multi-[0-9a-z]+\/now-5m-now\/500\/fields=none$/);
        expect(streamArg.addr.path.length).toBeLessThan(100);
        // parameter stays parameters[0] so backend code that only knows a single
        // parameter (validation, the initial historical frame) keeps working.
        expect(streamArg.addr.data.parameter).toBe('/drone/BatteryPackVoltage');
        expect(streamArg.addr.data.parameters).toEqual([
            '/drone/BatteryPackVoltage',
            '/drone/Motors[0].rpm',
            '/drone/Attitude.yaw',
        ]);
    });

    it('supports comma or newline separated legacy parameter text for multi-parameter plot queries', async () => {
        const ds = buildDatasource();

        await firstValueFrom(
            ds.query(
                buildRequest(QueryType.PLOT, {
                    parameter: '/drone/BatteryPackVoltage, /drone/BatteryPackCurrent\n/drone/BatteryTemperature',
                    fields: [],
                }) as any
            )
        );

        expect(getDataStreamMock).toHaveBeenCalledTimes(1);
        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.data.parameters).toEqual([
            '/drone/BatteryPackVoltage',
            '/drone/BatteryPackCurrent',
            '/drone/BatteryTemperature',
        ]);
    });

    it('keeps the Live channel path short for a wide multi-parameter query, regardless of parameter name length', async () => {
        // Regression test: concatenating every parameter name into the path
        // (the original implementation) silently exceeded Grafana Live's
        // Centrifuge channel length limit for a query this wide, so the
        // subscribe never even reached the backend - no error, just zero
        // data forever. 20 long parameter names is exactly the shape that
        // triggered it.
        const ds = buildDatasource();
        const parameters = Array.from({ length: 20 }, (_, i) => `/drone/SomeReasonablyLongParameterName_${i}`);

        await firstValueFrom(
            ds.query(buildRequest(QueryType.PLOT, { parameter: parameters[0], parameters, fields: [] }) as any)
        );

        expect(getDataStreamMock).toHaveBeenCalledTimes(1);
        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path.length).toBeLessThan(100);
        expect(streamArg.addr.data.parameters).toEqual(parameters);
    });

    it('still opens a single plain live stream for a single-parameter plot query', async () => {
        // A single parameter is the degenerate case of the same multi-parameter
        // wire shape (parameters: [one entry]), not a separate code path - see
        // RunParameterStream. The channel path still uses the readable
        // single-name scheme rather than a hash, since there's no collision risk
        // to guard against with only one candidate name.
        const ds = buildDatasource();

        await firstValueFrom(
            ds.query(
                buildRequest(QueryType.PLOT, {
                    parameter: '/drone/BatteryPackVoltage',
                    fields: [],
                }) as any
            )
        );

        expect(getDataStreamMock).toHaveBeenCalledTimes(1);
        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-drone-BatteryPackVoltage/now-5m-now/500/fields=none');
        expect(streamArg.addr.data.parameters).toEqual(['/drone/BatteryPackVoltage']);
    });

    it('shares one Live channel for the same parameter set requested in a different order', async () => {
        const ds = buildDatasource();

        await firstValueFrom(
            ds.query(
                buildRequest(QueryType.PLOT, {
                    parameter: '/drone/BatteryPackVoltage',
                    parameters: ['/drone/BatteryPackVoltage', '/drone/Motors[0].rpm', '/drone/Attitude.yaw'],
                    fields: [],
                }) as any
            )
        );
        const firstPath = getDataStreamMock.mock.calls[0][0].addr.path;

        getDataStreamMock.mockClear();
        await firstValueFrom(
            ds.query(
                buildRequest(QueryType.PLOT, {
                    parameter: '/drone/Attitude.yaw',
                    parameters: ['/drone/Attitude.yaw', '/drone/BatteryPackVoltage', '/drone/Motors[0].rpm'],
                    fields: [],
                }) as any
            )
        );
        const secondPath = getDataStreamMock.mock.calls[0][0].addr.path;

        expect(secondPath).toBe(firstPath);
    });

    it('includes automatic color setting in discrete stream path and payload', async () => {
        const ds = buildDatasource();

        await firstValueFrom(ds.query(buildRequest(QueryType.DISCRETE, { automaticColors: true }) as any));

        const streamArg = getDataStreamMock.mock.calls[0][0];
        expect(streamArg.addr.path).toBe('myproject_realtime/-sim-temperature/now-5m-now/500/colors=auto');
        expect(streamArg.addr.data.automaticColors).toBe(true);
    });

    it('maps core query types to expected stream paths and buffering actions', async () => {
        const ds = buildDatasource();
        const cases: Array<{ type: QueryType; expectedPath: string; expectedAction: StreamingFrameAction }> = [
            {
                type: QueryType.EVENTS,
                expectedPath: 'myproject_realtime/events',
                expectedAction: StreamingFrameAction.Append,
            },
            {
                type: QueryType.DEMANDS,
                expectedPath: 'myproject_realtime/demands',
                expectedAction: StreamingFrameAction.Replace,
            },
            {
                type: QueryType.SUBSCRIPTIONS,
                expectedPath: 'myproject_realtime/subscriptions',
                expectedAction: StreamingFrameAction.Replace,
            },
            {
                type: QueryType.COMMAND_HISTORY,
                expectedPath: 'myproject_realtime/commands/now-5m-now',
                expectedAction: StreamingFrameAction.Append,
            },
            {
                type: QueryType.ALARMS,
                expectedPath: 'myproject_realtime/alarms',
                expectedAction: StreamingFrameAction.Replace,
            },
            {
                type: QueryType.LINKS,
                expectedPath: 'myproject_realtime/links',
                expectedAction: StreamingFrameAction.Replace,
            },
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
                {
                    refId: 'B',
                    endpoint: 'myproject_realtime',
                    type: QueryType.PLOT,
                    parameter: '/sim/temperature',
                    command: '',
                    asVariable: false,
                },
            ],
        };

        await firstValueFrom(ds.query(request as any));
        expect(getDataStreamMock).toHaveBeenCalledTimes(1);
    });

    it('resolves endpoint from variable mode', async () => {
        const ds = buildDatasource();
        templateReplaceMock.mockImplementation((value: string) =>
            value === '$ENDPOINT' ? 'myproject_realtime' : value
        );

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
        expect(streamArg.addr.path).toContain('myproject_realtime/');
    });
});
