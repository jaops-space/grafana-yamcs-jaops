import { dateTime, PanelProps } from '@grafana/data';
import { Icon, InteractiveTable, Stack, Text, Tooltip } from '@grafana/ui';
import React, { useMemo } from 'react';

// ------------------
// Type Definitions
// ------------------

interface CommandAck {
    status: string;
    time: string; // ISO string
    message: string;
}

interface CommandArgument {
    name: string;
    value: string;
}

interface CommandEntry {
    id: string;
    time: string;
    command: string;
    comment?: string;
    arguments: CommandArgument[];
    queued?: CommandAck;
    released?: CommandAck;
    sent?: CommandAck;
    extraAcks: Record<string, CommandAck>;
    completion: CommandAck;
}

// ------------------
// Utility Functions
// ------------------

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

function timeDiffMs(start: string, end: string): string {
    const diff = dateTime(end).diff(dateTime(start), 'milliseconds');
    return `+ ${diff} ms`;
}

const dedupeById = (entries: CommandEntry[]): CommandEntry[] => {
    const map = new Map<string, number>();
    const result: CommandEntry[] = [];

    for (const entry of entries) {
        const id = entry.id;
        if (map.has(id)) {
            const index = map.get(id)!;
            result[index] = deepCombine(result[index], entry);
        } else {
            result.push({ ...entry }); // shallow clone at least
            map.set(id, result.length - 1);
        }
    }

    return result;
};

// ------------------
// Table Column Definitions
// ------------------

const columns = [
    {
        id: 'time',
        header: 'Time',
        accessorKey: 'time',
        cell: (info: any) => formatTime(info.row.original.row.time),
    },
    {
        id: 'command',
        header: 'Command',
        accessorKey: 'command',
        cell: (info: any) => info.row.original.row.command
    },
    {
        id: 'comment',
        header: 'Comment',
        cell: (info: any) => {
            const comment = info.row.original.row.comment;
            if (!comment) {
                return null;
            }
            return (
                <Tooltip content={comment}>
                    <Icon name='comment-alt-message' />
                </Tooltip>
            );
        },
    },
    ...['queued', 'released', 'sent'].map((key) => ({
        id: key,
        header: key[0].toUpperCase(),
        tooltip: '',
        cell: (info: any) => {
            const entry = info.row.original.row;
            const ack: CommandAck | undefined = entry[key];
            if (!ack) {
                return null;
            }

            const statusOk = ack.status.toLowerCase() === 'ok';
            const statusNok = ack.status.toLowerCase() === 'nok';
            const icon = statusOk ? 'check-circle' : 'exclamation-triangle';
            const color = statusOk ? 'green' : (statusNok ? 'red' : 'orange');
            const timeDelta = timeDiffMs(entry.time, ack.time);

            let msg = "";
            if (ack.message) {
                msg = "\n" + ack.message;
            }

            return (
                <Tooltip content={<>{`${key.charAt(0).toUpperCase() + key.slice(1)}: ${ack.status} (${timeDelta})`}<br />{msg}</>}>
                    <Icon name={icon} size="md" color={color} />
                </Tooltip>
            );
        },
    })),
    {
        id: 'extraAcks',
        header: 'Extra Acks.',
        tooltip: '',
        cell: (info: any) => {
            const entry = info.row.original.row;
            const acks: Record<string, CommandAck> | undefined = entry.extraAcks;
            if (!acks) {
                return null;
            }

            return <>
                {Object.keys(acks).map(key => {
                    const ack = acks[key];
                    const statusOk = ack.status.toLowerCase() === 'ok';
                    const statusNok = ack.status.toLowerCase() === 'nok';
                    const icon = statusOk ? 'check-circle' : 'exclamation-triangle';
                    const color = statusOk ? 'green' : (statusNok ? 'red' : 'orange');
                    const timeDelta = timeDiffMs(entry.time, ack.time);
                    let msg = "";
                    if (ack.message) {
                        msg = "\n" + ack.message;
                    }

                    return (
                        <Tooltip key={key} content={<>{`${key}: ${ack.status} (${timeDelta})`}<br />{msg}</>}>
                            <Icon name={icon} size="md" color={color} />
                        </Tooltip>
                    );
                })}
            </>;

        },
    },
    {
        id: 'completion',
        header: 'Completion',
        tooltip: '',
        cell: (info: any) => {
            const entry = info.row.original.row;
            const ack: CommandAck | undefined = entry.completion;
            if (!ack) {
                return null;
            }

            let statusText = "";
            switch (ack.status.toLowerCase()) {
                case "ok":
                    statusText = "COMPLETED";
                    break;
                case "timeout":
                    statusText = "TIMED OUT";
                    break;
                case "nok":
                    statusText = "FAILED";
                    break;
            }
            const statusOk = ack.status.toLowerCase() === 'ok';
            const statusNok = ack.status.toLowerCase() === 'nok';
            const icon = statusOk ? 'check-circle' : 'exclamation-triangle';
            const color = statusOk ? 'green' : (statusNok ? 'red' : 'orange');
            const variant = statusOk ? 'success' : (statusNok ? 'error' : 'warning');
            const timeDelta = timeDiffMs(entry.time, ack.time);

            return (
                <Tooltip content={`${ack.message ?? ""} (${timeDelta})`}>
                    <Stack direction='row' gap={1} alignItems='center'>
                        <Icon name={icon} size="md" color={color} />
                        <Text variant='h5' color={variant}>
                            {statusText}
                        </Text>
                    </Stack>

                </Tooltip>
            );

        },
    }
];

// ------------------
// Row Expansion Renderer
// ------------------

function renderSubComponent({ row }: { row: any }) {
    const args: CommandArgument[] = row.arguments;
    if (!args.length) {
        return null;
    }

    return (
        <div>
            <strong>Arguments:</strong>
            <ul style={{ paddingLeft: 20, marginTop: 1 }}>
                {args.map((arg) => (
                    <li key={arg.name}>
                        <strong>{arg.name}</strong>: {arg.value}
                    </li>
                ))}
            </ul>
        </div>
    );
}

// ------------------
// Main Component
// ------------------

const CommandHistoryPanel: React.FC<PanelProps> = ({ data }) => {
    const deduped = useMemo(() => {
        const frame = data.series[0];
        if (!frame || !frame.fields.length) { return []; }

        const rawField = frame.fields.find((f) => f.name === 'commands');
        if (!rawField) { return []; }

        const raw = rawField.values as any[];

        const parsed: CommandEntry[] = raw.filter(Boolean).reverse();
        console.log(parsed);
        const deduped = dedupeById(parsed);
        console.log(deduped);
        return deduped;
    }, [data]);

    if (!deduped.length) {
        return <div>No data</div>;
    }

    return (
        <div style={{ overflowY: 'scroll', overflowX: 'clip', width: '100%', height: '100%' }}>
            <InteractiveTable
                data={deduped.map(d => ({ row: d, id: d.id }))}
                getRowId={(row: any) => row.id}
                columns={columns}
                renderExpandedRow={renderSubComponent}
                showExpandAll={true}
            />
        </div>
    );
};

export default CommandHistoryPanel;
