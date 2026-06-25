import { SelectableValue } from '@grafana/data';
import { Combobox, ComboboxOption, Checkbox, InlineField, Input, MultiSelect, Stack } from '@grafana/ui';
import React, { useMemo } from 'react';
import { QueryField } from '../types';
import { FieldsOptions, QueryOptions, QueryProps } from './constants';

export function ParameterQuery({ query, onChange, datasource }: QueryProps) {
    const { endpoint } = query;

    const queryTypeInfo = QueryOptions.find((o) => o.value === query.type);
    const additionalFields = queryTypeInfo?.additionalFields;

    const selectedFields = useMemo(
        () => FieldsOptions.filter((opt) => query.fields?.includes(opt.value as QueryField)),
        [query.fields]
    );

    const updateQuery = (patch: Partial<typeof query>) => {
        onChange({
            ...query,
            ...patch,
        });
    };

    const handleParameterChange = (v: ComboboxOption | null) => {
        updateQuery({ parameter: (v?.value as string) ?? '' });
    };

    const isAggregate = Boolean(query.aggregatePath);

    const fetchOptions = async (inputValue: string): Promise<ComboboxOption[]> => {
        if (!endpoint) {
            return [];
        }
        const parameters: string[] = await datasource.getResource(
            `endpoint/${endpoint}/parameters`,
            inputValue ? { q: inputValue } : undefined
        );
        return parameters.map((p) => ({ label: p, value: p }));
    };

    return (
        <>
            <Stack direction="row" alignItems="center">
                <Stack direction="row" alignItems="center" gap={0} grow={1}>
                    <InlineField label="Parameter to query" grow>
                        <Combobox
                            key={`parameter-select-${endpoint ?? 'none'}`}
                            options={fetchOptions}
                            onChange={handleParameterChange}
                            value={query.parameter ?? null}
                        />
                    </InlineField>

                    {isAggregate && (
                        <InlineField label="." grow>
                            <Input
                                marginWidth={0}
                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                    updateQuery({ aggregatePath: e.target.value });
                                }}
                                placeholder="Path to value (case sensitive)"
                                value={query.aggregatePath || ''}
                            />
                        </InlineField>
                    )}
                </Stack>

                <InlineField>
                    <Checkbox
                        checked={isAggregate}
                        onChange={(e) => {
                            const newState = e.currentTarget.checked;
                            updateQuery({ aggregatePath: newState ? query.aggregatePath || '.' : '' });
                        }}
                        label="Aggregate"
                    />
                </InlineField>
            </Stack>

            {additionalFields && (
                <Stack direction="row">
                    <InlineField label="Additional fields" grow>
                        <MultiSelect
                            options={FieldsOptions}
                            onChange={(arr: Array<SelectableValue<QueryField>>) => {
                                updateQuery({ fields: arr.map((v) => v.value).filter(Boolean) as QueryField[] });
                            }}
                            value={selectedFields}
                        />
                    </InlineField>
                </Stack>
            )}
        </>
    );
}
