import { SelectableValue } from '@grafana/data';
import { Combobox, ComboboxOption, InlineField, MultiCombobox, RadioButtonGroup, Stack, Text } from '@grafana/ui';
import React, { useCallback } from 'react';
import { Query, QueryField, QueryType } from '../types';
import { FieldsOptions, QueryEditorModelProps, QueryOptions } from './constants';

const ColorOptions: Array<SelectableValue<boolean>> = [
    { label: 'None', value: false },
    { label: 'Automatic', value: true },
];

export function ParameterQuery({ query, onChange, datasource }: QueryEditorModelProps) {
    const { endpoint } = query;

    const queryTypeInfo = QueryOptions.find((o) => o.value === query.type);
    const additionalFields = queryTypeInfo?.additionalFields;

    const selectedFields = FieldsOptions.filter((opt) => query.fields?.includes(opt.value as QueryField));

    const updateQuery = useCallback(
        (patch: Partial<Query>) => {
            onChange({
                ...query,
                ...patch,
            });
        },
        [onChange, query]
    );

    const handleParameterChange = useCallback(
        (v: ComboboxOption | null) => {
            updateQuery({ parameter: (v?.value as string) ?? '' });
        },
        [updateQuery]
    );

    const fetchOptions = useCallback(
        async (inputValue: string): Promise<ComboboxOption[]> => {
            if (!endpoint) {
                return [];
            }
            const parameters: string[] = await datasource.getResource(
                `endpoint/${endpoint}/parameters`,
                inputValue ? { q: inputValue } : undefined
            );
            return parameters.map((p) => ({ label: p, value: p }));
        },
        [datasource, endpoint]
    );

    const tooltip = <>
        To query aggregate values, you might directly input the aggregate formula, e.g:
        <br/>
        <Text variant='code'>/myproject/Position.x</Text><br/>or<br/><Text variant='code'>/myproject/Aggregate[0].member</Text>

        <br/><br/>
        Alarm thresholds are provided automatically, to be able to visualize them (on a Time Series panel for example), navigate to <b>Thresholds</b> on panel options on the right, and modify <b>Show Thresholds</b>, you might need to set a manual minimum and maximum for your panel to view them. 
    </>

    return (
        <>
            <Stack direction="row" alignItems="center">
                <Stack direction="row" alignItems="center" gap={0} grow={1}>
                    <InlineField label="Parameter to query" 
                        tooltip={tooltip}
                    grow>
                        <Combobox
                            key={`parameter-select-${endpoint ?? 'none'}`}
                            options={fetchOptions}
                            onChange={handleParameterChange}
                            value={query.parameter ?? null}
                            createCustomValue
                            customValueDescription="Use custom parameter expression"
                            data-testid="jaops-parameter-select"
                        />
                    </InlineField>
                </Stack>
            </Stack>

            {additionalFields && (
                <Stack direction="row">
                    <InlineField label="Additional fields" grow>
                        <MultiCombobox
                            options={FieldsOptions}
                            onChange={(arr: Array<ComboboxOption<QueryField>>) => {
                                updateQuery({ fields: arr.map((v) => v.value).filter(Boolean) as QueryField[] });
                            }}
                            value={selectedFields}
                        />
                    </InlineField>
                </Stack>
            )}

            {query.type === QueryType.DISCRETE && (
                <Stack direction="row">
                    <InlineField label="Value colors"
                    tooltip="Automatically set value mappings with generated colors">
                        <RadioButtonGroup
                            options={ColorOptions}
                            value={Boolean(query.automaticColors)}
                            onChange={(value) => updateQuery({ automaticColors: value })}
                            data-testid="jaops-discrete-automatic-colors"
                        />
                    </InlineField>
                </Stack>
            )}
        </>
    );
}
