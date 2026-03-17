import { Combobox, ComboboxOption, InlineField, Stack } from '@grafana/ui';
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
    const [options, setOptions] = useState<ComboboxOption[]>(
        query.command ? [{ label: query.command, value: query.command }] : []
    );

    useEffect(() => {
        onChange({
            ...query,
            command,
        })
    }, [command]);

    // Handle command selection
    const handleCommandChange = (v: ComboboxOption | null) => {
        setCommand(v?.value ?? '' );
    };

    // Debounced fetch function for loading commands
    const debouncedFetchCommands = useRef(
        debounce(
            async (inputValue: string, ep: Optional<string>) => {
                const parameters: CommandInfo[] = await datasource.getResource(
                    `endpoint/${ep}/commands`,
                    inputValue ? { q: inputValue } : undefined
                );
                setOptions(parameters.map((cmd) => ({ label: cmd.name, value: cmd.name, description: cmd.description })));
            },
            300
        )
    ).current;

    useEffect(() => {
        if (endpoint) {
            debouncedFetchCommands('', endpoint);
        } else {
            setOptions([]);
        }
    }, [endpoint]);

    return (
        <Stack direction="row" alignItems="center" gap={0}>
            <InlineField label="Command" grow>
                <Combobox
                    options={options}
                    onChange={handleCommandChange}
                    value={command ?? null}
                    onInputChange={(value) => {
                        debouncedFetchCommands(value ?? '', endpoint);
                    }}
                    onOpenMenu={() => {
                        debouncedFetchCommands('', endpoint);
                    }}
                />
            </InlineField>
        </Stack>
    );
}
