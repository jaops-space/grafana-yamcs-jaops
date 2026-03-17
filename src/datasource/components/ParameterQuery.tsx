import { SelectableValue } from '@grafana/data';
import { Combobox, ComboboxOption, Checkbox, InlineField, Input, MultiSelect, Stack } from '@grafana/ui';
import { debounce } from 'lodash';
import React, { useEffect, useRef, useState } from 'react';
import { QueryField } from '../types';
import { FieldsOptions, QueryOptions, QueryProps } from './constants';

export function ParameterQuery({ query, onChange, datasource }: QueryProps) {
    const { endpoint } = query;

    const [aggregatePath, setAggregatePath] = useState(query.aggregatePath || '');
    const [parameter, setParameter] = useState(query.parameter);
    const [fields, setFields] = useState(query.fields);
    const [isAggregate, setIsAggregate] = useState(Boolean(query.aggregatePath));
    const [options, setOptions] = useState<ComboboxOption[]>(
        query.parameter ? [{ label: query.parameter, value: query.parameter }] : []
    );

    const queryTypeInfo = QueryOptions.find((o) => o.value === query.type);
    const additionalFields = queryTypeInfo?.additionalFields;

    useEffect(() => {
        onChange({
            ...query,
            parameter,
            fields,
            aggregatePath: isAggregate ? aggregatePath : '',
        });
    }, [parameter, aggregatePath, fields, isAggregate, query, onChange]);

    useEffect(() => {
        if (endpoint) {
            debouncedFetchParameters('', endpoint);
        }
    }, [endpoint]);

    const handleParameterChange = (v: ComboboxOption | null) => {
        setParameter(v?.value ?? '');
    };

    const debouncedFetchParameters = useRef(
        debounce(
            async (inputValue: string, ep: string | undefined) => {
                const parameters: string[] = await datasource.getResource(
                    `endpoint/${ep}/parameters`,
                    inputValue ? { q: inputValue } : undefined
                );
                setOptions(parameters.map((p) => ({ label: p, value: p })));
            },
            300
        )
    ).current;

    return (
        <>
            <Stack direction="row" alignItems="center">
                <Stack direction="row" alignItems="center" gap={0} grow={1}>
                    <InlineField label="Parameter to query" grow>
                        <Combobox
                            options={options}
                            onChange={handleParameterChange}
                            value={parameter ?? null}
                            onInputChange={(v) => { debouncedFetchParameters(v, endpoint); }}
                            onOpenMenu={() => { debouncedFetchParameters('', endpoint); }}
                        />
                    </InlineField>

                    {isAggregate && (
                        <InlineField label="." grow>
                            <Input
                                marginWidth={0}
                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => { setAggregatePath(e.target.value); }}
                                placeholder='Path to value (case sensitive)'
                                value={aggregatePath}
                            />
                        </InlineField>
                    )}

                </Stack>

                <InlineField>
                    <Checkbox
                        value={isAggregate}
                        onChange={(e) => {
                            const newState = e.currentTarget.checked;
                            setIsAggregate(newState);
                            if (!newState) { setAggregatePath(''); }
                        }}
                        label='Aggregate'
                    />
                </InlineField>

            </Stack>

            {additionalFields && (
                <Stack direction="row">
                    <InlineField label="Additional fields" grow>
                        <MultiSelect
                            options={FieldsOptions}
                            onChange={(arr: Array<SelectableValue<QueryField>>) => { setFields(arr.map((v) => v.value).filter(Boolean) as QueryField[]); }}
                            values={fields}
                        />
                    </InlineField>
                </Stack>
            )}
        </>
    );
}
