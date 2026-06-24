import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Combobox, Field } from '@grafana/ui';
import { CommandButton } from './CommandButton';
import { ArgumentField } from './ArgumentField';
import { ButtonStyleFields } from './ButtonStyleFields';
import { DualButtonConfig } from './DualButtonConfig';
import { FormSection } from './FormSection';
import { CommandSelector } from './CommandSelector';
import {
    CommandErrors,
    CommandInfo,
    DualButtonStates,
    DualCommandInfos,
    UpdateArgument,
    UpdateFormOption,
    ValidateArgument,
} from '../types';

export function CommandEditor(props: {
    commandInfo: CommandInfo;
    index: number;
    commandState: any;
    scopedVars: any;
    loading: boolean;
    datasource: any;
    errors: CommandErrors;
    dualCommandInfos: DualCommandInfos;
    dualButtonStates: DualButtonStates;
    onArgumentChange: UpdateArgument;
    onOptionChange: UpdateFormOption;
    onValidate: ValidateArgument;
    fetchDualCommandInfo: (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => void;
    clearDualCommandInfo: (commandKey: string, side: 'on' | 'off') => void;
    showPreview?: boolean;
}) {
    const {
        commandInfo,
        index,
        commandState,
        scopedVars,
        loading,
        datasource,
        errors,
        dualCommandInfos,
        dualButtonStates,
        onArgumentChange,
        onOptionChange,
        onValidate,
        fetchDualCommandInfo,
        clearDualCommandInfo,
        showPreview = true,
    } = props;
    const command = commandInfo.command;
    const set = (option: string, value: any) => onOptionChange(command.name, option, value, index);
    const isDualButton = commandState?.isDualButton === true;

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', width: '100%' }}>
            <FormSection separated={false}>
                <CommandSelector
                    label={isDualButton ? 'Default command' : 'Command'}
                    description={
                        isDualButton
                            ? 'Used when a side command is left empty'
                            : 'Select the command issued by this button'
                    }
                    endpoint={commandInfo.endpoint}
                    datasource={datasource}
                    value={commandState?.commandName ?? null}
                    disabled={loading}
                    commandInfo={command}
                    onChange={(name) => set('commandName', name)}
                />
            </FormSection>

            <FormSection separated={false}>
                <Field
                    label="Button type"
                    description="Use one command or a left/right split button"
                    style={{ marginBottom: 0 }}
                >
                    <Combobox
                        disabled={loading}
                        options={[
                            { label: 'Single Button', value: 'false' },
                            { label: 'Dual Button', value: 'true' },
                        ]}
                        value={isDualButton ? 'true' : 'false'}
                        onChange={(e: SelectableValue<string> | null) => {
                            if (e?.value === null || e?.value === undefined) {
                                return;
                            }
                            set('isDualButton', e.value === 'true');
                        }}
                    />
                </Field>
            </FormSection>

            {!isDualButton && (
                <>
                    {!!command.argument?.length && (
                        <FormSection title="Arguments">
                            {command.argument.map((arg: any) => (
                                <ArgumentField
                                    key={arg.name}
                                    commandName={command.name}
                                    index={index}
                                    arg={arg}
                                    value={commandState?.arguments?.[arg.name] ?? arg.initialValue}
                                    error={errors[command.name]?.[arg.name]}
                                    loading={loading}
                                    onChange={onArgumentChange}
                                    onValidate={onValidate}
                                />
                            ))}
                        </FormSection>
                    )}

                    <FormSection title="Button appearance">
                        <ButtonStyleFields
                            commandName={command.name}
                            index={index}
                            commandState={commandState}
                            loading={loading}
                            onOptionChange={onOptionChange}
                        />
                    </FormSection>
                </>
            )}

            {isDualButton && (
                <DualButtonConfig
                    command={command}
                    commandInfo={commandInfo}
                    index={index}
                    endpoint={commandInfo.endpoint}
                    datasource={datasource}
                    commandState={commandState}
                    dualCommandInfos={dualCommandInfos}
                    loading={loading}
                    onOptionChange={onOptionChange}
                    fetchDualCommandInfo={fetchDualCommandInfo}
                    clearDualCommandInfo={clearDualCommandInfo}
                />
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
                        <CommandButton
                            commandInfo={commandInfo}
                            index={index}
                            commandState={commandState}
                            scopedVars={scopedVars}
                            loading={loading}
                            dualButtonStates={dualButtonStates}
                        />
                    </div>
                </FormSection>
            )}
        </div>
    );
}
