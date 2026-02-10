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
    processOK: boolean;
    triggered: boolean;
    latching: boolean;
    shelved: boolean;
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
    const [actionType, setActionType] = useState<'acknowledge' | 'clear' | 'shelve'>('acknowledge');
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

    const handleAction = useCallback(async (alarm: AlarmEntry, action: 'acknowledge' | 'clear' | 'shelve') => {
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
                                   actionType === 'clear' ? 'clear' : 'shelve';
            
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
    const columns = useMemo(() => [
        {
            id: 'triggerTime',
            header: 'Trigger Time',
            accessorKey: 'triggerTime',
            cell: (info: any) => formatTime(info.row.original.row.triggerTime),
        },
        {
            id: 'name',
            header: 'Parameter',
            accessorKey: 'name',
            cell: (info: any) => {
                const name = info.row.original.row.name;
                return (
                    <Tooltip content={name}>
                        <span>{name.split('/').pop() || name}</span>
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
            id: 'type',
            header: 'Type',
            cell: (info: any) => info.row.original.row.type,
        },
        {
            id: 'currentValue',
            header: 'Current Value',
            cell: (info: any) => info.row.original.row.currentValue || '-',
        },
        {
            id: 'acknowledged',
            header: 'Ack',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                if (alarm.acknowledged) {
                    return (
                        <Tooltip content={`Acknowledged by ${alarm.acknowledgedBy || 'unknown'}`}>
                            <Icon name="check-circle" color="green" />
                        </Tooltip>
                    );
                }
                return <Icon name="times-circle" color="red" />;
            },
        },
        {
            id: 'processOK',
            header: 'Status',
            cell: (info: any) => {
                const alarm = info.row.original.row;
                const isOk = alarm.processOK;
                return (
                    <Tooltip content={isOk ? 'Within limits' : 'Out of limits'}>
                        <Icon 
                            name={isOk ? 'check' : 'exclamation-triangle'} 
                            color={isOk ? 'green' : 'orange'} 
                        />
                    </Tooltip>
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
                        {alarm.acknowledged && alarm.processOK && (
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
                        {!alarm.shelved && (
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
                        )}
                    </Stack>
                );
            },
        },
    ], [handleAction]);

    // Filter columns based on options
    const visibleColumns = useMemo(() => {
        return columns.filter(col => options.visibleFields.includes(col.id));
    }, [columns, options.visibleFields]);

    // Render expanded row with details
    function renderSubComponent({ row }: { row: any }) {
        const alarm: AlarmEntry = row;
        return (
            <div style={{ padding: theme.spacing(1), background: theme.colors.background.secondary }}>
                <Stack direction="column" gap={1}>
                    <Text><strong>Full Name:</strong> {alarm.name}</Text>
                    <Text><strong>Trigger Time:</strong> {formatTime(alarm.triggerTime)}</Text>
                    {alarm.updateTime && <Text><strong>Last Update:</strong> {formatTime(alarm.updateTime)}</Text>}
                    <Text><strong>Trigger Value:</strong> {alarm.triggerValue || '-'}</Text>
                    <Text><strong>Current Value:</strong> {alarm.currentValue || '-'}</Text>
                    <Text><strong>Violations:</strong> {alarm.violations}</Text>
                    <Text><strong>Sample Count:</strong> {alarm.count}</Text>
                    <Text><strong>Latching:</strong> {alarm.latching ? 'Yes' : 'No'}</Text>
                    {alarm.acknowledged && (
                        <>
                            <Text><strong>Acknowledged By:</strong> {alarm.acknowledgedBy}</Text>
                            {alarm.acknowledgeTime && <Text><strong>Acknowledge Time:</strong> {formatTime(alarm.acknowledgeTime)}</Text>}
                        </>
                    )}
                </Stack>
            </div>
        );
    }

    if (!deduped.length) {
        return (
            <Stack alignItems="center" justifyContent="center" height="100%">
                <Text color="secondary">No active alarms</Text>
            </Stack>
        );
    }

    return (
        <>
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
                    </Text>
                    {selectedAlarm && (
                        <Text color="secondary">
                            <strong>Alarm:</strong> {selectedAlarm.name}
                        </Text>
                    )}
                    <TextArea
                        placeholder="Add a comment (optional)"
                        value={comment}
                        onChange={(e) => setComment(e.currentTarget.value)}
                        rows={3}
                    />
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
