import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { QueryField, QueryType, Query, Configuration } from '../types';

export type QueryProps = QueryEditorProps<DataSource, Query, Configuration> & Partial<Query>;

export enum QueryCategory {
    PARAMETER = 'parameter',
    EVENT = 'event',
    IMAGE = 'image',
    COMMANDING = 'commanding',
    DEBUG = 'debug',
}

export const QueryOptions: Array<SelectableValue<QueryType>> = [
    {
        label: 'Graph',
        description: 'Plot a numerical parameter over time and get new updates in realtime.',
        value: QueryType.PLOT,
        category: QueryCategory.PARAMETER,
        additionalFields: true,
        
    },
    {
        label: 'Single Value',
        description: "Get a numerical parameter's latest value in realtime.",
        value: QueryType.SINGLE,
        category: QueryCategory.PARAMETER,
        additionalFields: false,
    },
    {
        label: 'Discrete State',
        description: "Plot a discrete parameter (enum/string)'s states over time and get new updates in realtime.",
        value: QueryType.DISCRETE,
        category: QueryCategory.PARAMETER,
        additionalFields: false,
    },
    {
        label: 'Events',
        description: 'List past and live events.',
        value: QueryType.EVENTS,
        category: QueryCategory.EVENT,
        additionalFields: false,
    },
    {
        label: 'Image',
        description: 'Visualize images. (Not available yet)',
        value: QueryType.IMAGE,
        category: QueryCategory.IMAGE,
        additionalFields: false,
    },
    {
        label: 'Commanding',
        description: 'Send commands to an endpoint.',
        value: QueryType.COMMANDING,
        category: QueryCategory.COMMANDING,
        additionalFields: false,
    },
    {
        label: 'Endpoint Stream Demands',
        description: 'Debug endpoint stream demands.',
        value: QueryType.DEMANDS,
        category: QueryCategory.DEBUG,
        additionalFields: false,
    },
    {
        label: 'Yamcs Subscriptions',
        description: 'Debug Yamcs parameter subscriptions for the endpoint.',
        value: QueryType.SUBSCRIPTIONS,
        category: QueryCategory.DEBUG,
        additionalFields: false,
    }
];

export const FieldsOptions: Array<SelectableValue<QueryField>> = [
    {
        label: 'Maximum',
        value: 'max',
    },
    {
        label: 'Minimum',
        value: 'min',
    },
];
