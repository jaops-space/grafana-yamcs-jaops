import { SelectableValue } from '@grafana/data';
import { AsyncSelect, Checkbox, InlineField, Input, MultiSelect, Stack } from '@grafana/ui';
import { debounce } from 'lodash';
import React, { useEffect, useRef, useState } from 'react';
import { Optional, QueryField } from '../types';
import { FieldsOptions, QueryOptions, QueryProps } from './constants';

export function ParameterQuery({ query, onChange, datasource }: QueryProps) {
    // Extract query fields
    const { endpoint, type } = query;

    const [aggregatePath, setAggregatePath] = useState(query.aggregatePath || '');
    const [parameter, setParameter] = useState(query.parameter);
    const [fields, setFields] = useState(query.fields);
    const [isAggregate, setIsAggregate] = useState(Boolean(query.aggregatePath));

    // Get additional fields if available
    const queryTypeInfo = QueryOptions.find((o) => o.value === type);
    const additionalFields = queryTypeInfo?.additionalFields;

    useEffect(() => {
        onChange({
            ...query,
            parameter,
            fields,
            aggregatePath: isAggregate ? aggregatePath : '',
        })
    }, [parameter, aggregatePath, fields]);

    // Handle parameter selection
    const handleParameterChange = (v: SelectableValue<string>) => {
        setParameter(v.value ?? '');
    };

    // Toggle aggregation state
    const toggleAggregate = (toggle: boolean) => {
        setIsAggregate(() => {
            const newState = toggle;
            setAggregatePath(newState ? aggregatePath : '');
            return newState;
        });
    };

    // Handle aggregate path input changes
    const handleAggregatePathChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = e.target.value;
        setAggregatePath(value);
    };

    // Handle additional fields selection
    const handleFieldsChange = (arr: Array<SelectableValue<QueryField>>) => {
        setFields(arr.map((v) => v.value).filter(Boolean) as QueryField[]);
    };

    // Default parameter options
    const defaultOptions = parameter ? [{ label: parameter, value: parameter }] : [];

    // Debounced fetch function for loading parameters
    const debouncedFetchParameters = useRef(
        debounce(
            async (inputValue: string, endpoint: Optional<string>, callback: (options: Array<SelectableValue<string>>) => void) => {
                const parameters: string[] = await datasource.getResource(
                    `endpoint/${endpoint}/parameters`,
                    inputValue ? { q: inputValue } : undefined
                );
                callback(parameters.map((p) => ({ label: p, value: p })));
            },
            1000
        )
    ).current;

    // Load parameter options
    const loadParameters = (inputValue: string): Promise<Array<SelectableValue<string>>> => {
        return new Promise((resolve) => {
            debouncedFetchParameters(inputValue, endpoint, resolve);
        });
    };

    return (
        <>
            <Stack direction="row" alignItems="center">
                <Stack direction="row" alignItems="center" gap={0} grow={1}>
                    <InlineField label="Parameter to query" grow>
                        <AsyncSelect
                            loadOptions={loadParameters}
                            defaultOptions={defaultOptions}
                            onChange={handleParameterChange}
                            value={parameter ? { label: parameter, value: parameter } : null}
                            allowCreateWhileLoading
                            allowCustomValue
                        />
                    </InlineField>

                    {isAggregate && (
                        <InlineField label="." grow>
                            <Input
                                marginWidth={0}
                                onChange={handleAggregatePathChange}
                                placeholder='Path to value (case sensitive)'
                                value={aggregatePath}
                            />
                        </InlineField>
                    )}

                </Stack>

                <InlineField>
                    <Checkbox
                        value={isAggregate}
                        onChange={(e) => toggleAggregate(e.currentTarget.checked)}
                        label='Aggregate'
                    />
                </InlineField>

            </Stack>

            {additionalFields && (
                <Stack direction="row">
                    <InlineField label="Additional fields" grow>
                        <MultiSelect options={FieldsOptions} onChange={handleFieldsChange} values={fields} />
                    </InlineField>
                </Stack>
            )}
        </>
    );
}
