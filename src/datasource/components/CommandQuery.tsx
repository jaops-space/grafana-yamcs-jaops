import { Combobox, ComboboxOption, InlineField, Stack } from '@grafana/ui';
import React, { useEffect, useState } from 'react';
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

    useEffect(() => {
        onChange({
            ...query,
            command,
        })
    }, [command, query, onChange]);

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
