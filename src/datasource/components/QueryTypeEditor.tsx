// eslint-disable-next-line no-restricted-imports
import { Badge, InlineField, Select, Stack } from '@grafana/ui';
import { QueryType } from '../types';
import React, { useEffect } from 'react';
import { QueryCategory, QueryEditorModelProps, QueryOptions } from './constants';
import { ParameterQuery } from './ParameterQuery';

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
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const queryTypeInfo = QueryOptions.find((o) => o.value === (query.type ?? QueryType.PLOT));
    const queryOptions = datasource.debugMode
        ? QueryOptions
        : QueryOptions.filter((o) => o.category !== QueryCategory.DEBUG);
    const selectedQueryTypeOption = queryOptions.find((o) => o.value === (query.type ?? QueryType.PLOT));

    function getBadgeCategory(category: any): React.ReactNode {
        switch (category) {
            case QueryCategory.PARAMETER:
                return <Badge color="blue" text="Parameter" />;
            case QueryCategory.TIMELINE:
                return <Badge color="purple" text="Timeline" />;
            case QueryCategory.IMAGE:
                return <Badge color="green" text="Image" />;
            case QueryCategory.COMMANDING:
                return <Badge color="red" text="Commanding" />;
            case QueryCategory.ALARMS:
                return <Badge color="orange" text="Alarm" />;
            case QueryCategory.LINKS:
                return <Badge color="blue" text="Links" />;
            case QueryCategory.DEBUG:
                return <Badge color="orange" text="Debug" />;
            default:
                return <Badge color="red" text="Unknown" />;
        }
    }

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
                                <span>{value.label}</span>
                                <span style={{ zIndex: 212 }}>{getBadgeCategory(value.category)}</span>
                            </Stack>
                        )}
                        data-testid="select"
                    />
                </InlineField>
            </Stack>
            {(queryTypeInfo?.category === QueryCategory.PARAMETER || queryTypeInfo?.category === QueryCategory.IMAGE) && (
                <ParameterQuery {...queryEditorModelProps} />
            )}
        </>
    );
}
