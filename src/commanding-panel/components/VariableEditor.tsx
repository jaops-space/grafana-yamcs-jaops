import React from 'react';
import { ComboboxOption, Checkbox, Combobox, Field, Input } from '@grafana/ui';
import { VariableWithMultiSupport } from '@grafana/data';
import { getTemplateSrv } from '@grafana/runtime';
import InputModeField from './InputModeField';
import { CommandButton } from './CommandButton';
import { ButtonStyleFields } from './ButtonStyleFields';
import { FormSection } from './FormSection';
import { CommandInfo, DualButtonStates, UpdateFormOption } from '../types';

export function VariableEditor(props: {
    commandInfo: CommandInfo;
    index: number;
    commandState: any;
    scopedVars: any;
    loading: boolean;
    dualButtonStates: DualButtonStates;
    onOptionChange: UpdateFormOption;
    showPreview?: boolean;
}) {
    const {
        commandInfo,
        index,
        commandState,
        scopedVars,
        loading,
        dualButtonStates,
        onOptionChange,
        showPreview = true,
    } = props;
    const command = commandInfo.command;
    const set = (option: string, value: any) => onOptionChange(command.name, option, value, index);

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', width: '100%' }}>
            <FormSection title="Variable behavior" separated={false}>
                <Field label="Variable" description="Variable to change" style={{ marginBottom: 0 }}>
                    <Combobox
                        disabled={loading}
                        options={getTemplateSrv()
                            .getVariables()
                            .map((vr) => ({ label: vr.label || vr.name, value: vr.name }))}
                        value={commandState?.variableToSet || ''}
                        onChange={(e: ComboboxOption<string>) => set('variableToSet', e.value)}
                    />
                </Field>
                <Field label="Change Mode" description="How to change the value" style={{ marginBottom: 0 }}>
                    <Combobox
                        disabled={loading}
                        options={[
                            { label: 'Set', value: 'change', description: 'Set the variable to a value' },
                            { label: 'Add', value: 'add', description: 'Add a number to the variable' },
                            { label: 'Multiply', value: 'multiply', description: 'Multiply the variable by a number' },
                            {
                                label: 'Input',
                                value: 'input',
                                description: 'Text box that displays current value and accepts keyboard input',
                            },
                        ]}
                        value={commandState?.changeMode || ''}
                        onChange={(e: ComboboxOption<string>) => set('changeMode', e.value)}
                    />
                </Field>
                {commandState?.changeMode !== 'input' && (
                    <Field
                        label="Value"
                        description="Value to use. You may write a custom value."
                        style={{ marginBottom: 0 }}
                    >
                        <Combobox
                            disabled={loading}
                            options={(
                                (
                                    getTemplateSrv()
                                        .getVariables()
                                        .find(
                                            (vr) => vr.name === commandState?.variableToSet
                                        ) as VariableWithMultiSupport
                                )?.options || []
                            ).map((option) => ({ label: option.text as string, value: option.value as string }))}
                            value={commandState?.valueToSet || ''}
                            createCustomValue
                            onChange={(e: ComboboxOption<string>) => set('valueToSet', e.value)}
                        />
                    </Field>
                )}
                <Field label="Display variable name" description="Show name in runtime" style={{ marginBottom: 0 }}>
                    <Checkbox
                        value={commandState?.showVariableLabel !== false}
                        onChange={(e) => set('showVariableLabel', e.currentTarget.checked)}
                        label="Show variable name"
                    />
                </Field>
                <Field label="Comment" description="Optional comment" style={{ marginBottom: 0 }}>
                    <Input
                        type="text"
                        disabled={loading}
                        value={commandState?.comment || ''}
                        onChange={(e) => set('comment', e.currentTarget.value)}
                    />
                </Field>
                <Field label="Unit" description="Displayed next to the value" style={{ marginBottom: 0 }}>
                    <Input
                        type="text"
                        disabled={loading}
                        value={commandState?.unit || ''}
                        placeholder="m/s, deg, ..."
                        onChange={(e) => set('unit', e.currentTarget.value)}
                        style={{ width: '100%' }}
                    />
                </Field>
            </FormSection>

            {commandState?.changeMode !== 'input' && (
                <FormSection title="Button appearance">
                    <ButtonStyleFields
                        commandName={command.name}
                        index={index}
                        commandState={commandState}
                        loading={loading}
                        onOptionChange={onOptionChange}
                    />
                </FormSection>
            )}

            {showPreview && (
                <FormSection title="Preview">
                    <div
                        style={{
                            gridColumn: '1 / -1',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            minHeight: '44px',
                            width: '100%',
                            objectFit: 'contain',
                        }}
                    >
                        {commandState?.changeMode === 'input' ? (
                            <InputModeField
                                variableToSet={commandState?.variableToSet}
                                scopedVars={scopedVars}
                                loading={false}
                                unit={commandState?.unit}
                                showVariableLabel={commandState?.showVariableLabel}
                                color={commandState?.color}
                                textColor={commandState?.textColor}
                                size={commandState?.size}
                            />
                        ) : (
                            <CommandButton
                                commandInfo={commandInfo}
                                index={index}
                                commandState={commandState}
                                scopedVars={scopedVars}
                                loading={loading}
                                dualButtonStates={dualButtonStates}
                            />
                        )}
                    </div>
                </FormSection>
            )}
        </div>
    );
}
