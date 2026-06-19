import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Alert, Badge, Combobox, Field, Input } from '@grafana/ui';
import { UpdateArgument, ValidateArgument } from '../types';
import { coerceInputValue } from '../utils/commandArguments';

export function ArgumentField(props: {
    commandName: string;
    index: number;
    arg: any;
    value: any;
    error?: string;
    loading: boolean;
    onChange: UpdateArgument;
    onValidate?: ValidateArgument;
    labelPrefix?: string;
}) {
    const { commandName, index, arg, value, error, loading, onChange, onValidate, labelPrefix } = props;

    const setValue = (next: any) => {
        onChange(commandName, arg.name, next, index);
        onValidate?.(commandName, arg, next);
    };

    let inputField: React.ReactNode;

    if (arg.type.engType === 'enumeration') {
        inputField = (
            <Combobox
                disabled={loading}
                value={value}
                onChange={(e: SelectableValue<any> | null) => {
                    if (e?.value === null || e?.value === undefined) {
                        return;
                    }
                    setValue(e.value);
                }}
                options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))}
            />
        );
    } else if (arg.type.engType === 'boolean') {
        inputField = (
            <Combobox
                value={value !== undefined && value !== null ? String(value) : ''}
                disabled={loading}
                onChange={(e: SelectableValue<any> | null) => {
                    if (e?.value === null || e?.value === undefined) {
                        return;
                    }
                    setValue(e.value === 'true' ? true : e.value === 'false' ? false : e.value);
                }}
                options={[
                    { label: arg.type.zeroStringValue || 'False', value: 'false' },
                    { label: arg.type.oneStringValue || 'True', value: 'true' },
                ]}
            />
        );
    } else {
        inputField = (
            <Input
                disabled={loading}
                type="text"
                value={value}
                placeholder={
                    arg.type.engType === 'integer' || arg.type.engType === 'float'
                        ? 'Enter value or $variable'
                        : undefined
                }
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const rawValue = e.target.value;
                    setValue(coerceInputValue(rawValue, arg.type.engType));
                    onValidate?.(commandName, arg, rawValue);
                }}
                style={{ width: '100%' }}
            />
        );
    }

    return (
        <Field
            key={`${labelPrefix ?? ''}${arg.name}`}
            label={labelPrefix ? `${labelPrefix} - ${arg.name}` : arg.name}
            description={arg.description}
            style={{ width: '100%', marginBottom: 0 }}
        >
            <>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    {inputField}
                    <Badge
                        text={`${arg.type.rangeMin ? `${arg.type.rangeMin} ≤` : ''} ${arg.type.engType} ${arg.type.rangeMax ? `≤ ${arg.type.rangeMax}` : ''}`}
                        color="blue"
                    />
                </div>
                {error && (
                    <Alert title="Invalid argument" severity="error">
                        {error}
                    </Alert>
                )}
            </>
        </Field>
    );
}
