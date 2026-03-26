import { SelectableValue } from '@grafana/data';
import { Box, Button, Checkbox, Combobox, ComboboxOption, InlineField, Input, Stack } from '@grafana/ui';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { QueryProps } from './constants';
import { QueryTypeEditor } from './QueryTypeEditor';
import { getTemplateSrv } from '@grafana/runtime';

export function QueryEditor(props: QueryProps) {
    const { query, onChange, datasource, onRunQuery } = props;

    const [endpoints, setEndpoints] = useState<Record<string, any>>({});
    const [loading, setLoading] = useState(true);

    const setEndpoint = useCallback(
        (endpoint: string) => {
            onChange({
                ...query,
                endpoint,
            });
        },
        [onChange, query]
    );

    const setAsVariable = useCallback(
        (asVariable: boolean) => {
            onChange({
                ...query,
                asVariable,
            });
        },
        [onChange, query]
    );

    const setEndpointVariable = useCallback(
        (endpointVariable: string) => {
            onChange({
                ...query,
                endpointVariable,
            });
        },
        [onChange]
    );

    const setCustomVariableString = useCallback(
        (customVariableString: boolean) => {
            onChange({
                ...query,
                customVariableString,
            });
        },
        [onChange]
    );


    const fetchEndpoints = useCallback(async () => {
        setLoading(true);
        try {
            const data = await datasource.getResource('fetch/endpoints');
            setEndpoints(data);
            const keys = Object.keys(data);
            setEndpoint(keys.length === 1 ? keys[0] : query.endpoint ?? '');
        } catch (error) {
            console.error('Failed to fetch endpoints', error);
        } finally {
            setLoading(false);
        }
    }, [datasource]);

    useEffect(() => {
        fetchEndpoints();
    }, [fetchEndpoints]);

    // Run the query only when the serialized query content actually changes,
    // not on every object reference change (which would cause an infinite refresh loop).
    const prevQueryJson = useRef<string>('');
    useEffect(() => {
        const json = JSON.stringify(query);
        if (json !== prevQueryJson.current) {
            prevQueryJson.current = json;
            onRunQuery();
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [query]);

    const variableOptions = getTemplateSrv().getVariables().map((variable) => ({
        label: variable.label || variable.name,
        description: variable.description ?? undefined,
        value: `$${variable.name}`,
    }));

    const endpointOptions = Object.entries(endpoints).map(([id, endpoint]) => ({
        label: (endpoint as any).name || `#${id}`,
        description: (endpoint as any).description,
        value: id,
        status: endpoint,
    }));

    const renderSelect = () => {
        if (!query.asVariable) {
            return (
                <Combobox
                    options={endpointOptions}
                    value={query.endpoint ?? null}
                    onChange={(e: ComboboxOption | null) => { setEndpoint(e?.value ?? ''); }}
                    data-testid='endpoint-select'
                />
            );
        }

        if (query.customVariableString) {
            return (
                <Input
                    value={query.endpointVariable}
                    onChange={(e: SelectableValue) => setEndpointVariable(e.value)}
                />
            );
        }

        return (
            <Combobox
                options={variableOptions.map((o) => ({ label: o.label, value: o.value }))}
                value={query.endpointVariable ?? null}
                onChange={(e: ComboboxOption | null) => { setEndpointVariable(e?.value ?? ''); }}
                data-testid='endpoint-select'
            />
        );
    };


    return (
        <Box backgroundColor="primary" borderColor="weak" data-testid="query-type-editor">
            <Stack direction="row" alignItems="center">
                <InlineField label={query.asVariable ? "Endpoint Variable" : "Endpoint"} disabled={Object.keys(endpoints).length <= 1} loading={loading} grow>
                    {renderSelect()}
                </InlineField>
                {query.asVariable && <InlineField>
                    <Checkbox
                        label="Custom string"
                        value={query.customVariableString}
                        onChange={(e) => setCustomVariableString(e.currentTarget.checked)}
                    />
                </InlineField>}
                <InlineField>
                    <Checkbox
                        label="As variable"
                        value={query.asVariable}
                        onChange={(e) => setAsVariable(e.currentTarget.checked)}
                    />
                </InlineField>
                <Button onClick={fetchEndpoints} disabled={loading} size="sm" data-testid="fetch-endpoints-button">
                    Fetch endpoints
                </Button>
            </Stack>
            <QueryTypeEditor {...props} />
        </Box>
    );
}
