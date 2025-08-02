import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

/**
 * Interface representing a data query.
 */
export interface Query extends DataQuery {
    type?: QueryType;
    endpoint?: string;
    parameter: string;
    command: string;
    aggregatePath: string;
    fields: QueryField[];
    asVariable: boolean;
    customVariableString: boolean;
    endpointVariable: string;
}

/**
 * Enumeration for different query types.
 */
export enum QueryType {
    PLOT = 'plot',
    SINGLE = 'single',
    DISCRETE = 'discrete',
    EVENTS = 'events',
    IMAGE = 'image',

    DEMANDS = 'demands',
    SUBSCRIPTIONS = 'subscriptions',

    COMMANDING = 'commanding',
    COMMAND_HISTORY = 'command-history'
}

/**
 * Allowed fields for query operations.
 */
export type QueryField = 'max' | 'min';

/**
 * Default values for a query.
 */
export const DEFAULT_QUERY: Partial<Query> = {
    type: undefined,
    endpoint: undefined,
    aggregatePath: "",
};

/**
 * Interface defining configuration settings for the data source.
 */
export interface Configuration extends DataSourceJsonData {
    /**
     * Mapping of host configurations.
     */
    hosts: Record<
        string,
        {
            name?: string;
            path: string;
            tlsEnabled: boolean;
            tlsInsecure?: boolean;
            authEnabled: boolean;
        }
    >;

    /**
     * Mapping of data sources.
     */
    endpoints: Record<
        string,
        {
            name: string;
            description?: string;
            host: string;
            instance: string;
            processor?: string;
        }
    >;

    bufferMaxLength: number;
    debugMode: boolean;
}

export const DefaultConfiguration = {
    hosts: {},
    endpoints: {},
    bufferMaxLength: 5000,
    debugMode: false,
} satisfies Configuration;

/**
 * Utility type to allow optional values (null or undefined).
 */
export type Optional<T> = T | null | undefined;

/**
 * Source fetching types
 */
export enum EndpointStatus {
    NoInfo,
    Error,
    Online,
    Offline,
}

export interface FetchEndpointsStatusResponse {
    sources: Record<string, EndpointInfo>;
}

export interface EndpointInfo {
    online: boolean;
    name: string;
    description: string;
}

export type ValueOf<T> = {
    [K in keyof T]: T[K];
}[keyof T];

export type Endpoints = Configuration['endpoints'];
export type Hosts = Configuration['hosts'];

export type Endpoint = ValueOf<Endpoints>;
export type Host = ValueOf<Hosts>;
export type IndexedEndpoint = ValueOf<Configuration['endpoints']> & {index: string};
