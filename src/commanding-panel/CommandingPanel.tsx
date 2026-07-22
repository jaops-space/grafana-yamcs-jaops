import React, { useState } from 'react';
import { CommandForms } from './types';
import { useLocationService } from '@grafana/runtime';
import { ButtonGroupPreview } from './components/ButtonGroupPreview';
import { CommandButton } from './components/CommandButton';
import { CommandCard } from './components/CommandCard';
import { LayoutEditor } from './components/LayoutEditor';
import { VariableRuntime } from './components/VariableRuntime';
import { useCommandInfos } from './hooks/useCommandInfos';
import { useCommandSubmit } from './hooks/useCommandSubmit';
import { useDatasource } from './hooks/useDatasource';
import { useDualButtonStates } from './hooks/useDualButtonStates';
import { useDualCommandInfos } from './hooks/useDualCommandInfos';
import { CommandErrors, CommandingPanelProps } from './types';
import { getCommandKey } from './utils/commandKeys';
import { getEditorCardsStyle, getRuntimeButtonWrapperStyle, getRuntimeLayoutStyle } from './utils/layout';
import { setArgumentError, validateCommandArgument } from './utils/validation';

export default function CommandingPanel({ variableMode = false, ...props }: CommandingPanelProps) {
    const { data, options, onOptionsChange } = props;
    const location = useLocationService().getLocation();
    const editing = location.search.includes('editPanel=');
    const scopedVars = props.data.request?.scopedVars;
    const datasourceUid = (data.request?.targets?.[0]?.datasource as any)?.uid;

    const datasource = useDatasource(datasourceUid);
    const { dualCommandInfos, fetchDualCommandInfo, clearDualCommandInfo } = useDualCommandInfos(datasource);
    const commandInfos = useCommandInfos({
        datasource,
        targets: data.request?.targets ?? [],
        scopedVars,
        variableMode,
        options,
        fetchDualCommandInfo,
    });

    const [formState, setFormState] = useState<CommandForms>(options.commandForms || {});
    const [errors, setErrors] = useState<CommandErrors>({});
    const [loading, setLoading] = useState(false);
    const { dualButtonStates, updateDualButtonStates } = useDualButtonStates(props.id, options, onOptionsChange);

    const hasGroupPreview = editing && commandInfos.length > 1;

    const updateLayoutOption = (key: string, value: any) => {
        onOptionsChange({
            ...options,
            [key]: value,
        });
    };

    const handleArgumentChange = (commandName: string, argName: string, value: any, index: number) => {
        setFormState((prevState) => {
            const commandKey = getCommandKey(commandName, index);
            const newState = {
                ...prevState,
                [commandKey]: {
                    ...prevState[commandKey],
                    arguments: {
                        ...prevState[commandKey]?.arguments,
                        [argName]: value,
                    },
                },
            };
            onOptionsChange({ ...options, commandForms: newState });
            return newState;
        });
    };

    const handleOptionChange = (commandName: string, option: string, value: any, index: number) => {
        setFormState((prevState) => {
            const commandKey = getCommandKey(commandName, index);
            const newState = {
                ...prevState,
                [commandKey]: {
                    ...prevState[commandKey],
                    [option]: value,
                },
            };
            onOptionsChange({ ...options, commandForms: newState });
            return newState;
        });
    };

    const validateArgument = (commandName: string, arg: any, value: any) => {
        setErrors((prev) => setArgumentError(prev, commandName, arg.name, validateCommandArgument(arg, value)));
    };

    const handleSubmit = useCommandSubmit({
        datasource,
        formState,
        scopedVars,
        variableMode,
        options,
        setLoading,
        dualCommandInfos,
        dualButtonStates,
        updateDualButtonStates,
    });

    const renderRuntimeButton = (commandInfo: any, index: number, preview = false) => {
        const command = commandInfo.command;
        const commandState = formState[getCommandKey(command.name, index)];

        if (variableMode) {
            return (
                <VariableRuntime
                    key={getCommandKey(command.name, index)}
                    commandInfo={commandInfo}
                    index={index}
                    commandState={commandState}
                    scopedVars={scopedVars}
                    loading={loading}
                    dualButtonStates={dualButtonStates}
                    onSubmit={preview ? () => undefined : handleSubmit}
                />
            );
        }

        return (
            <CommandButton
                key={getCommandKey(command.name, index)}
                commandInfo={commandInfo}
                index={index}
                commandState={commandState}
                scopedVars={scopedVars}
                loading={loading}
                dualButtonStates={dualButtonStates}
                onSubmit={preview ? undefined : handleSubmit}
            />
        );
    };

    return (
        <div
            data-testid={variableMode ? 'jaops-variable-setting-panel' : 'jaops-commanding-panel'}
            data-command-count={commandInfos.length}
            style={{ width: '100%', height: '100%', overflow: editing ? 'auto' : 'hidden' }}
        >
            {editing && commandInfos.length > 1 && (
                <>
                    <ButtonGroupPreview options={options}>
                        {commandInfos.map((commandInfo, index) => renderRuntimeButton(commandInfo, index, true))}
                    </ButtonGroupPreview>
                    <LayoutEditor options={options} onChange={updateLayoutOption} />
                </>
            )}

            <div style={editing ? getEditorCardsStyle() : getRuntimeLayoutStyle(options)}>
                {commandInfos.map((commandInfo, index) => {
                    const command = commandInfo.command;
                    const commandState = formState[getCommandKey(command.name, index)];
                    const commandKey = getCommandKey(command.name, index);

                    if (!editing) {
                        return (
                            <div key={commandKey} style={getRuntimeButtonWrapperStyle(options)}>
                                {renderRuntimeButton(commandInfo, index)}
                            </div>
                        );
                    }

                    return (
                        <CommandCard
                            key={commandKey}
                            commandInfo={commandInfo}
                            index={index}
                            commandState={commandState}
                            variableMode={variableMode}
                            scopedVars={scopedVars}
                            loading={loading}
                            datasource={datasource}
                            errors={errors}
                            dualCommandInfos={dualCommandInfos}
                            dualButtonStates={dualButtonStates}
                            onSubmit={handleSubmit}
                            onArgumentChange={handleArgumentChange}
                            onOptionChange={handleOptionChange}
                            onValidate={validateArgument}
                            fetchDualCommandInfo={fetchDualCommandInfo}
                            clearDualCommandInfo={clearDualCommandInfo}
                            showPreview={!hasGroupPreview}
                        />
                    );
                })}
            </div>
        </div>
    );
}
