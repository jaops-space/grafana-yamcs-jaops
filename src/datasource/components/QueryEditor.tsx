import { SelectableValue } from '@grafana/data';
import { Badge, Box, Button, InlineField, Select, Stack } from '@grafana/ui';
import React, { useCallback, useEffect, useState } from 'react';
import { QueryProps } from './constants';
import { QueryTypeEditor } from './QueryTypeEditor';

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

    return (
        <Box backgroundColor="primary" borderColor="weak" data-testid="query-type-editor">
            <Stack direction="row" alignItems="center">
                <InlineField label="Endpoint" disabled={Object.keys(endpoints).length <= 1} loading={loading} grow>
                    <Select
                        options={Object.entries(endpoints).map(([id, endpoint]) => ({
                            label: endpoint.name || `Unnamed host ${id}`,
                            status: endpoint,
                            description: endpoint.description,
                            value: id,
                        }))}
                        getOptionLabel={(value: any) => (
                            <Stack direction={'row'} justifyContent={'space-between'}>
                                <span>{value.label}</span>
                                <span style={{ zIndex: 212 }}>{getBadge(value.status)}</span>
                            </Stack>
                        )}
                        value={query.endpoint}
                        onChange={(e: SelectableValue) => setEndpoint(e.value)}
                        isLoading={loading}
                        loadingMessage="Fetching endpoints..."
                        data-testid='endpoint-select'
                    ></Select>
                </InlineField>
                <Button onClick={fetchEndpoints} disabled={loading} size="sm" data-testid="fetch-endpoints-button">
                    Fetch endpoints
                </Button>
            </Stack>
            {endpoints[query.endpoint ?? '']?.online && <QueryTypeEditor {...props} />}
        </Box>
    );
}
