import {
    CoreApp,
    DataQueryRequest,
    DataQueryResponse,
    DataSourceInstanceSettings,
    LiveChannelScope,
    LoadingState,
    StreamingFrameAction,
} from '@grafana/data';

import { DataSourceWithBackend, getGrafanaLiveSrv, getTemplateSrv } from '@grafana/runtime';

import { Observable, merge } from 'rxjs';
import { tap } from 'rxjs/operators';
import { Configuration, DEFAULT_QUERY as DefaultQuery, DefaultConfiguration, Query, QueryType } from './types';

/**
 * Opt-in counter of data actually delivered to this browser tab over Grafana
 * Live, i.e. what panels really receive after Grafana Live's client-side
 * channel sharing/fan-out - as opposed to backend-side production metrics,
 * which only see one send per unique backend stream, even when many panels
 * share it. Only accumulates once a benchmark harness installs
 * `window.__jaopsBenchmarkStats` (see tests/benchmark), so it is a no-op
 * (single property check) for every other user.
 */
declare global {
    interface Window {
        __jaopsBenchmarkStats?: {
            framesReceived: number;
            valuesReceived: number;
        };
    }
}

/**
 * Grafana Live's client-side `getDataStream()` re-emits the *entire*
 * accumulated streaming buffer on every tick (see
 * `GrafanaLiveService.getDataStream` in grafana/public/app/features/live/live.ts),
 * not just the values that arrived since the previous emission. Naively
 * summing `frame.length` across emissions therefore counts already-seen rows
 * again and again as the buffer grows, wildly inflating the rate. Track the
 * previous frame length per subscription and only count the delta (the
 * genuinely new rows) on each tick.
 */
function createBenchmarkFrontendTap(): (response: DataQueryResponse) => void {
    let previousLength = 0;
    return (response: DataQueryResponse) => {
        for (const frame of response.data ?? []) {
            const length = frame.length ?? 0;
            // A shrink (e.g. schema change resets the buffer) means the whole
            // frame is effectively new; never count a negative delta.
            const delta = length >= previousLength ? length - previousLength : length;
            previousLength = length;

            const stats = typeof window !== 'undefined' ? window.__jaopsBenchmarkStats : undefined;
            if (stats && delta > 0) {
                stats.framesReceived += 1;
                stats.valuesReceived += delta;
            }
        }
    };
}

function formatAbsoluteRangePath(request: DataQueryRequest<Query>): string {
    const fromUnix = request.range.from.unix();
    const toUnix = request.range.to.unix();
    return `${fromUnix}-${toUnix}`;
}

function isRelativeTimeValue(value: unknown): value is string {
    return typeof value === 'string' && value.startsWith('now');
}

function formatRangePath(request: DataQueryRequest<Query>): string {
    const rawFrom = request.range.raw?.from;
    const rawTo = request.range.raw?.to;

    if (isRelativeTimeValue(rawFrom) && isRelativeTimeValue(rawTo)) {
        return `${encodeURIComponent(rawFrom)}-${encodeURIComponent(rawTo)}`;
    }

    return formatAbsoluteRangePath(request);
}

function formatGraphFieldsPath(query: Query): string {
    const fields = [...(query.fields ?? [])].filter((field) => field === 'min' || field === 'max').sort();
    return fields.length > 0 ? `fields=${fields.join('-')}` : 'fields=none';
}

function formatDiscreteOptionsPath(query: Query): string {
    return query.automaticColors ? 'colors=auto' : 'colors=none';
}

function roundDataPoints(maxDataPoints: number, dataPointsRounding: number, bufferMaxLength: number): number {
    const rounded = Math.round(maxDataPoints / dataPointsRounding) * dataPointsRounding;
    // Never resolve to 0 (or negative) datapoints. A panel shrunk small
    // enough that maxDataPoints falls under half of dataPointsRounding
    // (e.g. maxDataPoints=100 with the default 500-wide rounding bucket)
    // rounds down to 0 here, which the backend/Yamcs rejects outright
    // ("invalid point count 0, must be between 1 and 10000") and kills the
    // stream subscription entirely instead of just showing a smaller plot.
    return Math.min(Math.max(rounded, dataPointsRounding), bufferMaxLength);
}

/**
 * Custom Grafana DataSource for retrieving and streaming data.
 */
export class DataSource extends DataSourceWithBackend<Query, Configuration> {
    bufferMaxLength = DefaultConfiguration.bufferMaxLength;
    dataPointsRounding = DefaultConfiguration.dataPointsRounding;
    debugMode = false;

    constructor(instanceSettings: DataSourceInstanceSettings<Configuration>) {
        super(instanceSettings);
        this.bufferMaxLength = instanceSettings.jsonData.bufferMaxLength ?? this.bufferMaxLength;
        this.dataPointsRounding = Math.min(
            instanceSettings.jsonData.dataPointsRounding ?? this.dataPointsRounding,
            this.bufferMaxLength
        );
        this.debugMode = instanceSettings.jsonData.debugMode ?? this.debugMode;
    }

    /**
     * Returns the default query parameters.
     * @param app - The Grafana application context.
     * @returns A partial Query object with default values.
     */
    getDefaultQuery(_: CoreApp): Partial<Query> {
        return DefaultQuery;
    }

    /**
     * Processes the query request and retrieves data streams from Grafana Live.
     * @param request - The data query request from Grafana.
     * @returns An observable emitting the query response.
     */
    query(request: DataQueryRequest<Query>): Observable<DataQueryResponse> {
        const observables = request.targets
            .map((query) => {
                if ((!query.endpoint && !query.asVariable) || !query.type) {
                    return new Observable<DataQueryResponse>((subscriber) => {
                        subscriber.next({ data: [], state: LoadingState.NotStarted });
                        subscriber.complete();
                    });
                }

                // Commanding query type doesn't need streaming.
                // Commanding panel fetches command info via a resource call and sends commands via postResource.
                if (query.type === QueryType.COMMANDING) {
                    return new Observable<DataQueryResponse>((subscriber) => {
                        subscriber.next({ data: [], state: LoadingState.Done });
                        subscriber.complete();
                    });
                }

                let pathName = 'query';
                if (query.parameter) {
                    pathName = `${query.parameter.replaceAll('/', '-')}`;
                } else if (query.type === QueryType.EVENTS) {
                    pathName = `events`;
                } else if (query.type === QueryType.DEMANDS) {
                    pathName = `demands`;
                } else if (query.type === QueryType.SUBSCRIPTIONS) {
                    pathName = 'subscriptions';
                } else if (query.type === QueryType.COMMAND_HISTORY) {
                    pathName = `commands`;
                } else if (query.type === QueryType.ALARMS) {
                    pathName = `alarms`;
                } else if (query.type === QueryType.LINKS) {
                    pathName = `links`;
                }

                let action = StreamingFrameAction.Append;
                if (
                    query.type === QueryType.DEMANDS ||
                    query.type === QueryType.SUBSCRIPTIONS ||
                    query.type === QueryType.ALARMS ||
                    query.type === QueryType.LINKS
                ) {
                    action = StreamingFrameAction.Replace;
                }

                const templateSrv = getTemplateSrv();

                pathName = templateSrv.replace(pathName, request.scopedVars);
                query.parameter = templateSrv.replace(query.parameter, request.scopedVars);
                query.command = templateSrv.replace(query.command, request.scopedVars);

                if (query.asVariable) {
                    query.endpoint = templateSrv.replace(query.endpointVariable, request.scopedVars);
                }

                const fromUnix = request.range.from.unix();
                const toUnix = request.range.to.unix();
                const roundedMaxDataPoints = roundDataPoints(
                    request.maxDataPoints ?? this.dataPointsRounding,
                    this.dataPointsRounding,
                    this.bufferMaxLength
                );

                const pathParts = [query.endpoint, pathName];
                if (query.type === QueryType.PLOT) {
                    pathParts.push(
                        formatRangePath(request),
                        `${roundedMaxDataPoints}`,
                        formatGraphFieldsPath(query)
                    );
                } else if (query.type === QueryType.DISCRETE) {
                    pathParts.push(
                        formatRangePath(request),
                        `${roundedMaxDataPoints}`,
                        formatDiscreteOptionsPath(query)
                    );
                } else if (query.type === QueryType.COMMAND_HISTORY || query.type === QueryType.SINGLE) {
                    pathParts.push(formatRangePath(request));
                }

                return getGrafanaLiveSrv()
                    .getDataStream({
                        buffer: {
                            maxLength: this.bufferMaxLength,
                            action,
                        },
                        addr: {
                            scope: LiveChannelScope.DataSource,
                            stream: this.uid,
                            path: pathParts.join('/'),
                            data: {
                                ...query,
                                from: fromUnix,
                                to: toUnix,
                                points: roundedMaxDataPoints,
                            },
                        },
                    })
                    .pipe(tap(createBenchmarkFrontendTap()));
            })
            .filter(Boolean) as Array<Observable<DataQueryResponse>>; // Remove undefined values

        return merge(...observables);
    }
}
