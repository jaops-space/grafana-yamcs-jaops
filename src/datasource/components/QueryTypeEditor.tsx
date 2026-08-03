import { InlineField, Select, Stack, Tooltip } from '@grafana/ui';
import { QueryType } from '../types';
import React, { useEffect } from 'react';
import { QueryCategory, QueryEditorModelProps, QueryOptions } from './constants';
import { ParameterQuery } from './ParameterQuery';
import { QueryCategoryBadge } from './QueryCategoryBadge';
import { queryHelp } from './queryHelp';

export function QueryTypeEditor(props: QueryEditorModelProps) {
    const { query, onChange, datasource } = props;
    const queryEditorModelProps: QueryEditorModelProps = { query, onChange, datasource };

    const setQueryType = (type: QueryType) => {
        onChange({ ...query, type });
    };

    // Default to PLOT if no type is set
    useEffect(() => {
        if (!query.type) {
            onChange({ ...query, type: QueryType.PLOT });
        }
    }, []);

    const queryTypeInfo = QueryOptions.find((o) => o.value === (query.type ?? QueryType.PLOT));
    const queryOptions = datasource.debugMode
        ? QueryOptions
        : QueryOptions.filter((o) => o.category !== QueryCategory.DEBUG);
    const selectedQueryTypeOption = queryOptions.find((o) => o.value === (query.type ?? QueryType.PLOT));
    const getTooltip = (type: QueryType, fallback?: string) => {
        const help = queryHelp[type];
        if (!help) {
            return fallback ?? '';
        }

        return (
            <>
                {help.tooltip}
                <br />
                Recommended panel: {help.recommendedPanel}
            </>
        );
    };

    return (
        <>
            <Stack direction="row" alignItems="center">
                <InlineField label="Query Type" grow>
                    {/* eslint-disable-next-line @typescript-eslint/no-deprecated */}
                    <Select
                        onChange={(s) => setQueryType((s.value as QueryType) ?? QueryType.PLOT)}
                        value={selectedQueryTypeOption}
                        isClearable={false}
                        options={queryOptions}
                        getOptionLabel={(value: any) => value.label ?? ''}
                        formatOptionLabel={(value: any) => (
                            <Stack direction="row" justifyContent="space-between">
                                <Tooltip content={getTooltip(value.value as QueryType, value.description)}>
                                    <span>{value.label}</span>
                                </Tooltip>
                                <span style={{ zIndex: 212 }}>{<QueryCategoryBadge category={value.category} />}</span>
                            </Stack>
                        )}
                        data-testid="jaops-query-type-select"
                    />
                </InlineField>
            </Stack>
            {(queryTypeInfo?.category === QueryCategory.PARAMETER ||
                queryTypeInfo?.category === QueryCategory.IMAGE) && <ParameterQuery {...queryEditorModelProps} />}
        </>
    );
}
