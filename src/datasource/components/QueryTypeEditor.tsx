import { Combobox, ComboboxOption, InlineField, Stack } from '@grafana/ui';
import { QueryType } from '../types';
import React from 'react';
import { QueryCategory, QueryOptions, QueryProps } from './constants';
import { ParameterQuery } from './ParameterQuery';
import { CommandQuery } from './CommandQuery';

export function QueryTypeEditor(props: QueryProps) {

    const { query, onChange, datasource } = props;

    const setQueryType = (type: QueryType) => {
        onChange({ ...query, type });
    };

    const queryTypeInfo = QueryOptions.find((o) => o.value === query.type);
    const queryOptions = datasource.debugMode ? QueryOptions : QueryOptions.filter((o) => o.category !== QueryCategory.DEBUG);

    return (
        <>
            <Stack direction="row" alignItems="center">
                <InlineField label="Query Type" grow>
                    <Combobox
                        onChange={(s: ComboboxOption<string> | null) => { setQueryType((s?.value ?? QueryType.PLOT) as QueryType); }}
                        value={query.type ?? null}
                        options={queryOptions.map((o) => ({ label: o.label ?? '', value: String(o.value ?? '') }))}
                        data-testid='select'
                    />
                </InlineField>
            </Stack>
            {(queryTypeInfo?.category === QueryCategory.PARAMETER ||
                queryTypeInfo?.category === QueryCategory.IMAGE) &&
                <ParameterQuery {...props} />
            }
            {queryTypeInfo?.value === QueryType.COMMANDING && <CommandQuery {...props} />}
        </>
    );
}
