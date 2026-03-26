import { Combobox, ComboboxOption, InlineField, Stack } from '@grafana/ui';
import React, { useEffect, useRef, useState } from 'react';
import { QueryProps } from './constants';

type CommandInfo = {
    name: string;
    description: string;
}

export function CommandQuery({ query, onChange, datasource }: QueryProps) {

    // Extract query fields
    const { endpoint } = query;

    const [command, setCommand] = useState(query.command);
    // comboboxKey forces the Combobox to remount (and re-fetch options) when endpoint changes
    const [comboboxKey, setComboboxKey] = useState(0);

    // Keep a ref to the latest query so the effect below doesn't need query as a dependency
    // (which would cause an infinite loop: onChange -> query changes -> effect fires -> onChange -> …)
    const queryRef = useRef(query);
    queryRef.current = query;

    // Only propagate when the user actually picks a different command
    const prevCommandRef = useRef(command);
    useEffect(() => {
        if (command === prevCommandRef.current) {
            return;
        }
        prevCommandRef.current = command;
        onChange({
            ...queryRef.current,
            command,
        });
    }, [command, onChange]);

    // Reset combobox when endpoint changes so options are re-fetched
    useEffect(() => {
        setComboboxKey((k) => k + 1);
    }, [endpoint]);

    // Handle command selection
    const handleCommandChange = (v: ComboboxOption | null) => {
        setCommand(v?.value ?? '');
    };

    // Async options function — Grafana Combobox calls this on open and on every keypress
    const fetchOptions = async (inputValue: string): Promise<ComboboxOption[]> => {
        if (!endpoint) {
            return [];
        }
        const parameters: CommandInfo[] = await datasource.getResource(
            `endpoint/${endpoint}/commands`,
            inputValue ? { q: inputValue } : undefined
        );
        return parameters.map((cmd) => ({ label: cmd.name, value: cmd.name, description: cmd.description }));
    };

    return (
        <Stack direction="row" alignItems="center" gap={0}>
            <InlineField label="Command" grow>
                <Combobox
                    key={comboboxKey}
                    options={fetchOptions}
                    onChange={handleCommandChange}
                    value={command ?? null}
                />
            </InlineField>
        </Stack>
    );
}
