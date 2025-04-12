import { Badge, InlineField, Select, Stack } from '@grafana/ui';
import { QueryType } from '../types';
import React from 'react';
import { QueryCategory, QueryOptions, QueryProps } from './constants';
import { ParameterQuery } from './ParameterQuery';
import { CommandQuery } from './CommandQuery';

export function QueryTypeEditor(props: QueryProps) {

    const { query, onChange, datasource } = props;

    const setQueryType = (type: QueryType) => {
        onChange({
            ...query,
            type,
        });
    }

    const queryTypeInfo = QueryOptions.find((o) => o.value === query.type);
    let queryOptions = datasource.debugMode ? QueryOptions : QueryOptions.filter((o) => o.category !== QueryCategory.DEBUG);

    function getBadgeCategory(category: any): React.ReactNode {
        switch(category) {
            case QueryCategory.PARAMETER:
                return <Badge color="blue" text="Parameter" />;
            case QueryCategory.EVENT:
                return <Badge color="purple" text="Event" />;
            case QueryCategory.IMAGE:
                return <Badge color="green" text="Image" />;
            case QueryCategory.COMMANDING:
                    return <Badge color='red' text='Commanding' />;
            case QueryCategory.DEBUG:
                 return <Badge color='orange' text='Debug' />;
            default:
                return <Badge color="red" text="Unknown" />;
        }
    }

    return (
        <>
            <Stack direction="row" alignItems="center">
                <InlineField label="Query Type" grow>
                    <Select
                        onChange={(s) => setQueryType(s.value ?? QueryType.PLOT)}
                        value={query.type}
                        isClearable={false}
                        defaultValue={query.type}
                        options={queryOptions}
                        getOptionLabel={(value: any) => (
                            <Stack direction={'row'} justifyContent={'space-between'}>
                                <span>{value.label}</span>
                                <span style={{ zIndex: 212 }}>{getBadgeCategory(value.category)}</span>
                            </Stack>
                        )}
                        data-testid='select'
                    />
                </InlineField>
            </Stack>
            {
                (queryTypeInfo?.category === QueryCategory.PARAMETER || 
                queryTypeInfo?.category === QueryCategory.IMAGE) && 
                <ParameterQuery {...props} />
            }
            {queryTypeInfo?.category === QueryCategory.COMMANDING && <CommandQuery {...props} />}
        </>
    );
}
