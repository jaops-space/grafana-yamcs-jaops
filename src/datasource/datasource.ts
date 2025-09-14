import {
    CoreApp,
    DataQueryRequest,
    DataQueryResponse,
    DataSourceInstanceSettings,
    LiveChannelScope,
    StreamingFrameAction,
} from '@grafana/data';

import { DataSourceWithBackend, getGrafanaLiveSrv, getTemplateSrv } from '@grafana/runtime';

import { Observable, merge } from 'rxjs';
import { Configuration, DEFAULT_QUERY as DefaultQuery, Query, QueryType } from './types';

/**
 * Custom Grafana DataSource for retrieving and streaming data.
 */
export class DataSource extends DataSourceWithBackend<Query, Configuration> {
    bufferMaxLength = 10000;
    debugMode = false;

    constructor(instanceSettings: DataSourceInstanceSettings<Configuration>) {
        super(instanceSettings);
        this.bufferMaxLength = instanceSettings.jsonData.bufferMaxLength ?? this.bufferMaxLength;
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
                    return undefined; // Skip invalid queries
                }

                let pathName = 'query';
                if (query.parameter) {
                    pathName = `${query.endpoint}-${query.parameter.replaceAll("/", "")}${query.aggregatePath}`;
                } else if (query.type === QueryType.EVENTS){
                    pathName = 'events'
                } else if (query.type === QueryType.DEMANDS) {
                    pathName = 'demands'
                } else if (query.type === QueryType.SUBSCRIPTIONS) {
                    pathName = 'subscriptions'
                } else if (query.type === QueryType.COMMAND_HISTORY) {
                    pathName = 'commands'
                }   

                let action = StreamingFrameAction.Append;
                if (query.type === QueryType.DEMANDS || query.type === QueryType.SUBSCRIPTIONS || query.type === QueryType.COMMANDING) {
                    action = StreamingFrameAction.Replace;
                }

                const templateSrv = getTemplateSrv();

                pathName = templateSrv.replace(pathName, request.scopedVars);
                query.aggregatePath = templateSrv.replace(query.aggregatePath, request.scopedVars);
                query.parameter = templateSrv.replace(query.parameter, request.scopedVars);
                query.command = templateSrv.replace(query.command, request.scopedVars);

                if (query.asVariable) {
                    query.endpoint = templateSrv.replace(query.endpointVariable, request.scopedVars);
                }

                return getGrafanaLiveSrv().getDataStream({
                    buffer: {
                        maxLength: this.bufferMaxLength,
                        action,
                    },
                    addr: {
                        scope: LiveChannelScope.DataSource,
                        namespace: this.uid,
                        path: 
                        `req/${pathName}-${request.range.from.unix()}-${request.range.to.unix()}-${request.maxDataPoints ?? 1000}-${Math.round(Math.random() * 9999)}`,
                        data: {
                            ...query,
                            from: request.range.from.unix(),
                            to: request.range.to.unix(),
                            realtime: request.range.raw.to === 'now',
                            points: request.maxDataPoints ?? 1000,
                        },
                    },
                });
            })
            .filter(Boolean) as Array<Observable<DataQueryResponse>>; // Remove undefined values

        return merge(...observables);
    }
}
