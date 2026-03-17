import { dateTime, PanelProps } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { Button, Icon, InteractiveTable, Modal, Combobox, Stack, Text, TextArea, Tooltip, useTheme2 } from '@grafana/ui';
import { AlarmsOptions } from 'alarms-panel/module';
import React, { useCallback, useMemo, useState } from 'react';


interface AlarmEntry {
    id: string;
    name: string;
    triggerTime: string;
    updateTime?: string;
    severity: string;
    type: string;
    violations: number;
    acknowledged: boolean;
    acknowledgedBy?: string;
    acknowledgeTime?: string;
        acknowledgeComment?: string;
    processOK: boolean;
    triggered: boolean;
    latching: boolean;
    shelved: boolean;
    shelvedBy?: string;
    shelveTime?: string;
    shelveExpiration?: string;
    shelveComment?: string;
    cleared?: boolean;
    clearedBy?: string;
    clearTime?: string;
    clearComment?: string;
    currentValue?: string;
    triggerValue?: string;
    mostSevereValue?: string;
    triggerValueDetail?: any;
    mostSevereValueDetail?: any;
    currentValueDetail?: any;
    parameterInfo?: any;
    notificationType: string;
    seqNum: number;
}


function deepCombine<T = any>(obj1: T, obj2: T): T {
    if (Array.isArray(obj1) && Array.isArray(obj2)) {
        return obj1.concat(obj2) as T;
    }

    if (isObject(obj1) && isObject(obj2)) {
        const result: any = {};
        const keys = new Set([...Object.keys(obj1), ...Object.keys(obj2)]);
        for (const key of keys) {
            const val1 = (obj1 as any)[key];
            const val2 = (obj2 as any)[key];
            if (key in obj1 && key in obj2) {
                result[key] = deepCombine(val1, val2);
            } else if (key in obj1) {
                result[key] = val1;
            } else {
                result[key] = val2;
            }
        }
        return result;
    }

    return obj2;
}

function isObject(val: any): val is Record<string, any> {
    return val !== null && typeof val === 'object' && !Array.isArray(val);
}

function formatTime(iso: string): string {
    return dateTime(iso).format('YYYY-MM-DD HH:mm:ss');
}

const dedupeById = (entries: AlarmEntry[]): AlarmEntry[] => {
    const map = new Map<string, number>();
    const result: AlarmEntry[] = [];

    for (const entry of entries) {
        const id = entry.id;
        if (map.has(id)) {
            const index = map.get(id)!;
            result[index] = deepCombine(result[index], entry);
        } else {
            result.push({ ...entry });
            map.set(id, result.length - 1);
        }
    }

    return result;
};

// Severity colors and icons - 5 distinct levels
const severityConfig: Record<string, { color: string; icon: any }> = {
    'WATCH': { color: '#3399FF', icon: 'record-audio' },      // Light blue
    'WARNING': { color: '#FF9933', icon: 'record-audio' },    // Orange
    'DISTRESS': { color: '#FF6600', icon: 'record-audio' },   // Dark orange
    'CRITICAL': { color: '#FF3333', icon: 'record-audio' },   // Red
    'SEVERE': { color: '#CC0000', icon: 'record-audio' },     // Dark red
};

// Format precise duration (e.g., "56 minutes ago", "1h 10 minutes ago")
const formatPreciseDuration = (timestamp: string): string => {
    const now = Date.now();
    const then = new Date(timestamp).getTime();
    const diffMs = now - then;

    if (diffMs < 0) {
        return 'just now';
    }

    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) {
        const remainingHours = hours % 24;
        if (remainingHours > 0) {
            return `${days}d ${remainingHours}h ago`;
        }
        return `${days} ${days === 1 ? 'day' : 'days'} ago`;
    } else if (hours > 0) {
        const remainingMinutes = minutes % 60;
        if (remainingMinutes > 0) {
            return `${hours}h ${remainingMinutes} minutes ago`;
        }
        return `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
    } else if (minutes > 0) {
        return `${minutes} ${minutes === 1 ? 'minute' : 'minutes'} ago`;
    } else {
        return `${seconds} ${seconds === 1 ? 'second' : 'seconds'} ago`;
    }
};


const AlarmsPanel: React.FC<PanelProps<AlarmsOptions>> = ({ data, options }) => {
    const theme = useTheme2();
    const [modalOpen, setModalOpen] = useState(false);
    const [selectedAlarm, setSelectedAlarm] = useState<AlarmEntry | null>(null);
    const [actionType, setActionType] = useState<'acknowledge' | 'clear' | 'shelve' | 'unshelve'>('acknowledge');
    const [comment, setComment] = useState('');
    const [loading, setLoading] = useState(false);
    const [shelveDuration, setShelveDuration] = useState<number>(7200000); // Default 2 hours in milliseconds
    const [expandedParamData, setExpandedParamData] = useState<Record<string, boolean>>({});
    const [expandedFields, setExpandedFields] = useState<Record<string, Record<string, boolean>>>({});

    // Extract endpoint from query target
    const endpoint = useMemo(() => {
        const request = data.request;
        if (request?.targets?.[0]) {
            const target = request.targets[0] as any;
            return target.endpoint;
        }
        return null;
    }, [data]);

    // Extract datasource from query target
    const datasource = useMemo(() => {
        const request = data.request;
        if (request?.targets?.[0]) {
            const ds = request.targets[0].datasource as DataSourceWithBackend;
            if (ds) {
                Object.setPrototypeOf(ds, DataSourceWithBackend.prototype);
                return ds;
            }
        }
        return null;
    }, [data]);

    const deduped = useMemo(() => {
        const frame = data.series[0];
        if (!frame || !frame.fields.length) { return []; }

        const rawField = frame.fields.find((f) => f.name === 'alarms');
        if (!rawField) { return []; }

        const raw = rawField.values as any[];
        const parsed: AlarmEntry[] = raw.filter(Boolean);
        const result = dedupeById(parsed);

        // Filter out cleared alarms - they should not appear in the active alarms list
        const activeAlarms = result.filter(alarm => !alarm.cleared);

        // Sort consistently by ID to maintain stable order
        activeAlarms.sort((a, b) => {
            // First compare by trigger time (newest first)
            const timeA = new Date(a.triggerTime).getTime();
            const timeB = new Date(b.triggerTime).getTime();
            if (timeA !== timeB) {
                return timeB - timeA;
            }
            // Then by name
            if (a.name !== b.name) {
                return a.name.localeCompare(b.name);
            }
            // Finally by sequence number
            return a.seqNum - b.seqNum;
        });

        return activeAlarms;
    }, [data]);

    // Extract GlobalAlarmStatus from frame metadata
    const globalAlarmStatus = useMemo(() => {
        const frame = data.series[0];
        if (frame?.meta?.custom?.globalAlarmStatus) {
            return frame.meta.custom.globalAlarmStatus;
        }
        return null;
    }, [data]);

    // Compute alarm statistics from the actual alarms data
    const alarmStats = useMemo(() => {
        if (!deduped.length) {
            return {
                unacknowledgedCount: 0,
                acknowledgedCount: 0,
                shelvedCount: 0,
                unacknowledgedSeverity: null,
                acknowledgedSeverity: null,
                shelvedSeverity: null,
            };
        }

        const severityOrder = ['WATCH', 'WARNING', 'DISTRESS', 'CRITICAL', 'SEVERE'];
        const getHighestSeverity = (alarms: AlarmEntry[]) => {
            if (!alarms.length) {
                return null;
            }
            let highest = alarms[0].severity;
            for (const alarm of alarms) {
                if (severityOrder.indexOf(alarm.severity) > severityOrder.indexOf(highest)) {
                    highest = alarm.severity;
                }
            }
            return highest;
        };

        const unacknowledged = deduped.filter(a => !a.acknowledged && !a.shelved && !a.cleared);
        const acknowledged = deduped.filter(a => a.acknowledged && !a.shelved && !a.cleared);
        const shelved = deduped.filter(a => a.shelved && !a.cleared);

        return {
            unacknowledgedCount: unacknowledged.length,
            acknowledgedCount: acknowledged.length,
            shelvedCount: shelved.length,
            unacknowledgedSeverity: getHighestSeverity(unacknowledged),
            acknowledgedSeverity: getHighestSeverity(acknowledged),
            shelvedSeverity: getHighestSeverity(shelved),
        };
    }, [deduped]);

    // Use computed stats if backend data is not available
    const displayStatus = globalAlarmStatus || alarmStats;

    const handleAction = useCallback(async (alarm: AlarmEntry, action: 'acknowledge' | 'clear' | 'shelve' | 'unshelve') => {
        setSelectedAlarm(alarm);
        setActionType(action);
        setComment('');
        setModalOpen(true);
    }, []);

    const executeAction = useCallback(async () => {
        if (!selectedAlarm || !endpoint || !datasource) { return; }

        setLoading(true);
        try {
            const actionEndpoint = actionType === 'acknowledge' ? 'acknowledge' :
                                   actionType === 'clear' ? 'clear' :
                                   actionType === 'shelve' ? 'shelve' : 'unshelve';

            await datasource.postResource(
                `endpoint/${endpoint}/alarm/${actionEndpoint}`,
                {
                    name: selectedAlarm.name,
                    seqNum: selectedAlarm.seqNum,
                    comment: comment,
                    ...(actionType === 'shelve' ? { shelveDuration: shelveDuration } : {}),
                }
            );
            setModalOpen(false);
        } catch (error) {
            console.error(`Failed to ${actionType} alarm:`, error);
        } finally {
            setLoading(false);
        }
    }, [selectedAlarm, endpoint, datasource, actionType, comment, shelveDuration]);

    // Build columns
    // Columns: State, Severity, Alarm time, Alarm name, Alarm type, Trigger value, Most severe value, Live value, Actions
    const columns = useMemo(() => [
        {
            id: 'state',
            header: 'State',
            cell: (info: any) => {
                const alarm = info.row.original.row;

                // Determine concise icon + color for the alarm state
                let iconName = 'question-circle';
                let iconColor = theme.colors.text.disabled; // default grey
                let tooltipText = 'Unknown';

                if (alarm.shelved) {
                    iconName = 'clock-nine';
                    iconColor = theme.colors.text.disabled; // grey
                    tooltipText = 'Shelved';
                } else if (alarm.cleared) {
                    iconName = 'check';
                    iconColor = theme.colors.text.disabled; // show cleared as grey
                    tooltipText = 'Cleared';
                } else if (!alarm.triggered) {
                    // Not triggered but not cleared -> treat as acknowledged active
                    iconName = 'check';
                    iconColor = theme.colors.info.text; // blue for acknowledged
                    tooltipText = 'Active, acknowledged';
                } else if (alarm.acknowledged) {
                    iconName = 'check';
                    iconColor = theme.colors.info.text; // blue for acknowledged
                    tooltipText = 'Active, acknowledged';
                } else {
                    // active and unacknowledged
                    iconName = 'exclamation-triangle';
                    iconColor = theme.colors.error.text; // red for unacknowledged
                    tooltipText = 'Active, unacknowledged';
                }

                return (
                    <Tooltip content={tooltipText}>
                        <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 28 }}>
                            <Icon name={iconName as any} color={iconColor} />
                        </span>
                    </Tooltip>
                );
            },
        },
        {
            id: 'severity',
            header: 'Severity',
            cell: (info: any) => {
                const severity = info.row.original.row.severity;
                const config = severityConfig[severity] || { color: 'gray', icon: 'question-circle' };
                return (
                    <Stack direction="row" gap={1} alignItems="center">
                        <Icon name={config.icon} color={config.color} />
                        <Text color={config.color as any}>{severity}</Text>
                    </Stack>
                );
            },
        },
        {
            id: 'triggerTime',
            header: 'Alarm time',
            accessorKey: 'triggerTime',
            cell: (info: any) => {
                const triggerTime = info.row.original.row.triggerTime;
                const timestamp = formatTime(triggerTime);
                return (
                    <Tooltip content={timestamp}>
                        <span>{formatPreciseDuration(triggerTime)}</span>
                    </Tooltip>
                );
            },
        },
        {
            id: 'name',
            header: 'Alarm name',
            accessorKey: 'name',
            cell: (info: any) => {
                const fullPath = info.row.original.row.name;
                // Extract just the parameter name (last part after the last '/')
                const paramName = fullPath.split('/').pop() || fullPath;
                return (
                    <Tooltip content={fullPath}>
                        <span>{paramName}</span>
                    </Tooltip>
                );
            },
        },
        {
            id: 'type',
            header: 'Alarm type',
            accessorKey: 'type',
            cell: (info: any) => info.row.original.row.type || '-',
        },
        {
            id: 'triggerValue',
            header: 'Trigger value',
            accessorKey: 'triggerValue',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                const val = alarm.triggerValue;
                if (!val) { return '-'; }
                if (alarm.type === 'EVENT') {
                    // "SEVERITY: message" -> show only "SEVERITY"
                    const colonIdx = val.indexOf(': ');
                    return colonIdx >= 0 ? val.substring(0, colonIdx) : val;
                }
                return val;
            },
        },

        {
            id: 'currentValue',
            header: 'Live value',
            accessorKey: 'currentValue',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                const val = alarm.currentValue;
                if (!val) { return '-'; }
                if (alarm.type === 'EVENT') {
                    const colonIdx = val.indexOf(': ');
                    return colonIdx >= 0 ? val.substring(0, colonIdx) : val;
                }
                return val;
            },
        },
        {
            id: 'triggerTimestamp',
            header: 'Trigger Timestamp',
            accessorKey: 'triggerTime',
            cell: (info: any) => {
                const triggerTime = info.row.original.row.triggerTime;
                return triggerTime ? formatTime(triggerTime) : '-';
            },
        },
        {
            id: 'violations',
            header: 'Violations',
            accessorKey: 'violations',
            cell: (info: any) => {
                const v = info.row.original.row.violations;
                return v !== undefined && v !== null ? v : '-';
            },
        },
        {
            id: 'acknowledged',
            header: 'Ack',
            accessorKey: 'acknowledged',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                return alarm.acknowledged ? (
                    <Tooltip content={alarm.acknowledgedBy ? `By: ${alarm.acknowledgedBy}` : 'Acknowledged'}>
                        <Icon name="check" color={theme.colors.info.text} />
                    </Tooltip>
                ) : '-';
            },
        },
        {
            id: 'actions',
            header: 'Actions',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                return (
                    <Stack direction="row" gap={0.5}>
                        {!alarm.acknowledged && (
                            <Button
                                size="sm"
                                variant="secondary"
                                icon="check"
                                tooltip="Acknowledge alarm"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    handleAction(alarm, 'acknowledge');
                                }}
                            />
                        )}
                        {alarm.acknowledged && !alarm.cleared && (
                            <Button
                                size="sm"
                                variant="secondary"
                                icon="times"
                                tooltip="Clear alarm"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    handleAction(alarm, 'clear');
                                }}
                            />
                        )}
                        {!alarm.shelved ? (
                            <Button
                                size="sm"
                                variant="secondary"
                                icon="pause-circle" // Updated icon for shelve
                                tooltip="Shelve alarm"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    handleAction(alarm, 'shelve');
                                }}
                            />
                        ) : (
                            <Button
                                size="sm"
                                variant="secondary"
                                icon="play" // use allowed icon name
                                tooltip="Unshelve alarm"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    handleAction(alarm, 'unshelve');
                                }}
                            />
                        )}
                    </Stack>
                );
            },
        },
    ], [severityConfig, formatTime, formatPreciseDuration, handleAction, theme]);

    // Filter columns based on user's visible fields selection
    const visibleColumns = useMemo(() => {
        // If no visible fields are configured, use all columns
        let fieldsToShow = options.visibleFields && options.visibleFields.length > 0
            ? [...options.visibleFields]
            : ['state', 'severity', 'triggerTime', 'name', 'type', 'triggerValue', 'currentValue', 'actions'];

        // Ensure 'state' is always present (for panels saved before the State column was added).
        if (!fieldsToShow.includes('state')) {
            const sevIdx = fieldsToShow.indexOf('severity');
            if (sevIdx >= 0) {
                fieldsToShow.splice(sevIdx, 0, 'state');
            } else {
                fieldsToShow.unshift('state');
            }
        }

        // Return columns in the order they appear in the visibleFields array
        return fieldsToShow
            .map(fieldId => columns.find(col => col.id === fieldId))
            .filter((col): col is NonNullable<typeof col> => !!col);
    }, [columns, options.visibleFields]);

    // Render expanded row with details
    function renderSubComponent({ row }: { row: any }) {
        const alarm: AlarmEntry = row;
        const isEventAlarm = alarm.type === 'EVENT';
        const isParamDataExpanded = expandedParamData[alarm.id] || false;
        const alarmExpandedFields = expandedFields[alarm.id] || {};

        const toggleParamData = () => {
            setExpandedParamData(prev => ({
                ...prev,
                [alarm.id]: !prev[alarm.id]
            }));
        };

        const toggleField = (fieldName: string) => {
            setExpandedFields(prev => ({
                ...prev,
                [alarm.id]: {
                    ...(prev[alarm.id] || {}),
                    [fieldName]: !(prev[alarm.id]?.[fieldName] || false)
                }
            }));
        };

        return (
            <div style={{ padding: theme.spacing(1), background: theme.colors.background.secondary }}>
                <Stack direction="column" gap={1}>
                    <Text><strong>{isEventAlarm ? 'Event Source:' : 'Full Parameter Path:'}</strong> {alarm.name}</Text>
                    <Text><strong>Alarm Type:</strong> {alarm.type}</Text>
                    <Text><strong>Trigger Timestamp:</strong> {formatTime(alarm.triggerTime)}</Text>
                    <Text><strong>Alarm time:</strong> {formatPreciseDuration(alarm.triggerTime)}</Text>
                    {alarm.updateTime && <Text><strong>Last Update:</strong> {formatTime(alarm.updateTime)}</Text>}
                    <Text><strong>{isEventAlarm ? 'Trigger Event:' : 'Trigger Value:'}</strong> {alarm.triggerValue || '-'}</Text>
                    {!isEventAlarm && alarm.mostSevereValue && (
                        <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                            <Text><strong>Most Severe Value:</strong> {alarm.mostSevereValue}</Text>
                            <Tooltip content="The parameter value at the moment this alarm reached its highest severity level. This is not necessarily the highest numeric value seen, it only updates when the alarm severity increases (e.g. WARNING → CRITICAL).">
                                <span style={{ cursor: 'help', lineHeight: 1 }}>
                                    <Icon name="info-circle" size="sm" />
                                </span>
                            </Tooltip>
                        </span>
                    )}
                    <Text><strong>{isEventAlarm ? 'Current Event:' : 'Live Value:'}</strong> {alarm.currentValue || '-'}</Text>
                    <Tooltip content="Number of rule violations (this is what Yamcs web shows)">
                        <Text><strong>Violations:</strong> {alarm.violations}</Text>
                    </Tooltip>

                    {!isEventAlarm && <Text><strong>Latching:</strong> {alarm.latching ? 'Yes' : 'No'}</Text>}

                    {/* ParameterAlarmData Section - Collapsible */}
                    {!isEventAlarm && (alarm.triggerValueDetail || alarm.mostSevereValueDetail || alarm.currentValueDetail || alarm.parameterInfo) && (
                        <div style={{ marginTop: theme.spacing(1) }}>
                            <div
                                onClick={toggleParamData}
                                style={{
                                    cursor: 'pointer',
                                    display: 'flex',
                                    alignItems: 'center',
                                    padding: theme.spacing(0.5),
                                    background: theme.colors.background.primary,
                                    borderRadius: theme.shape.radius.default
                                }}
                            >
                                <Icon name={isParamDataExpanded ? 'angle-down' : 'angle-right'} />
                                <Text><strong>ParameterAlarmData</strong></Text>
                            </div>

                            {isParamDataExpanded && (
                                <div style={{ marginLeft: theme.spacing(2), marginTop: theme.spacing(1) }}>
                                    <Stack direction="column" gap={0.5}>
                                        {/* Trigger Value - Collapsible */}
                                        {alarm.triggerValueDetail && (
                                            <div>
                                                <div
                                                    onClick={() => toggleField('triggerValue')}
                                                    style={{
                                                        cursor: 'pointer',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        padding: theme.spacing(0.25)
                                                    }}
                                                >
                                                    <Icon name={alarmExpandedFields['triggerValue'] ? 'angle-down' : 'angle-right'} size="sm" />
                                                    <Text><strong>triggerValue: {alarm.triggerValue || JSON.stringify(alarm.triggerValueDetail.engValue)}</strong></Text>
                                                </div>
                                                {alarmExpandedFields['triggerValue'] && (
                                                    <div style={{ marginLeft: theme.spacing(3) }}>
                                                        <Stack direction="column" gap={0.5}>
                                                            {alarm.triggerValueDetail.engValue !== undefined && (
                                                                <Text>engValue: {JSON.stringify(alarm.triggerValueDetail.engValue)}</Text>
                                                            )}
                                                            {alarm.triggerValueDetail.rawValue !== undefined && (
                                                                <Text>rawValue: {JSON.stringify(alarm.triggerValueDetail.rawValue)}</Text>
                                                            )}
                                                            {alarm.triggerValueDetail.acquisitionTime && (
                                                                <Text>acquisitionTime: {alarm.triggerValueDetail.acquisitionTime}</Text>
                                                            )}
                                                            {alarm.triggerValueDetail.generationTime && (
                                                                <Text>generationTime: {alarm.triggerValueDetail.generationTime}</Text>
                                                            )}
                                                            {alarm.triggerValueDetail.monitoringResult && (
                                                                <Text>monitoringResult: {alarm.triggerValueDetail.monitoringResult}</Text>
                                                            )}
                                                            {alarm.triggerValueDetail.rangeCondition && (
                                                                <Text>rangeCondition: {alarm.triggerValueDetail.rangeCondition}</Text>
                                                            )}
                                                        </Stack>
                                                    </div>
                                                )}
                                            </div>
                                        )}

                                        {/* Most Severe Value - Collapsible */}
                                        {alarm.mostSevereValueDetail && (
                                            <div style={{ marginTop: theme.spacing(0.5) }}>
                                                <div
                                                    onClick={() => toggleField('mostSevereValue')}
                                                    style={{
                                                        cursor: 'pointer',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        padding: theme.spacing(0.25)
                                                    }}
                                                >
                                                    <Icon name={alarmExpandedFields['mostSevereValue'] ? 'angle-down' : 'angle-right'} size="sm" />
                                                    <Text><strong>mostSevereValue: {alarm.mostSevereValue || JSON.stringify(alarm.mostSevereValueDetail.engValue)}</strong></Text>
                                                </div>
                                                {alarmExpandedFields['mostSevereValue'] && (
                                                    <div style={{ marginLeft: theme.spacing(3) }}>
                                                        <Stack direction="column" gap={0.5}>
                                                            {alarm.mostSevereValueDetail.engValue !== undefined && (
                                                                <Text>engValue: {JSON.stringify(alarm.mostSevereValueDetail.engValue)}</Text>
                                                            )}
                                                            {alarm.mostSevereValueDetail.rawValue !== undefined && (
                                                                <Text>rawValue: {JSON.stringify(alarm.mostSevereValueDetail.rawValue)}</Text>
                                                            )}
                                                            {alarm.mostSevereValueDetail.acquisitionTime && (
                                                                <Text>acquisitionTime: {alarm.mostSevereValueDetail.acquisitionTime}</Text>
                                                            )}
                                                            {alarm.mostSevereValueDetail.generationTime && (
                                                                <Text>generationTime: {alarm.mostSevereValueDetail.generationTime}</Text>
                                                            )}
                                                            {alarm.mostSevereValueDetail.monitoringResult && (
                                                                <Text>monitoringResult: {alarm.mostSevereValueDetail.monitoringResult}</Text>
                                                            )}
                                                            {alarm.mostSevereValueDetail.rangeCondition && (
                                                                <Text>rangeCondition: {alarm.mostSevereValueDetail.rangeCondition}</Text>
                                                            )}
                                                        </Stack>
                                                    </div>
                                                )}
                                            </div>
                                        )}

                                        {/* Current Value - Collapsible */}
                                        {alarm.currentValueDetail && (
                                            <div style={{ marginTop: theme.spacing(0.5) }}>
                                                <div
                                                    onClick={() => toggleField('currentValue')}
                                                    style={{
                                                        cursor: 'pointer',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        padding: theme.spacing(0.25)
                                                    }}
                                                >
                                                    <Icon name={alarmExpandedFields['currentValue'] ? 'angle-down' : 'angle-right'} size="sm" />
                                                    <Text><strong>currentValue: {alarm.currentValue || JSON.stringify(alarm.currentValueDetail.engValue)}</strong></Text>
                                                </div>
                                                {alarmExpandedFields['currentValue'] && (
                                                    <div style={{ marginLeft: theme.spacing(3) }}>
                                                        <Stack direction="column" gap={0.5}>
                                                            {alarm.currentValueDetail.engValue !== undefined && (
                                                                <Text>engValue: {JSON.stringify(alarm.currentValueDetail.engValue)}</Text>
                                                            )}
                                                            {alarm.currentValueDetail.rawValue !== undefined && (
                                                                <Text>rawValue: {JSON.stringify(alarm.currentValueDetail.rawValue)}</Text>
                                                            )}
                                                            {alarm.currentValueDetail.acquisitionTime && (
                                                                <Text>acquisitionTime: {alarm.currentValueDetail.acquisitionTime}</Text>
                                                            )}
                                                            {alarm.currentValueDetail.generationTime && (
                                                                <Text>generationTime: {alarm.currentValueDetail.generationTime}</Text>
                                                            )}
                                                            {alarm.currentValueDetail.monitoringResult && (
                                                                <Text>monitoringResult: {alarm.currentValueDetail.monitoringResult}</Text>
                                                            )}
                                                            {alarm.currentValueDetail.rangeCondition && (
                                                                <Text>rangeCondition: {alarm.currentValueDetail.rangeCondition}</Text>
                                                            )}
                                                        </Stack>
                                                    </div>
                                                )}
                                            </div>
                                        )}

                                        {/* Parameter Info - Collapsible */}
                                        {alarm.parameterInfo && (
                                            <div style={{ marginTop: theme.spacing(0.5) }}>
                                                <div
                                                    onClick={() => toggleField('parameter')}
                                                    style={{
                                                        cursor: 'pointer',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        padding: theme.spacing(0.25)
                                                    }}
                                                >
                                                    <Icon name={alarmExpandedFields['parameter'] ? 'angle-down' : 'angle-right'} size="sm" />
                                                    <Text><strong>parameter: {alarm.parameterInfo.qualifiedName || alarm.name}</strong></Text>
                                                </div>
                                                {alarmExpandedFields['parameter'] && (
                                                    <div style={{ marginLeft: theme.spacing(3) }}>
                                                        <Stack direction="column" gap={0.5}>
                                                            {alarm.parameterInfo.qualifiedName !== undefined && alarm.parameterInfo.qualifiedName !== null && alarm.parameterInfo.qualifiedName !== '' && (
                                                                <Text>qualifiedName: {alarm.parameterInfo.qualifiedName}</Text>
                                                            )}
                                                            {alarm.parameterInfo.dataSource !== undefined && alarm.parameterInfo.dataSource !== null && alarm.parameterInfo.dataSource !== '' && (
                                                                <Text>
                                                                    <strong>dataSource:</strong> {alarm.parameterInfo.dataSource}
                                                                    {(() => {
                                                                      const name = getDatasourceDisplayName(alarm.parameterInfo.dataSource);
                                                                      return name ? ` (${name})` : '';
                                                                    })()}
                                                                </Text>
                                                            )}
                                                            {alarm.parameterInfo.shortDescription !== undefined && alarm.parameterInfo.shortDescription !== null && alarm.parameterInfo.shortDescription !== '' && (
                                                                <Text>shortDescription: {alarm.parameterInfo.shortDescription}</Text>
                                                            )}
                                                            {alarm.parameterInfo.longDescription !== undefined && alarm.parameterInfo.longDescription !== null && alarm.parameterInfo.longDescription !== '' && (
                                                                <Text>longDescription: {alarm.parameterInfo.longDescription}</Text>
                                                            )}
                                                        </Stack>
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                    </Stack>
                                </div>
                            )}
                        </div>
                    )}

                    {alarm.acknowledged && (
                        <>
                            <Text><strong>Acknowledged By:</strong> {alarm.acknowledgedBy}</Text>
                            {alarm.acknowledgeTime && <Text><strong>Acknowledge Time:</strong> {formatTime(alarm.acknowledgeTime)}</Text>}
                            {alarm.acknowledgeComment && <Text><strong>Acknowledge Comment:</strong> {alarm.acknowledgeComment}</Text>}
                        </>
                    )}
                    {alarm.cleared && (
                        <>
                            <Text><strong>Cleared By:</strong> {alarm.clearedBy}</Text>
                            {alarm.clearTime && <Text><strong>Clear Time:</strong> {formatTime(alarm.clearTime)}</Text>}
                            {alarm.clearComment && <Text><strong>Clear Comment:</strong> {alarm.clearComment}</Text>}
                        </>
                    )}
                    {alarm.shelved && (
                        <>
                            <Text><strong>Shelved:</strong> Yes</Text>
                            <Text><strong>Shelved By:</strong> {alarm.shelvedBy || '-'}</Text>
                            {alarm.shelveTime && <Text><strong>Shelve Time:</strong> {formatTime(alarm.shelveTime)}</Text>}
                            {alarm.shelveExpiration && <Text><strong>Shelve Until:</strong> {formatTime(alarm.shelveExpiration)}</Text>}
                            {alarm.shelveComment && <Text><strong>Shelve Comment:</strong> {alarm.shelveComment}</Text>}
                        </>
                    )}
                </Stack>
            </div>
        );
    }

    // Pagination state to avoid InteractiveTable resetting to page 1 on data updates
    const [currentPage, setCurrentPage] = useState<number>(1);

    const pageSize = Math.max(1, options.pageSize || 10);
    const totalPages = Math.max(1, Math.ceil(deduped.length / pageSize));

    // Keep current page within bounds when data changes
    React.useEffect(() => {
        if (currentPage > totalPages) {
            setCurrentPage(totalPages);
        }
        // Don't reset to 1 on data updates; preserve the user's current page
    }, [totalPages]);

    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            width: '100%',
            overflow: 'hidden',
        }}>
            {/* Global Alarm Status Bar - Always show all categories */}
            <div style={{ padding: theme.spacing(1), background: theme.colors.background.canvas, borderBottom: `1px solid ${theme.colors.border.weak}` }}>
                <Stack direction="row" gap={2} alignItems="center" justifyContent="space-around">
                    {/* Unacknowledged Alarms - Always displayed */}
                    <Stack direction="row" gap={1} alignItems="center">
                        <Icon
                            name="exclamation-triangle"
                            size="lg"
                            style={{ color: displayStatus.unacknowledgedCount > 0 ? theme.colors.error.text : theme.colors.text.disabled }}
                        />
                        <Stack direction="column" gap={0}>
                            <Text weight="bold" color={displayStatus.unacknowledgedCount > 0 ? "error" : "secondary"}>
                                {displayStatus.unacknowledgedCount}
                            </Text>
                            <Text variant="bodySmall" color="secondary">
                                Unacknowledged
                            </Text>
                        </Stack>
                        {displayStatus.unacknowledgedSeverity && displayStatus.unacknowledgedSeverity !== 'UNRECOGNIZED' && (
                            <Text variant="bodySmall" color="secondary">
                                ({displayStatus.unacknowledgedSeverity})
                            </Text>
                        )}
                    </Stack>

                    {/* Acknowledged Alarms - Always displayed */}
                    <Stack direction="row" gap={1} alignItems="center">
                        <Icon
                            name="check"
                            size="lg"
                            style={{ color: displayStatus.acknowledgedCount > 0 ? theme.colors.info.text : theme.colors.text.disabled }}
                        />
                        <Stack direction="column" gap={0}>
                            <Text weight="bold" color={displayStatus.acknowledgedCount > 0 ? "info" : "secondary"}>
                                {displayStatus.acknowledgedCount}
                            </Text>
                            <Text variant="bodySmall" color="secondary">
                                Acknowledged
                            </Text>
                        </Stack>
                        {displayStatus.acknowledgedSeverity && displayStatus.acknowledgedSeverity !== 'UNRECOGNIZED' && (
                            <Text variant="bodySmall" color="secondary">
                                ({displayStatus.acknowledgedSeverity})
                            </Text>
                        )}
                    </Stack>

                    {/* Shelved Alarms - Always displayed */}
                    <Stack direction="row" gap={1} alignItems="center">
                        <Icon
                            name="clock-nine"
                            size="lg"
                            style={{ color: theme.colors.text.disabled }}
                        />
                        <Stack direction="column" gap={0}>
                            <Text weight="bold" color="secondary">
                                {displayStatus.shelvedCount}
                            </Text>
                            <Text variant="bodySmall" color="secondary">
                                Shelved
                            </Text>
                        </Stack>
                        {displayStatus.shelvedSeverity && displayStatus.shelvedSeverity !== 'UNRECOGNIZED' && (
                            <Text variant="bodySmall" color="secondary">
                                ({displayStatus.shelvedSeverity})
                            </Text>
                        )}
                    </Stack>
                </Stack>
            </div>

            {!deduped.length ? (
                <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <Text color="secondary">No alarms to display</Text>
                </div>
            ) : (
                <div style={{
                    flex: 1,
                    overflow: 'auto',
                    minHeight: 0,
                    width: '100%'
                }}>
                    {options.pagination ? (
                        <>
                            {/* Render only the rows for the current page so updates don't reset pagination */}
                            <InteractiveTable
                                key={`page-${currentPage}`}
                                data={deduped.slice((currentPage - 1) * pageSize, currentPage * pageSize).map(d => ({ row: d, id: d.id }))}
                                getRowId={(row: any) => row.id}
                                columns={visibleColumns}
                                renderExpandedRow={options.showDetails ? renderSubComponent : undefined}
                                // Do not pass pageSize to let us control pagination externally
                            />

                            {/* Pagination controls */}
                            <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '8px' }}>
                                <Stack direction="row" gap={0.5} alignItems="center">
                                    <Button size="sm" variant="secondary" onClick={() => setCurrentPage(p => Math.max(1, p - 1))} disabled={currentPage <= 1} icon="angle-left" />

                                    {/* Render page buttons (limit to reasonable number) */}
                                    {(() => {
                                        const buttons = [] as any[];
                                        const maxButtons = 7;
                                        let start = 1;
                                        let end = totalPages;
                                        if (totalPages > maxButtons) {
                                            const half = Math.floor(maxButtons / 2);
                                            start = Math.max(1, currentPage - half);
                                            end = Math.min(totalPages, start + maxButtons - 1);
                                            if (end - start + 1 < maxButtons) {
                                                start = Math.max(1, end - maxButtons + 1);
                                            }
                                        }
                                        for (let p = start; p <= end; p++) {
                                            buttons.push(
                                                <Button key={p} size="sm" variant={p === currentPage ? 'primary' : 'secondary'} onClick={() => setCurrentPage(p)}>
                                                    {p}
                                                </Button>
                                            );
                                        }
                                        return buttons;
                                    })()}

                                    <Button size="sm" variant="secondary" onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))} disabled={currentPage >= totalPages} icon="angle-right" />
                                </Stack>
                            </div>
                        </>
                    ) : (
                        <InteractiveTable
                            key="without-pagination"
                            data={deduped.map(d => ({ row: d, id: d.id }))}
                            getRowId={(row: any) => row.id}
                            columns={visibleColumns}
                            renderExpandedRow={options.showDetails ? renderSubComponent : undefined}
                        />
                    )}
                </div>
            )}

            <Modal
                title={`${actionType.charAt(0).toUpperCase() + actionType.slice(1)} Alarm`}
                isOpen={modalOpen}
                onDismiss={() => setModalOpen(false)}
            >
                <Stack direction="column" gap={2}>
                    <Text>
                        {actionType === 'acknowledge' && 'Acknowledge this alarm to indicate you are aware of it.'}
                        {actionType === 'clear' && 'Clear this alarm to remove it from the active alarms list.'}
                        {actionType === 'shelve' && 'Shelve this alarm to temporarily hide it from the active alarms list.'}
                        {actionType === 'unshelve' && 'Unshelve this alarm to make it visible again in the active alarms list.'}
                    </Text>
                    {selectedAlarm && (
                        <Text color="secondary">
                            <strong>Alarm:</strong> {selectedAlarm.name}
                        </Text>
                    )}
                    {actionType === 'shelve' && (
                        <div>
                            <label htmlFor="shelve-duration" style={{ display: 'block', marginBottom: '8px' }}>
                                <strong>Duration</strong>
                            </label>
                            <Combobox
                                id="shelve-duration"
                                value={shelveDuration}
                                options={[
                                    { label: '15 minutes', value: 900000 },
                                    { label: '30 minutes', value: 1800000 },
                                    { label: '1 hour', value: 3600000 },
                                    { label: '2 hours', value: 7200000 },
                                    { label: '1 day', value: 86400000 },
                                    { label: 'unlimited', value: 0 },
                                ]}
                                onChange={(option) => { setShelveDuration(option.value as number); }}
                            />
                        </div>
                    )}
                    {actionType !== 'unshelve' && (
                        <TextArea
                            placeholder="Add a comment (optional)"
                            value={comment}
                            onChange={(e) => setComment(e.currentTarget.value)}
                            rows={3}
                        />
                    )}
                    <Stack direction="row" gap={1} justifyContent="flex-end">
                        <Button variant="secondary" onClick={() => setModalOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={executeAction} disabled={loading}>
                            {loading ? 'Processing...' : actionType.charAt(0).toUpperCase() + actionType.slice(1)}
                        </Button>
                    </Stack>
                </Stack>
            </Modal>
        </div>
    );
};

// Yamcs DataSourceType enum mapping (from mdb.pb.go)
const YAMCS_DATA_SOURCE_NAMES: Record<string, string> = {
  '0': 'TELEMETERED',
  '1': 'DERIVED',
  '2': 'CONSTANT',
  '3': 'LOCAL',
  '4': 'SYSTEM',
  '5': 'COMMAND',
  '6': 'COMMAND_HISTORY',
  '7': 'EXTERNAL1',
  '8': 'EXTERNAL2',
  '9': 'EXTERNAL3',
  '10': 'GROUND',
};

// Helper to get the human-readable Yamcs DataSourceType name for a given numeric value
function getDatasourceDisplayName(dsValue: string | number | undefined): string | null {
  if (dsValue === undefined || dsValue === null || dsValue === '') { return null; }
  const dsStr = String(dsValue);
  return YAMCS_DATA_SOURCE_NAMES[dsStr] || null;
}

export default AlarmsPanel;
