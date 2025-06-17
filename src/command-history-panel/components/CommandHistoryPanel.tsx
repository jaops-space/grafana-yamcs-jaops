import { dateTime, PanelProps } from '@grafana/data';
import { Icon, InteractiveTable, Tooltip } from '@grafana/ui';
import React from 'react';

// ------------------
// Type Definitions
// ------------------

interface CommandAck {
    status: string;
    time: string; // ISO string
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
}

// ------------------
// Utility Functions
// ------------------

function formatTime(iso: string): string {
    return dateTime(iso).format('YYYY-MM-DD HH:mm:ss');
}

function timeDiffMs(start: string, end: string): string {
    const diff = dateTime(end).diff(dateTime(start), 'milliseconds');
    return `+ ${diff} ms`;
}

function dedupeById(entries: CommandEntry[]): CommandEntry[] {
    const map = new Map<string, {index: number, entry: CommandEntry}>();
    const result: CommandEntry[] = [];

    for (const entry of entries) {
        if (map.has(entry.id)) {
            let mapEntry = map.get(entry.id);
            let index = mapEntry!.index, oldEntry = mapEntry!.entry;
            entry.comment ||= oldEntry.comment;
            entry.arguments = [...entry.arguments, ...oldEntry.arguments];
            entry.sent ||= oldEntry.sent;
            entry.released ||= oldEntry.released;
            entry.queued ||= oldEntry.queued;
            map.set(entry.id, {index, entry});
            result[index] = entry;
        } else {
            const ind = result.push(entry) - 1;
            map.set(entry.id, {index: ind, entry});
        }
    }

    return result;
}

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
        tooltip: 'test',
        cell: (info: any) => {
            const entry = info.row.original.row;
            const ack: CommandAck | undefined = entry[key];
            if (!ack) {
                return null;
            }

            const statusOk = ack.status.toLowerCase() === 'ok';
            const icon = statusOk ? 'check-circle' : 'exclamation-triangle';
            const color = statusOk ? 'green' : 'orange';
            const timeDelta = timeDiffMs(entry.time, ack.time);

            return (
                <Tooltip content={`${key.charAt(0).toUpperCase() + key.slice(1)}: ${ack.status} (${timeDelta})`}>
                    <Icon name={icon} size="md" color={color} />
                </Tooltip>
            );
        },
    })),
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
    const frame = data.series[0];

    if (!frame || !frame.fields.length) {
        return <div>No data</div>;
    }

    const rawField = frame.fields.find((f) => f.name === 'commands');
    if (!rawField) {
        return <div>Field &apos;commands&apos; not found</div>;
    }

    const raw = rawField.values as any[];

    const parsed: CommandEntry[] = raw.filter(Boolean).reverse();

    const deduped = dedupeById(parsed);

    return (
        <InteractiveTable
            data={deduped.map(d => ({ row: d, id: d.id }))}
            getRowId={(row: any) => row.id}
            columns={columns}
            renderExpandedRow={renderSubComponent}
        />
    );
};

export default CommandHistoryPanel;
