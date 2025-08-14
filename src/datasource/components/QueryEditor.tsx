import { SelectableValue } from '@grafana/data';
import { Badge, Box, Button, Checkbox, InlineField, Input, Select, Stack, Text } from '@grafana/ui';
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
        console.log(query);
    }, [query]);

    const getBadge = (status: any) => {
        if (!status) {
            return <></>;
        }
        if (status.error) {
            return <Badge color="red" tooltip={status.error} text="Error" data-testid="status-badge" />;
        }
        return status.online ? <Badge color="green" text="Online" data-testid="status-badge" /> : <Badge color="orange" text="Offline" data-testid="status-badge" />;
    };

    const variableOptions = getTemplateSrv().getVariables().map((variable) => ({
        label: variable.label || variable.name,
        description: variable.description ?? undefined,
        value: `$${variable.name}`,
    }));

    const endpointOptions = Object.entries(endpoints).map(([id, endpoint]) => ({
        label: endpoint.name || <Text variant='code'>#{id}</Text>,
        status: endpoint,
        description: endpoint.description,
        value: id,
    }));

    const getEndpointLabel = (value: any) => (
        <Stack direction='row' justifyContent='space-between'>
            <span>{value.label}</span>
            <span style={{ zIndex: 212 }}>{getBadge(value.status)}</span>
        </Stack>
    );

    const renderSelect = () => {
        if (!query.asVariable) {
            return (
                <Select
                    options={endpointOptions}
                    getOptionLabel={getEndpointLabel}
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
            <Select
                options={variableOptions}
                value={query.endpointVariable}
                onChange={(e: SelectableValue) => setEndpointVariable(e.value)}
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
            {endpoints[query.endpoint ?? '']?.online && <QueryTypeEditor {...props} />}
        </Box>
    );
}
