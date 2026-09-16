import { css } from '@emotion/css';
import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import React from 'react';
import { Combobox, Field, Input, useStyles2 } from '@grafana/ui';
import { DataSourceWithBackend } from '@grafana/runtime';
import { ArgumentField } from './ArgumentField';
import { ButtonStyleFields } from './ButtonStyleFields';
import { FormSection } from './FormSection';
import { DualCommandInfos, UpdateFormOption } from '../types';
import { getCommandKey, getDualInfoKey } from '../utils/commandKeys';

const getStyles = (theme: GrafanaTheme2) => ({
    side: css({
        border: `1px solid ${theme.colors.border.weak}`,
        borderRadius: theme.shape.radius.default,
        padding: theme.spacing(1.25),
        minWidth: 0,
    }),
    grid: css({
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
        gap: theme.spacing(1.5),
        width: '100%',
    }),
});

function DualSideConfig(props: {
    side: 'on' | 'off';
    title: string;
    labelPrefix: string;
    command: any;
    commandInfo: any;
    index: number;
    endpoint: string;
    datasource: DataSourceWithBackend | null;
    commandState: any;
    dualCommandInfos: DualCommandInfos;
    loading: boolean;
    onOptionChange: UpdateFormOption;
    fetchDualCommandInfo: (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => void;
    clearDualCommandInfo: (commandKey: string, side: 'on' | 'off') => void;
}) {
    const {
        side,
        title,
        labelPrefix,
        command,
        commandInfo,
        index,
        endpoint,
        datasource,
        commandState,
        dualCommandInfos,
        loading,
        onOptionChange,
        fetchDualCommandInfo,
        clearDualCommandInfo,
    } = props;
    const stateKey = side === 'on' ? 'onCommand' : 'offCommand';
    const commandKey = getCommandKey(command.name, index);
    const sideCommandInfo = dualCommandInfos[getDualInfoKey(commandKey, side)] ?? command;
    const sideState = commandState?.[stateKey] ?? {};
    const styles = useStyles2(getStyles);

    const updateSide = (patch: Record<string, any>) => {
        onOptionChange(command.name, stateKey, { ...sideState, ...patch }, index);
    };
    const setSideField: UpdateFormOption = (_commandName: string, option: string, value: any) =>
        updateSide({ [option]: value });

    const updateSideArgument = (_commandName: string, argName: string, value: any) => {
        updateSide({ arguments: { ...sideState.arguments, [argName]: value } });
    };

    return (
        <div className={styles.side}>
            <FormSection title={title} separated={false}>
                <Field
                    label={`${labelPrefix} Command`}
                    description="Leave empty to use the default command"
                    style={{ marginBottom: 0 }}
                >
                    <Combobox
                        key={`${side}-cmd-${commandKey}`}
                        options={async (q: string) => {
                            if (!endpoint || !datasource) {
                                return [];
                            }
                            const results: Array<{ name: string; description: string }> = await datasource.getResource(
                                `endpoint/${endpoint}/commands`,
                                q ? { q } : undefined
                            );
                            return results.map((c) => ({ label: c.name, value: c.name, description: c.description }));
                        }}
                        value={sideState.commandName ?? null}
                        isClearable
                        onChange={(e: SelectableValue<string> | null) => {
                            const name = e?.value ?? '';
                            updateSide({ commandName: name, arguments: {} });
                            if (name) {
                                fetchDualCommandInfo(commandKey, side, name, commandInfo.endpoint);
                            } else {
                                clearDualCommandInfo(commandKey, side);
                            }
                        }}
                    />
                </Field>

                {sideCommandInfo.argument?.map((arg: any) => (
                    <ArgumentField
                        key={`${side}-${arg.name}`}
                        commandName={command.name}
                        index={index}
                        arg={arg}
                        value={sideState.arguments?.[arg.name] ?? arg.initialValue}
                        loading={loading}
                        onChange={updateSideArgument}
                        labelPrefix={labelPrefix}
                    />
                ))}

                <Field label={`${labelPrefix} Label`} description="Supports $variable" style={{ marginBottom: 0 }}>
                    <Input
                        type="text"
                        disabled={loading}
                        value={sideState.label ?? ''}
                        onChange={(e) => updateSide({ label: e.currentTarget.value })}
                        style={{ width: '100%' }}
                    />
                </Field>
                <Field label={`${labelPrefix} Tooltip`} description="Supports $variable" style={{ marginBottom: 0 }}>
                    <Input
                        type="text"
                        disabled={loading}
                        value={sideState.tooltip ?? ''}
                        onChange={(e) => updateSide({ tooltip: e.currentTarget.value })}
                        style={{ width: '100%' }}
                    />
                </Field>
                <Field
                    label={`${labelPrefix} Comment`}
                    description="Optional issue comment"
                    style={{ marginBottom: 0 }}
                >
                    <Input
                        type="text"
                        disabled={loading}
                        value={sideState.comment || ''}
                        onChange={(e) => updateSide({ comment: e.currentTarget.value })}
                        style={{ width: '100%' }}
                    />
                </Field>
            </FormSection>

            <FormSection title={`${labelPrefix} appearance`}>
                <ButtonStyleFields
                    commandName={command.name}
                    index={index}
                    commandState={sideState}
                    loading={loading}
                    onOptionChange={setSideField}
                />
            </FormSection>
        </div>
    );
}

export function DualButtonConfig(
    props: Omit<React.ComponentProps<typeof DualSideConfig>, 'side' | 'title' | 'labelPrefix'>
) {
    const styles = useStyles2(getStyles);

    return (
        <div className={styles.grid}>
            <DualSideConfig {...props} side="on" title="Left button" labelPrefix="LEFT" />
            <DualSideConfig {...props} side="off" title="Right button" labelPrefix="RIGHT" />
        </div>
    );
}
