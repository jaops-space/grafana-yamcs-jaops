import { dateTime, PanelProps } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { Button, Icon, InteractiveTable, Modal, Select, Stack, Text, TextArea, Tooltip, useTheme2 } from '@grafana/ui';
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
    count: number;
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


const AlarmsPanel: React.FC<PanelProps<AlarmsOptions>> = ({ data, options, replaceVariables }) => {
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
            id: 'processOK',
            header: 'State',
            cell: (info: any) => {
                const alarm = info.row.original.row;

                // Determine the alarm state based on flags (priority order like Yamcs Web)
                let statusText = '';
                let statusColor: any = 'primary';
                let icon: any = 'circle';

                if (alarm.shelved) {
                    statusText = 'Shelved';
                    statusColor = 'disabled';
                    icon = 'clock-nine';
                } else if (alarm.cleared) {
                    statusText = 'Cleared';
                    statusColor = 'success';
                    icon = 'check-circle';
                } else if (!alarm.triggered) {
                    statusText = 'OK';
                    statusColor = 'success';
                    icon = 'check-circle';
                } else if (alarm.acknowledged) {
                    statusText = 'Acknowledged';
                    statusColor = 'info';
                    icon = 'check';
                } else {
                    statusText = 'Triggered';
                    statusColor = 'error';
                    icon = 'exclamation-triangle';
                }

                return (
                    <Stack direction="row" gap={0.5} alignItems="center">
                        <Icon name={icon} color={statusColor} />
                        <Text color={statusColor}>{statusText}</Text>
                    </Stack>
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
            id: 'alarmTime',
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
            id: 'triggerTimestamp',
            header: 'Trigger Timestamp',
            accessorKey: 'triggerTime',
            cell: (info: any) => {
                const dt = dateTime(info.row.original.row.triggerTime);
                return <span title={dt.fromNow()}>{dt.format('YYYY-MM-DD HH:mm:ss')}</span>;
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
                // Try both triggerValue and tripValue for compatibility
                const row = info.row.original.row;
                return row.triggerValue || row.tripValue || '-';
            },
        },
        {
            id: 'mostSevereValue',
            header: 'Most severe value',
            accessorKey: 'mostSevereValue',
            cell: (info: any) => info.row.original.row.mostSevereValue || '-',
        },
        {
            id: 'currentValue',
            header: 'Live value',
            accessorKey: 'currentValue',
            cell: (info: any) => info.row.original.row.currentValue || '-',
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
                                icon="clock-nine"
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
                                icon="eye"
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
    ], [handleAction]);

    // Filter columns based on options, but if not set, use Yamcs Web default order
    const yamcsDefaultOrder = [
        'processOK', 'severity', 'alarmTime', 'triggerTimestamp', 'name', 'type', 'triggerValue', 'mostSevereValue', 'currentValue', 'actions',
    ];

    // Always use the yamcsDefaultOrder to ensure all columns are visible
    // The order matches Yamcs Web: State, Severity, Alarm time (duration), Trigger Timestamp, Alarm name, Alarm type, Trigger value, Most severe value, Live value, Actions
    const visibleColumns = useMemo(() => {
        return yamcsDefaultOrder
            .map(fid => columns.find(col => col.id === fid))
            .filter((col): col is NonNullable<typeof col> => !!col);
    }, [columns]);

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
                    {!isEventAlarm && alarm.mostSevereValue && <Text><strong>Most Severe Value:</strong> {alarm.mostSevereValue}</Text>}
                    <Text><strong>{isEventAlarm ? 'Current Event:' : 'Live Value:'}</strong> {alarm.currentValue || '-'}</Text>
                    <Text><strong>Violations:</strong> {alarm.violations}</Text>
                    <Text><strong>Count:</strong> {alarm.count}</Text>
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
                                                                <Text>dataSource: {alarm.parameterInfo.dataSource}</Text>
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
                        <InteractiveTable
                            key="with-pagination"
                            data={deduped.map(d => ({ row: d, id: d.id }))}
                            getRowId={(row: any) => row.id}
                            columns={visibleColumns}
                            renderExpandedRow={options.showDetails ? renderSubComponent : undefined}
                            pageSize={Math.max(1, options.pageSize)}
                        />
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
                            <Select
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
                                onChange={(option) => setShelveDuration(option.value as number)}
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

export default AlarmsPanel;
