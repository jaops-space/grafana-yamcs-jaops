import { QueryType } from '../types';

export const queryHelp: Record<QueryType, { tooltip: string; recommendedPanel: string }> = {
    [QueryType.PLOT]: {
        tooltip: 'Historical range plus live stream for numerical telemetry.',
        recommendedPanel: 'Time series',
    },
    [QueryType.SINGLE]: {
        tooltip: 'Latest Yamcs value only. No historical query.',
        recommendedPanel: 'Stat',
    },
    [QueryType.DISCRETE]: {
        tooltip: 'State ranges plus live updates for strings, booleans, enums, and integers.',
        recommendedPanel: 'State timeline',
    },
    [QueryType.TIME]: {
        tooltip: 'Current Yamcs processor time and replay speed.',
        recommendedPanel: 'Yamcs Time Sync',
    },
    [QueryType.EVENTS]: {
        tooltip: 'Past events over the selected range plus live Yamcs events.',
        recommendedPanel: 'Logs',
    },
    [QueryType.IMAGE]: {
        tooltip: 'Latest image value from a Yamcs parameter.',
        recommendedPanel: 'Telemetric Image Panel',
    },
    [QueryType.COMMANDING]: {
        tooltip: 'Command metadata for rendering configurable command controls.',
        recommendedPanel: 'Commanding Panel',
    },
    [QueryType.COMMAND_HISTORY]: {
        tooltip: 'Command entries, arguments, and acknowledgements.',
        recommendedPanel: 'Command History Panel',
    },
    [QueryType.ALARMS]: {
        tooltip: 'Active Yamcs alarms plus live alarm updates.',
        recommendedPanel: 'Alarms Panel',
    },
    [QueryType.LINKS]: {
        tooltip: 'Yamcs data links and live link updates.',
        recommendedPanel: 'Links Panel',
    },
    [QueryType.DEMANDS]: {
        tooltip: 'Debug view of Grafana stream paths demanding endpoint parameters.',
        recommendedPanel: 'Table',
    },
    [QueryType.SUBSCRIPTIONS]: {
        tooltip: 'Debug view of active Yamcs parameter subscriptions.',
        recommendedPanel: 'Table',
    },
};
