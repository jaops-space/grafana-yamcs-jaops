import { SelectableValue } from '@grafana/data';
import { Badge, Box, Button, Checkbox, Combobox, ComboboxOption, InlineField, Input, Select, Stack } from '@grafana/ui';
import React, { useCallback, useEffect, useState } from 'react';
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

    useEffect(() => {
        onRunQuery();
    }, [query]);

    const variableOptions = getTemplateSrv().getVariables().map((variable) => ({
        label: variable.label || variable.name,
        description: variable.description ?? undefined,
        value: `$${variable.name}`,
    }));

    const getStatusBadge = (endpoint: any) => {
        if (endpoint.error) {
            return <Badge color="red" text="Error" data-testid="status-badge" />;
        }
        return endpoint.online
            ? <Badge color="green" text="Online" data-testid="status-badge" />
            : <Badge color="orange" text="Offline" data-testid="status-badge" />;
    };

    const endpointOptions = Object.entries(endpoints).map(([id, endpoint]) => ({
        label: (endpoint as any).name || `#${id}`,
        description: (endpoint as any).description,
        value: id,
        status: endpoint,
    }));

    const getEndpointOptionLabel = (option: any) => (
        <Stack direction="row" justifyContent="space-between" alignItems="center" grow={1}>
            <span>{option.label}</span>
            {option.status && getStatusBadge(option.status)}
        </Stack>
    );

    const renderSelect = () => {
        if (!query.asVariable) {
            return (
                // eslint-disable-next-line @typescript-eslint/no-deprecated
                <Select
                    options={endpointOptions}
                    getOptionLabel={getEndpointOptionLabel}
                    value={query.endpoint}
                    onChange={(e: SelectableValue) => setEndpoint(e.value)}
                    isLoading={loading}
                    loadingMessage="Fetching endpoints..."
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
                options={variableOptions.map((o: any) => ({ label: o.label, value: o.value }))}
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
