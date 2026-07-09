import { getTemplateSrv } from '@grafana/runtime';
import { Box, Button, Checkbox, Combobox, ComboboxOption, InlineField, Input, Stack } from '@grafana/ui';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { QueryEditorModelProps, QueryProps } from './constants';
import { QueryTypeEditor } from './QueryTypeEditor';

export function QueryEditor(props: QueryProps) {
    const { query, onChange, datasource, onRunQuery } = props;
    const queryRef = useRef(query);
    const onChangeRef = useRef(onChange);

    const [endpoints, setEndpoints] = useState<Record<string, any>>({});
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        queryRef.current = query;
    }, [query]);

    useEffect(() => {
        onChangeRef.current = onChange;
    }, [onChange]);

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
        [onChange, query]
    );

    const setCustomVariableString = useCallback(
        (customVariableString: boolean) => {
            onChange({
                ...query,
                customVariableString,
            });
        },
        [onChange, query]
    );

    const fetchEndpoints = useCallback(async () => {
        setLoading(true);
        try {
            const data = await datasource.getResource('fetch/endpoints');
            setEndpoints(data);
            const keys = Object.keys(data);
            onChangeRef.current({
                ...queryRef.current,
                endpoint: keys.length === 1 ? keys[0] : (queryRef.current.endpoint ?? ''),
            });
        } catch (error) {
            console.error('Failed to fetch endpoints', error);
        } finally {
            setLoading(false);
        }
    }, [datasource]);

    useEffect(() => {
        fetchEndpoints();
    }, [fetchEndpoints]);

    const variableOptions = getTemplateSrv()
        .getVariables()
        .map((variable) => ({
            label: variable.label || variable.name,
            description: variable.description ?? undefined,
            value: `$${variable.name}`,
        }));

    const endpointOptions: Array<ComboboxOption<string>> = Object.entries(endpoints).map(([id, endpoint]) => {
        const online = (endpoint as any).online as boolean;
        const desc = (endpoint as any).description;
        const statusLabel = online ? '🟢 Online' : '🔴 Offline';
        return {
            label: (endpoint as any).name || `#${id}`,
            description: desc ? `${desc} — ${statusLabel}` : statusLabel,
            value: id,
        };
    });

    const selectedEndpointOnline: boolean | null =
        query.endpoint && endpoints[query.endpoint] != null
            ? ((endpoints[query.endpoint] as any).online ?? false)
            : null;

    const queryEditorModelProps: QueryEditorModelProps = { query, onChange, datasource };

    const renderSelect = () => {
        if (!query.asVariable) {
            return (
                <div style={{ position: 'relative', width: '100%' }}>
                    <Combobox
                        options={endpointOptions}
                        value={query.endpoint ?? null}
                        onChange={(e: ComboboxOption | null) => {
                            setEndpoint(e?.value ?? '');
                        }}
                        data-testid="endpoint-select"
                    />
                    {selectedEndpointOnline !== null && (
                        <span
                            style={{
                                position: 'absolute',
                                right: 36,
                                top: '50%',
                                transform: 'translateY(-50%)',
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: 5,
                                pointerEvents: 'none',
                                zIndex: 1,
                            }}
                        >
                            <span
                                style={{
                                    width: 10,
                                    height: 10,
                                    borderRadius: '50%',
                                    background: selectedEndpointOnline ? '#36BA00' : '#E02F44',
                                    display: 'inline-block',
                                    flexShrink: 0,
                                }}
                            />
                            <span
                                style={{
                                    fontSize: '0.85em',
                                    color: selectedEndpointOnline ? '#36BA00' : '#E02F44',
                                    whiteSpace: 'nowrap',
                                }}
                            >
                                {selectedEndpointOnline ? 'Online' : 'Offline'}
                            </span>
                        </span>
                    )}
                </div>
            );
        }

        if (query.customVariableString) {
            return (
                <Input
                    value={query.endpointVariable || ''}
                    onChange={(e) => setEndpointVariable(e.currentTarget.value)}
                />
            );
        }

        return (
            <Combobox
                options={variableOptions.map((o) => ({ label: o.label, value: o.value }))}
                value={query.endpointVariable ?? null}
                onChange={(e: ComboboxOption | null) => {
                    setEndpointVariable(e?.value ?? '');
                }}
                data-testid="endpoint-select"
            />
        );
    };

    return (
        <Box backgroundColor="primary" borderColor="weak" data-testid="query-type-editor">
            <Stack direction="row" alignItems="center">
                <InlineField
                    label={query.asVariable ? 'Endpoint Variable' : 'Endpoint'}
                    disabled={!query.asVariable && Object.keys(endpoints).length <= 1}
                    loading={loading}
                    grow
                >
                    {renderSelect()}
                </InlineField>
                {query.asVariable && (
                    <InlineField>
                        <Checkbox
                            label="Custom string"
                            checked={Boolean(query.customVariableString)}
                            onChange={(e) => setCustomVariableString(e.currentTarget.checked)}
                        />
                    </InlineField>
                )}
                <InlineField>
                    <Checkbox
                        label="As variable"
                        checked={Boolean(query.asVariable)}
                        onChange={(e) => setAsVariable(e.currentTarget.checked)}
                    />
                </InlineField>
                <Button onClick={fetchEndpoints} disabled={loading} size="sm" data-testid="fetch-endpoints-button">
                    Fetch endpoints
                </Button>
                <Button
                    onClick={onRunQuery}
                    disabled={loading}
                    size="sm"
                    variant="success"
                    data-testid="fetch-endpoints-button"
                >
                    Query
                </Button>
            </Stack>
            <QueryTypeEditor {...queryEditorModelProps} />
        </Box>
    );

}
