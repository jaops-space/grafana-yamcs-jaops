import { dateTime, PanelProps } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { Button, Icon, InteractiveTable, Modal, Stack, Text, TextArea, Tooltip, useTheme2 } from '@grafana/ui';
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

// Severity colors and icons
const severityConfig: Record<string, { color: string; icon: any }> = {
    'WATCH': { color: 'blue', icon: 'info-circle' },
    'WARNING': { color: 'orange', icon: 'exclamation-triangle' },
    'DISTRESS': { color: 'orange', icon: 'exclamation-triangle' },
    'CRITICAL': { color: 'red', icon: 'exclamation-circle' },
    'SEVERE': { color: 'red', icon: 'times-circle' },
};


const AlarmsPanel: React.FC<PanelProps<AlarmsOptions>> = ({ data, options, replaceVariables }) => {
    const theme = useTheme2();
    const [modalOpen, setModalOpen] = useState(false);
    const [selectedAlarm, setSelectedAlarm] = useState<AlarmEntry | null>(null);
    const [actionType, setActionType] = useState<'acknowledge' | 'clear' | 'shelve' | 'unshelve'>('acknowledge');
    const [comment, setComment] = useState('');
    const [loading, setLoading] = useState(false);

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
        return dedupeById(parsed);
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
            if (!alarms.length) return null;
            let highest = alarms[0].severity;
            for (const alarm of alarms) {
                if (severityOrder.indexOf(alarm.severity) > severityOrder.indexOf(highest)) {
                    highest = alarm.severity;
                }
            }
            return highest;
        };

        const unacknowledged = deduped.filter(a => !a.acknowledged && !a.shelved);
        const acknowledged = deduped.filter(a => a.acknowledged && !a.shelved);
        const shelved = deduped.filter(a => a.shelved);

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
                    ...(actionType === 'shelve' ? { shelveDuration: 3600000 } : {}), // Default 1 hour
                }
            );
            setModalOpen(false);
        } catch (error) {
            console.error(`Failed to ${actionType} alarm:`, error);
        } finally {
            setLoading(false);
        }
    }, [selectedAlarm, endpoint, datasource, actionType, comment]);

    // Build columns
    // Columns: State, Severity, Alarm time, Alarm name, Alarm type, Trip value, Live value, Status, Actions
    const columns = useMemo(() => [
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
                const dt = dateTime(info.row.original.row.triggerTime);
                return <span title={dt.fromNow()}>{dt.format('YYYY-MM-DD HH:mm:ss')}</span>;
            },
        },
        {
            id: 'name',
            header: 'Alarm name',
            accessorKey: 'name',
            cell: (info: any) => {
                const name = info.row.original.row.name;
                return (
                    <Tooltip content={name}>
                        <span>{name}</span>
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
            header: 'Trip value',
            accessorKey: 'triggerValue',
            cell: (info: any) => {
                // Try both triggerValue and tripValue for compatibility
                const row = info.row.original.row;
                return row.triggerValue || row.tripValue || '-';
            },
        },
        {
            id: 'currentValue',
            header: 'Live value',
            accessorKey: 'currentValue',
            cell: (info: any) => info.row.original.row.currentValue || '-',
        },
        {
            id: 'processOK',
            header: 'Status',
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
                        {alarm.acknowledged && (
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
        'severity', 'triggerTime', 'name', 'type', 'triggerValue', 'currentValue', 'processOK', 'actions',
    ];

    // Always use the yamcsDefaultOrder to ensure Trip value column is visible
    // The order matches Yamcs Web: Severity, Alarm time, Alarm name, Alarm type, Trip value, Live value, Status, Actions
    const visibleColumns = useMemo(() => {
        return yamcsDefaultOrder
            .map(fid => columns.find(col => col.id === fid))
            .filter((col): col is NonNullable<typeof col> => !!col);
    }, [columns]);

    // Render expanded row with details
    function renderSubComponent({ row }: { row: any }) {
        const alarm: AlarmEntry = row;
        return (
            <div style={{ padding: theme.spacing(1), background: theme.colors.background.secondary }}>
                <Stack direction="column" gap={1}>
                    <Text><strong>Full Parameter Path:</strong> {alarm.name}</Text>
                    <Text><strong>Trigger Time:</strong> {formatTime(alarm.triggerTime)}</Text>
                    {alarm.updateTime && <Text><strong>Last Update:</strong> {formatTime(alarm.updateTime)}</Text>}
                    <Text><strong>Trip Value:</strong> {alarm.triggerValue || '-'}</Text>
                    <Text><strong>Live Value:</strong> {alarm.currentValue || '-'}</Text>
                    <Text><strong>Violations:</strong> {alarm.violations}</Text>
                    <Text><strong>Sample Count:</strong> {alarm.count}</Text>
                    <Text><strong>Latching:</strong> {alarm.latching ? 'Yes' : 'No'}</Text>
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
        <>
            {/* GlobalAlarmStatus Bar - show computed stats */}
            <div style={{
                padding: theme.spacing(1, 2),
                marginBottom: theme.spacing(1),
                background: theme.colors.background.secondary,
                borderRadius: theme.shape.radius.default,
                border: `1px solid ${theme.colors.border.weak}`
            }}>
                <Stack direction="row" gap={3} alignItems="center" wrap="wrap">
                    {/* Unacknowledged Alarms */}
                    {displayStatus.unacknowledgedCount > 0 && (
                        <Stack direction="row" gap={1} alignItems="center">
                            <Icon
                                name="exclamation-triangle"
                                size="lg"
                                style={{ color: theme.colors.error.text }}
                            />
                            <Stack direction="column" gap={0}>
                                <Text weight="bold" color="error">
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
                    )}

                    {/* Acknowledged Alarms */}
                    {displayStatus.acknowledgedCount > 0 && (
                        <Stack direction="row" gap={1} alignItems="center">
                            <Icon
                                name="check"
                                size="lg"
                                style={{ color: theme.colors.info.text }}
                            />
                            <Stack direction="column" gap={0}>
                                <Text weight="bold" color="info">
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
                    )}

                    {/* Shelved Alarms */}
                    {displayStatus.shelvedCount > 0 && (
                        <Stack direction="row" gap={1} alignItems="center">
                            <Icon
                                name="clock-nine"
                                size="lg"
                                style={{ color: theme.colors.text.disabled }}
                            />
                            <Stack direction="column" gap={0}>
                                <Text weight="bold" style={{ color: theme.colors.text.disabled }}>
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
                    )}

                    {/* Show "No active alarms" when all counts are 0 */}
                    {displayStatus.unacknowledgedCount === 0 &&
                     displayStatus.acknowledgedCount === 0 &&
                     displayStatus.shelvedCount === 0 && (
                        <Stack direction="row" gap={1} alignItems="center">
                            <Icon
                                name="check-circle"
                                size="lg"
                                style={{ color: theme.colors.success.text }}
                            />
                            <Text color="success" weight="bold">
                                No active alarms
                            </Text>
                        </Stack>
                    )}
                </Stack>
            </div>

            {!deduped.length ? (
                <Stack alignItems="center" justifyContent="center" height="100%">
                    <Text color="secondary">No alarms to display</Text>
                </Stack>
            ) : (
                <div style={{ overflowY: 'scroll', overflowX: 'clip', width: '100%', height: '100%' }}>
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
        </>
    );
};

export default AlarmsPanel;
