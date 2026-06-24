import { Combobox, ComboboxOption, InlineField, Stack } from '@grafana/ui';
import React, { useEffect, useRef, useState } from 'react';
import { QueryProps } from './constants';

type CommandInfo = {
    name: string;
    description: string;
};

export function CommandQuery({ query, onChange, datasource }: QueryProps) {
    const { endpoint } = query;

    const [command, setCommand] = useState(query.command);
    const [comboboxKey, setComboboxKey] = useState(0);

    const queryRef = useRef(query);
    queryRef.current = query;

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

    useEffect(() => {
        setComboboxKey((k) => k + 1);
    }, [endpoint]);

    // handle command selection
    const handleCommandChange = (v: ComboboxOption | null) => {
        setCommand(v?.value ?? '');
    };

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
