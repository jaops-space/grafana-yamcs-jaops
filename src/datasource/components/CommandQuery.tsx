import { SelectableValue } from '@grafana/data';
import { AsyncSelect, InlineField, Stack } from '@grafana/ui';
import { debounce } from 'lodash';
import React, { useEffect, useRef, useState } from 'react';
import { Optional } from '../types';
import { QueryProps } from './constants';

type CommandInfo = {
    name: string;
    description: string;
}

export function CommandQuery({ query, onChange, datasource }: QueryProps) {

    // Extract query fields
    const { endpoint } = query;
    
    const [command, setCommand] = useState(query.command);

    useEffect(() => {
        onChange({
            ...query,
            command,
        })
    }, [command]);

    // Handle command selection
    const handleCommandChange = (v: SelectableValue<string>) => {
        setCommand(v.value ?? '' );
    };

    const defaultOptions = command ? [{ label: command, value: command }] : [];

    // Debounced fetch function for loading commands
    const debouncedFetchCommands = useRef(
        debounce(
            async (inputValue: string, endpoint: Optional<string>, callback: (options: Array<SelectableValue<string>>) => void) => {
                const parameters: CommandInfo[] = await datasource.getResource(
                    `endpoint/${endpoint}/commands`,
                    inputValue ? { q: inputValue } : undefined
                );
                callback(parameters.map((command) => ({ label: command.name, value: command.name, description: command.description })));
            },
            1000
        )
    ).current;

    // Load parameter options
    const loadCommands = (inputValue: string): Promise<Array<SelectableValue<string>>> => {
        return new Promise((resolve) => {
            debouncedFetchCommands(inputValue, endpoint, resolve);
        });
    };

    return (
        <Stack direction="row" alignItems="center" gap={0}>
            <InlineField label="Command" grow>
                <AsyncSelect
                    loadOptions={loadCommands}
                    defaultOptions={defaultOptions}
                    onChange={handleCommandChange}
                    value={command ? { label: command, value: command } : null}
                />
            </InlineField>
        </Stack>
    );
}
