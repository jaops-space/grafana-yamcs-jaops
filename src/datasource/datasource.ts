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

import { Observable, ReplaySubject, Subscription, merge } from 'rxjs';
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
            parameterFieldsSeen?: Record<string, boolean>;
        };
        __jaopsBenchmarkNoRetention?: boolean;
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
            if (stats?.parameterFieldsSeen) {
                for (const field of frame.fields ?? []) {
                    if (field.name !== 'time') {
                        stats.parameterFieldsSeen[field.name] = true;
                    }
                }
            }
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

// Short deterministic hash (djb2) used to build a bounded-length Live
// channel path for a multi-parameter query. Grafana Live channel names go
// through Centrifuge, which enforces a channel length limit (~255 bytes);
// concatenating every parameter name directly (as a single-parameter path
// does) silently exceeds that for wide queries - the subscribe never even
// reaches the plugin backend, with no visible error. Hashing keeps the path
// short regardless of parameter count while still being deterministic per
// exact parameter list, so two panels with the same parameters in the same
// order share one channel/backend stream, and two different multi-parameter
// queries don't collide.
function hashParameterList(parameters: string[]): string {
    let hash = 5381;
    const joined = parameters.join('\u0000');
    for (let i = 0; i < joined.length; i++) {
        hash = (hash * 33) ^ joined.charCodeAt(i);
    }
    return (hash >>> 0).toString(36);
}

function parseParameterList(value: string | undefined): string[] {
    return (value ?? '')
        .split(/[\n,]+/)
        .map((parameter) => parameter.trim())
        .filter(Boolean);
}

function getQueryParameters(query: Query): string[] {
    const parameters = (query.parameters ?? []).map((parameter) => parameter.trim()).filter(Boolean);
    if (parameters.length > 0) {
        return parameters;
    }
    return parseParameterList(query.parameter);
}

function isParameterQueryType(type: QueryType | undefined): boolean {
    return (
        type === QueryType.PLOT || type === QueryType.SINGLE || type === QueryType.DISCRETE || type === QueryType.IMAGE
    );
}

function isMultiParameterQueryType(type: QueryType | undefined): boolean {
    return type === QueryType.PLOT || type === QueryType.SINGLE || type === QueryType.DISCRETE;
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

// Scenes unsubscribes hidden tabs; keep their Live channels warm briefly so a returning tab receives current data.
const inactiveStreamRetentionMs = 2 * 60 * 1000;

interface RetainedStream {
    subject: ReplaySubject<DataQueryResponse>;
    source: Subscription;
    subscribers: number;
    idleTimer?: ReturnType<typeof setTimeout>;
}

/**
 * Custom Grafana DataSource for retrieving and streaming data.
 */
export class DataSource extends DataSourceWithBackend<Query, Configuration> {
    bufferMaxLength = DefaultConfiguration.bufferMaxLength;
    dataPointsRounding = DefaultConfiguration.dataPointsRounding;
    debugMode = false;
    private retainedStreams = new Map<string, RetainedStream>();

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

    private getRetainedStream(
        path: string,
        data: Query & { from: number; to: number; points: number },
        action: StreamingFrameAction
    ): Observable<DataQueryResponse> {
        const stream = getGrafanaLiveSrv().getDataStream({
            buffer: { maxLength: this.bufferMaxLength, action },
            addr: {
                scope: LiveChannelScope.DataSource,
                namespace: this.uid,
                stream: this.uid,
                path,
                data,
            },
        });

        return new Observable<DataQueryResponse>((subscriber) => {
            let retained = this.retainedStreams.get(path);
            if (!retained) {
                const subject = new ReplaySubject<DataQueryResponse>(1);
                retained = { subject, source: Subscription.EMPTY, subscribers: 0 };
                this.retainedStreams.set(path, retained);
                const entry = retained;
                entry.source = stream.subscribe({
                    next: (response) => subject.next(response),
                    error: (error) => {
                        this.retainedStreams.delete(path);
                        subject.error(error);
                    },
                    complete: () => {
                        this.retainedStreams.delete(path);
                        subject.complete();
                    },
                });
            }

            if (retained.idleTimer) {
                clearTimeout(retained.idleTimer);
                retained.idleTimer = undefined;
            }
            retained.subscribers++;
            const subscription = retained.subject.subscribe(subscriber);
            return () => {
                subscription.unsubscribe();
                retained.subscribers--;
                if (retained.subscribers === 0 && this.retainedStreams.get(path) === retained) {
                    if (typeof window !== 'undefined' && window.__jaopsBenchmarkNoRetention) {
                        retained.source.unsubscribe();
                        this.retainedStreams.delete(path);
                        return;
                    }
                    retained.idleTimer = setTimeout(() => {
                        retained.source.unsubscribe();
                        this.retainedStreams.delete(path);
                    }, inactiveStreamRetentionMs);
                }
            };
        }).pipe(tap(createBenchmarkFrontendTap()));
    }

    /**
     * Processes the query request and retrieves data streams from Grafana Live.
     * @param request - The data query request from Grafana.
     * @returns An observable emitting the query response.
     */
    query(request: DataQueryRequest<Query>): Observable<DataQueryResponse> {
        const observables = request.targets
            .flatMap((target) => {
                const templateSrv = getTemplateSrv();
                const endpoint = target.asVariable
                    ? templateSrv.replace(target.endpointVariable, request.scopedVars)
                    : templateSrv.replace(target.endpoint, request.scopedVars);
                const command = templateSrv.replace(target.command, request.scopedVars);
                const parameterList = getQueryParameters(target)
                    .flatMap((parameter) => parseParameterList(templateSrv.replace(parameter, request.scopedVars)))
                    .filter(Boolean);

                // A Plot/Single/Discrete query is always sent as one query carrying
                // the whole parameter list (parameters), one parameter or several -
                // there's no separate single-parameter shape on the wire. The backend
                // (RunParameterStream) opens one Live channel/goroutine per query
                // regardless of how many parameters are in that list, and the N=1
                // case is simply the degenerate case of the same code path, not a
                // different one. `parameter` is kept as parameters[0] for backend
                // code that only wants a single representative parameter (query
                // validation, the historical-range fetch for a single parameter).
                const queries =
                    isMultiParameterQueryType(target.type) && parameterList.length > 0
                        ? [
                              {
                                  ...target,
                                  endpoint,
                                  command,
                                  parameter: parameterList[0],
                                  parameters: parameterList,
                              },
                          ]
                        : [
                              {
                                  ...target,
                                  endpoint,
                                  command,
                                  parameter: isParameterQueryType(target.type)
                                      ? templateSrv.replace(target.parameter, request.scopedVars)
                                      : target.parameter,
                              },
                          ];

                return queries.map((query) => {
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
                    if (query.parameters && query.parameters.length > 1) {
                        // A hash, not the joined parameter names: see
                        // hashParameterList's doc comment for why concatenating
                        // every name would silently break wide queries. Hashed on a
                        // sorted copy (order-insensitive) so two panels requesting
                        // the same parameter *set* in a different order still share
                        // one backend stream, the same way Yamcs's own subscription
                        // dedup already works by parameter, not by request order.
                        pathName = `multi-${hashParameterList([...query.parameters].sort())}`;
                    } else if (query.parameter) {
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

                    pathName = templateSrv.replace(pathName, request.scopedVars);

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

                    return this.getRetainedStream(
                        pathParts.join('/'),
                        { ...query, from: fromUnix, to: toUnix, points: roundedMaxDataPoints },
                        action
                    );
                });
            })
            .filter(Boolean) as Array<Observable<DataQueryResponse>>; // Remove undefined values

        return merge(...observables);
    }
}
