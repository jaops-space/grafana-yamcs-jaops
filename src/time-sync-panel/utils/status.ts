export type TimeSyncStatus = 'functional' | 'not-functional' | 'disabled';

export type StatusInfo = {
    label: string;
    details: string;
};

export function getStatusInfo(state: TimeSyncStatus): StatusInfo {
    if (state === 'functional') {
        return {
            label: 'Time sync functional',
            details: 'Dashboard range is auto-aligned to Yamcs current_time.',
        };
    }

    if (state === 'disabled') {
        return {
            label: 'Time sync disabled',
            details: 'Enable panel option "Enable Yamcs time sync" to use it.',
        };
    }

    return {
        label: 'Time sync not functional',
        details: 'Needs current_time data and a relative range (for example now-15m to now).',
    };
}
