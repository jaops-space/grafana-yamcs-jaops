import { AppEvents, PanelProps, SelectableValue, VariableWithMultiSupport } from '@grafana/data';
import { DataSourceWithBackend, getAppEvents, getDataSourceSrv, getTemplateSrv, locationService, useLocationService } from '@grafana/runtime';
import { Alert, Badge, Button, Card, Checkbox, ColorPickerInput, Combobox, Divider, Field, FieldSet, FileUpload, getAvailableIcons, Input, LoadingPlaceholder } from '@grafana/ui';
import { CommandForms, PanelOptions } from 'commanding-panel/types';
import React, { useState, useEffect, useRef } from 'react';
import Shapes from './Shapes';

type CommandInfos = Array<{
    command: any,
    endpoint: string
}>

export interface CommandingPanelProps extends PanelProps<PanelOptions> {
    variableMode?: boolean;
}

// Component for input mode that displays current value and accepts keyboard input
function InputModeField({ variableToSet, scopedVars, loading, unit, showVariableLabel, color, textColor, size }: { variableToSet?: string, scopedVars?: any, loading: boolean, unit?: string, showVariableLabel?: boolean, color?: string, textColor?: string, size?: string }) {
    // Subscribe reactively to location changes to get notified on every variable update
    const locService = useLocationService();
    const [locationTick, setLocationTick] = useState(0);

    useEffect(() => {
        const subscription = locService.getLocationObservable().subscribe(() => {
            setLocationTick(n => n + 1);
        });
        return () => subscription.unsubscribe();
    }, [locService]);

    // Read the variable value directly from the URL search params — these are updated synchronously
    // with every locationService.partial() call, so they always reflect the latest value immediately.
    const currentVariableValue = variableToSet
        ? (() => {
            const search = locService.getSearch();
            const fromUrl = search.get(`var-${variableToSet}`);
            if (fromUrl !== null) {
                return fromUrl;
            }
            // Fallback to template service (for initial load before any URL param is set)
            return getTemplateSrv().replace("$" + variableToSet);
        })()
        : '';

    // Get the variable's display label from dashboard settings (label takes priority over name)
    const variableDisplayLabel = variableToSet
        ? (() => {
            const variable = getTemplateSrv().getVariables().find(vr => vr.name === variableToSet);
            return variable ? (variable.label || variable.name) : variableToSet;
        })()
        : '';

    const [inputValue, setInputValue] = useState<string>(currentVariableValue);
    const isFocused = useRef(false);
    const lastSubmitted = useRef<string | null>(null);

    // Sync the input box with the live variable value on every location change,
    // as long as the user is not actively typing.
    useEffect(() => {
        if (!isFocused.current) {
            setInputValue(currentVariableValue);
            lastSubmitted.current = currentVariableValue;
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [locationTick]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);
    };

    const handleSubmit = (value: string) => {
        if (variableToSet) {
            lastSubmitted.current = value;
            locationService.partial({
                [`var-${variableToSet}`]: value,
                replace: true
            });
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            handleSubmit(inputValue);
            (e.target as HTMLInputElement).blur();
        }
    };

    const handleBlur = () => {
        isFocused.current = false;
        if (inputValue !== lastSubmitted.current) {
            handleSubmit(inputValue);
        }
    };

    const handleFocus = () => {
        isFocused.current = true;
    };

    const fontSizeMap: { [key: string]: string } = {
        xs: '10px',
        sm: '12px',
        md: '14px',
        lg: '18px',
    };

    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', overflow: 'hidden' }}>
            {showVariableLabel !== false && variableDisplayLabel && (
                <span
                    title={variableDisplayLabel}
                    style={{ whiteSpace: 'nowrap', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', flexShrink: 0, maxWidth: '40%' }}
                >{variableDisplayLabel}</span>
            )}
            <Input
                type="text"
                disabled={loading}
                value={inputValue}
                placeholder="Enter value"
                onChange={handleChange}
                onKeyDown={handleKeyDown}
                onBlur={handleBlur}
                onFocus={handleFocus}
                style={{
                    flex: 1,
                    minWidth: 0,
                    height: '100%',
                    backgroundColor: color || undefined,
                    color: textColor || undefined,
                    fontSize: size ? fontSizeMap[size] : undefined,
                }}
            />
            {unit && <span style={{ whiteSpace: 'nowrap' }}>{unit}</span>}
        </div>
    );
}

export default function CommandingPanel({ variableMode = false, ...props }: CommandingPanelProps) {

    const { data, options, onOptionsChange } = props;
    const locService = useLocationService();
    const location = locService.getLocation();
    const editing = location.search.includes('editPanel=');
    const scopedVars = props.data.request?.scopedVars;

    // Get a live datasource instance (needed for getResource / postResource)
    const [datasource, setDatasource] = useState<DataSourceWithBackend | null>(null);
    useEffect(() => {
        const uid = (data.request?.targets?.[0]?.datasource as any)?.uid;
        if (!uid) { return; }
        getDataSourceSrv().get(uid).then((ds) => {
            setDatasource(ds as DataSourceWithBackend);
        }).catch(console.error);
    // re-run only when the datasource UID changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [(data.request?.targets?.[0]?.datasource as any)?.uid]);

    // For commanding (non-variable) mode, fetch command info via resource call instead of streaming.
    const [commandInfos, setCommandInfos] = useState<CommandInfos>([]);
    useEffect(() => {
        if (variableMode) {
            setCommandInfos([{ command: {}, endpoint: "" }]);
            return;
        }
        if (!datasource) { return; }

        const targets = data.request?.targets ?? [];
        if (targets.length === 0) { return; }

        Promise.all(
            targets.map(async (target: any) => {
                const endpoint: string = target.asVariable
                    ? getTemplateSrv().replace(target.endpointVariable, scopedVars)
                    : target.endpoint;
                const command: string = getTemplateSrv().replace(target.command, scopedVars);
                if (!endpoint || !command) { return null; }
                try {
                    const info = await datasource.getResource(
                        `endpoint/${endpoint}/command/info`,
                        { name: command }
                    );
                    return { command: info, endpoint };
                } catch (err) {
                    console.error('Failed to fetch command info', err);
                    return null;
                }
            })
        ).then((results) => {
            const infos = results.filter(Boolean) as CommandInfos;
            setCommandInfos(infos);

            // Re-hydrate per-button command infos for any dual buttons that have saved commandName overrides
            infos.forEach((info, idx) => {
                const cmdKey = (info.command.name ?? '') + idx;
                const savedState = (options.commandForms ?? {})[cmdKey];
                if (savedState?.isDualButton) {
                    if (savedState.onCommand?.commandName) {
                        fetchDualCommandInfo(cmdKey, 'on', savedState.onCommand.commandName, info.endpoint);
                    }
                    if (savedState.offCommand?.commandName) {
                        fetchDualCommandInfo(cmdKey, 'off', savedState.offCommand.commandName, info.endpoint);
                    }
                }
            });
        });
    }, [datasource, data.request?.targets, variableMode]);

    // Per-button command info for dual button mode: keyed by "commandKey-on" / "commandKey-off"
    // This allows each side of a dual button to display arguments for its own selected command.
    const [dualCommandInfos, setDualCommandInfos] = useState<{ [key: string]: any }>({});

    const fetchDualCommandInfo = async (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => {
        if (!datasource || !commandName || !endpoint) { return; }
        try {
            const info = await datasource.getResource(
                `endpoint/${endpoint}/command/info`,
                { name: commandName }
            );
            setDualCommandInfos(prev => ({ ...prev, [`${commandKey}-${side}`]: info }));
        } catch (err) {
            console.error('Failed to fetch dual command info', err);
        }
    };

    const [formState, setFormState] = useState<CommandForms>(options.commandForms || {});
    const [errors, setErrors] = useState<{ [command: string]: { [arg: string]: string } }>({});
    const [loading, setLoading] = useState<boolean>(false);
    
    // State to track which button (on/off) was last clicked
    // Use localStorage to persist state across refreshes without saving dashboard
    const storageKey = `commanding-panel-state-${props.id}`;
    const [dualButtonStates, setDualButtonStates] = useState<{ [key: string]: 'on' | 'off' }>(() => {
        try {
            const stored = localStorage.getItem(storageKey);
            return stored ? JSON.parse(stored) : options.dualButtonStates || {};
        } catch {
            return options.dualButtonStates || {};
        }
    });

    const updateDualButtonStates = (newStates: { [key: string]: 'on' | 'off' }) => {
        setDualButtonStates(newStates);
        localStorage.setItem(storageKey, JSON.stringify(newStates));
        onOptionsChange({ ...options, dualButtonStates: newStates });
    };

    const handleInputChange = (commandName: string, argName: string, value: any, i: number) => {
        setFormState(prevState => {
            const newState = {
                ...prevState,
                [commandName + i]: {
                    ...prevState[commandName + i],
                    arguments: {
                        ...prevState[commandName + i]?.arguments,
                        [argName]: value
                    }
                },
            };
            onOptionsChange({ ...options, commandForms: newState });
            return newState;
        });
    };

    const handleOptionChange = (commandName: string, option: string, value: any, i: number) => {
        setFormState(prevState => {
            const newState = {
                ...prevState,
                [commandName + i]: {
                    ...prevState[commandName + i],
                    [option]: value
                },
            };
            onOptionsChange({ ...options, commandForms: newState });
            return newState;
        });
    };

    const validateInput = (commandName: string, arg: any, value: any) => {
        let error = '';

        // Skip validation if the value contains a variable reference
        if (typeof value === 'string' && (value.includes('$') || value.includes('{'))) {
            // Clear any existing error for this field
            setErrors(prev => ({
                ...prev,
                [commandName]: {
                    ...prev[commandName],
                    [arg.name]: '',
                },
            }));
            return;
        }

        if (arg.type.engType === 'integer' || arg.type.engType === 'float') {
            const numValue = parseFloat(value);
            if (isNaN(numValue) || (arg.type.rangeMin && numValue < arg.type.rangeMin) || (arg.type.rangeMax && numValue > arg.type.rangeMax)) {
                error = `Must be between ${arg.type.rangeMin} and ${arg.type.rangeMax}`;
                if (parseInt(value, 10) !== numValue && arg.type.engType === 'integer') {
                    error = `Must be a whole number`;
                }
                if (!arg.type.rangeMin) {
                    error = `Must be less than ${arg.type.rangeMax}`;
                }
                if (!arg.type.rangeMax) {
                    error = `Must be greater than ${arg.type.rangeMin}`;
                }
            }
        } else if (arg.type.engType === 'string') {
            if ((arg.type.minChars && value.length < arg.type.minChars) || (arg.type.maxChars && value.length > arg.type.maxChars)) {
                error = `Length must be between ${arg.type.minChars} and ${arg.type.maxChars} characters`;
                if (!arg.type.minChars) {
                    error = `Length must be less than ${arg.type.maxChars} characters`;
                }
                if (!arg.type.maxChars) {
                    error = `Length must be greater than ${arg.type.minChars} characters`;
                }
            }
        }
        setErrors(prev => ({
            ...prev,
            [commandName]: {
                ...prev[commandName],
                [arg.name]: error,
            },
        }));
    };

    const appEvents = getAppEvents();

    // isOffCommand parameter handles dual button submissions
    const handleSubmit = (commandInfo: CommandInfos[number], i: number, isOffCommand = false) => {
        const command = commandInfo.command;
        const endpoint = commandInfo.endpoint;
        const commandData = formState[command.name + i];

        if (variableMode) {
            const variableValueBefore = getTemplateSrv().replace("$" + commandData.variableToSet, scopedVars);
            const valueToSet = getTemplateSrv().replace(commandData.valueToSet, scopedVars);
            let newValue: any = variableValueBefore;
            switch(commandData.changeMode) {
                case 'change':
                    newValue = valueToSet;
                    break;
                case 'add':
                    try {
                        newValue = parseFloat(variableValueBefore) + parseFloat(valueToSet);
                    }catch(err){};
                    break;
                case 'multiply':
                    try {
                        newValue = parseFloat(variableValueBefore) * parseFloat(valueToSet);
                    }catch(err){};
                    break;
                case 'input':
                    newValue = valueToSet;
                    break;
            }
            locationService.partial({[`var-${commandData.variableToSet}`]: newValue, replace: true})
            return;
        }

        setLoading(true);
        if (!datasource) {
            setLoading(false);
            appEvents.publish({
                type: AppEvents.alertError.name,
                payload: ['Datasource not available']
            });
            return;
        }

        // Use on/off command configuration based on which button was clicked
        let argumentsToUse = commandData?.arguments;
        let commentToUse = commandData?.comment;
        // Determine which command name to use — per-button override takes priority
        let commandNameToUse: string = command.qualifiedName;

        if (isOffCommand) {
            if (commandData?.offCommand?.commandName) {
                commandNameToUse = getTemplateSrv().replace(commandData.offCommand.commandName, scopedVars);
            }
            if (commandData?.offCommand?.arguments) {
                argumentsToUse = commandData.offCommand.arguments;
            }
            if (commandData?.offCommand?.comment) {
                commentToUse = commandData.offCommand.comment;
            }
        } else if (commandData?.isDualButton) {
            if (commandData?.onCommand?.commandName) {
                commandNameToUse = getTemplateSrv().replace(commandData.onCommand.commandName, scopedVars);
            }
            if (commandData?.onCommand?.arguments) {
                argumentsToUse = commandData.onCommand.arguments;
            }
            if (commandData?.onCommand?.comment) {
                commentToUse = commandData.onCommand.comment;
            }
        }

        // Apply variable substitution to all argument values,
        // then coerce the result to the correct type based on the command's argument metadata.
        // getTemplateSrv().replace() always returns a string, so numeric types must be re-parsed.
        const resolvedArguments: { [key: string]: any } = {};
        if (argumentsToUse) {
            // Build a lookup from arg name → engType using the command's argument metadata.
            // For dual buttons the active command may differ from the base command, so check
            // dualCommandInfos first and fall back to the shared commandInfo.
            const commandKey = command.name + i;
            const activeCommandInfo = isOffCommand
                ? (dualCommandInfos[`${commandKey}-off`] ?? commandInfo.command)
                : (commandData?.isDualButton ? (dualCommandInfos[`${commandKey}-on`] ?? commandInfo.command) : commandInfo.command);
            const argTypeLookup: { [name: string]: string } = {};
            (activeCommandInfo?.argument ?? []).forEach((arg: any) => {
                argTypeLookup[arg.name] = arg.type?.engType ?? 'string';
            });

            Object.keys(argumentsToUse).forEach((argName) => {
                const argValue = argumentsToUse[argName];
                const engType = argTypeLookup[argName] ?? 'string';

                if (typeof argValue === 'string') {
                    const resolvedValue = getTemplateSrv().replace(argValue, scopedVars);
                    // Coerce to the correct type after variable substitution
                    if (engType === 'integer') {
                        const parsed = parseInt(resolvedValue, 10);
                        resolvedArguments[argName] = isNaN(parsed) ? resolvedValue : parsed;
                    } else if (engType === 'float') {
                        const parsed = parseFloat(resolvedValue);
                        resolvedArguments[argName] = isNaN(parsed) ? resolvedValue : parsed;
                    } else if (engType === 'boolean') {
                        resolvedArguments[argName] = resolvedValue === 'true' ? true : resolvedValue === 'false' ? false : resolvedValue;
                    } else {
                        resolvedArguments[argName] = resolvedValue;
                    }
                } else {
                    resolvedArguments[argName] = argValue;
                }
            });
        }

        datasource.postResource(`endpoint/${endpoint}/command/issue`, {
            name: commandNameToUse,
            arguments: resolvedArguments,
            comment: getTemplateSrv().replace(commentToUse || '', scopedVars),
        })
            .then((_: any) => {
                setLoading(false);
                // Update dual button state to track which side was clicked
                if (commandData?.isDualButton) {
                    const newDualButtonStates = {
                        ...dualButtonStates,
                        [command.name + i]: isOffCommand ? 'off' : 'on'
                    };
                    updateDualButtonStates(newDualButtonStates);
                }
                appEvents.publish({
                    type: AppEvents.alertSuccess.name,
                    payload: [`Command ${command.name} issued successfully`]
                })
            })
            .catch(_ => {
                setLoading(false);
            });
    };

    return (
        <div style={{ width: '100%', height: '100%', overflow: editing ? 'scroll' : 'unset' }}>
            <div style={
                editing ? { display: 'grid', gridTemplateColumns: `repeat(auto-fit, minmax(300px, 2fr))`, gap: '2px', padding: '10px', width: '100%' }
                    : { display: 'flex', flexDirection: 'column', gap: '2px', padding: '10px', width: '100%', height: '100%' }}>
                {commandInfos.map((commandInfo, i) => {
                    const command = commandInfo.command;
                    const commandState = formState[command.name + i];

                    // Enhanced render function to support dual button mode
                    const render = (withSubmit = false) => {
                        // If this is a dual button, render the split view
                        if (commandState?.isDualButton) {
                            const activeState = dualButtonStates[command.name + i];
                            return (
                                <div style={{ 
                                    display: 'flex', 
                                    width: '100%', 
                                    height: '100%',
                                    gap: '0px'
                                }}>
                                    {/* ON Button (Left) */}
                                    <Button
                                        disabled={loading}
                                        style={{
                                            ...Shapes[commandState?.shape as any]?.css,
                                            width: '50%',
                                            height: '100%',
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            backgroundColor: commandState?.transparent === 'solid' || !commandState?.transparent
                                                ? commandState?.color as any
                                                : '#00000000',
                                            color: commandState?.textColor as any,
                                            borderColor: commandState?.transparent === 'outline'
                                                ? commandState?.color as any
                                                : undefined,
                                            borderRight: 'none',
                                            borderTopRightRadius: '0',
                                            borderBottomRightRadius: '0',
                                            opacity: activeState === 'off' ? 0.5 : 1,
                                        }}
                                        size={commandState?.size as any}
                                        fill={commandState?.transparent as any}
                                        tooltip={getTemplateSrv().replace(
                                            commandState?.onCommand?.tooltip
                                                ?? commandState?.tooltip
                                                ?? commandState?.onCommand?.label
                                                ?? commandState?.label
                                                ?? 'ON',
                                            scopedVars
                                        )}
                                        onClick={withSubmit ? () => handleSubmit(commandInfo, i, false) : undefined}
                                    >
                                        {getTemplateSrv().replace(commandState?.onCommand?.label ?? commandState?.label ?? 'ON', scopedVars)}
                                    </Button>
                                    
                                    {/* OFF Button (Right) */}
                                    <Button
                                        disabled={loading}
                                        style={{
                                            ...Shapes[commandState?.shape as any]?.css,
                                            width: '50%',
                                            height: '100%',
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            backgroundColor: commandState?.transparent === 'solid' || !commandState?.transparent
                                                ? (commandState?.offCommand?.color || commandState?.color) as any
                                                : '#00000000',
                                            color: (commandState?.offCommand?.textColor || commandState?.textColor) as any,
                                            borderColor: commandState?.transparent === 'outline'
                                                ? (commandState?.offCommand?.color || commandState?.color) as any
                                                : undefined,
                                            borderTopLeftRadius: '0',
                                            borderBottomLeftRadius: '0',
                                            opacity: activeState === 'on' ? 0.5 : 1,
                                        }}
                                        size={commandState?.size as any}
                                        fill={commandState?.transparent as any}
                                        tooltip={getTemplateSrv().replace(
                                            commandState?.offCommand?.tooltip
                                                ?? commandState?.tooltip
                                                ?? commandState?.offCommand?.label
                                                ?? commandState?.label
                                                ?? 'OFF',
                                            scopedVars
                                        )}
                                        onClick={withSubmit ? () => handleSubmit(commandInfo, i, true) : undefined}
                                    >
                                        {getTemplateSrv().replace(commandState?.offCommand?.label || 'OFF', scopedVars)}
                                    </Button>
                                </div>
                            );
                        }
                        
                        // Original single button rendering
                        return <Button
                            disabled={loading}
                            style={{
                                ...Shapes[commandState?.shape as any]?.css,
                                width: '100%',
                                height: '100%',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                backgroundColor:
                                    commandState?.transparent === 'solid' || !commandState?.transparent
                                        ? commandState?.color as any
                                        : '#00000000',
                                color: commandState?.textColor as any,
                                borderColor:
                                    commandState?.transparent === 'outline'
                                        ? commandState?.color as any
                                        : undefined,
                                backgroundImage:
                                    commandState?.shape === 'svg' && commandState?.customSVG
                                        ? `url("data:image/svg+xml;utf8,${encodeURIComponent(commandState?.customSVG)}")`
                                        : undefined,
                                backgroundRepeat: 'no-repeat',
                                backgroundSize: commandState?.bgSize || 'contain',
                                backgroundPosition: commandState?.bgPosition || 'center',
                            }}
                            size={commandState?.size as any}
                            icon={commandState?.icon as any}
                            fill={commandState?.transparent as any}
                            tooltip={getTemplateSrv().replace(commandState?.tooltip, scopedVars)}
                            onClick={withSubmit ? () => handleSubmit(commandInfo, i) : undefined}
                        >
                            {getTemplateSrv().replace(commandState?.label, scopedVars)}
                        </Button>
                    };
                    
                    if (!editing) {
                        // For variable mode with 'input' change mode, render an input field
                        if (variableMode && commandState?.changeMode === 'input') {
                            return (
                                <InputModeField
                                    key={command.name + i}
                                    variableToSet={commandState?.variableToSet}
                                    scopedVars={scopedVars}
                                    loading={loading}
                                    unit={commandState?.unit}
                                    showVariableLabel={commandState?.showVariableLabel}
                                    color={commandState?.color}
                                    textColor={commandState?.textColor}
                                    size={commandState?.size}
                                />
                            );
                        }
                        // For variable mode with add/multiply/set, wrap the button with label and unit
                        if (variableMode) {
                            const variableDisplayLabel = commandState?.variableToSet
                                ? (() => {
                                    const variable = getTemplateSrv().getVariables().find(vr => vr.name === commandState?.variableToSet);
                                    return variable ? (variable.label || variable.name) : commandState?.variableToSet;
                                })()
                                : '';
                            return (
                                <div key={command.name + i} style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', height: '100%', overflow: 'hidden' }}>
                                    {commandState?.showVariableLabel !== false && variableDisplayLabel && (
                                        <span style={{ whiteSpace: 'nowrap', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', flexShrink: 1, minWidth: 0 }}>{variableDisplayLabel}</span>
                                    )}
                                    <div style={{ flex: 1, height: '100%', minWidth: 0 }}>{render(true)}</div>
                                    {commandState?.unit && (
                                        <span style={{ whiteSpace: 'nowrap', flexShrink: 0 }}>{commandState.unit}</span>
                                    )}
                                </div>
                            );
                        }
                        return render(true);
                    }

                    return <Card key={command.name} style={{ width: '100%', padding: '20px' }}>
                        <Card.Heading>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                                <h4>{variableMode ? 'Variable Panel' : command.name}</h4>
                                {!variableMode && <Button
                                    disabled={loading}
                                    onClick={() => handleSubmit(commandInfo, i)} style={{ marginLeft: '20px' }} size='sm'>
                                    {loading ? <LoadingPlaceholder text="Issuing..." /> : "Issue Command"}
                                </Button>}
                            </div>
                        </Card.Heading>
                        <Card.Meta>{variableMode ? 'Configure Grafana variables through buttons' : (command.shortDescription || command.longDescription)}</Card.Meta>
                        <Card.Description>
                            <FieldSet style={{ display: 'flex', flexDirection: 'column', gap: '3px', width: '100%' }}>
                                {variableMode ? <>
                                    <Field label='Variable' description='Variable to change'>
                                        <Combobox
                                            disabled={loading}
                                            options={getTemplateSrv().getVariables().map(vr => ({ label: vr.label || vr.name, value: vr.name }))}
                                            value={commandState?.variableToSet || ''}
                                            onChange={(e: SelectableValue<string>) => {
                                                handleOptionChange(command.name, 'variableToSet', e.value, i);
                                            }}
                                        />
                                    </Field>
                                    <Field label='Change Mode' description='How to change the value'>
                                        <Combobox
                                            disabled={loading}
                                            options={[
                                                { label: "Set", value: 'change', description: "Set the variable to a value" },
                                                { label: "Add", value: 'add', description: "Add a number to the variable" },
                                                { label: "Multiply", value: 'multiply', description: "Multiply the variable by a number" },
                                                { label: "Input", value: 'input', description: "Text box that displays current value and accepts keyboard input" },
                                            ]}
                                            value={commandState?.changeMode || ''}
                                            onChange={(e: SelectableValue<string>) => {
                                                handleOptionChange(command.name, 'changeMode', e.value, i);
                                            }}
                                        />
                                    </Field>
                                    <Field label='Display variable name' description='Show the variable display name to the left of the input box in runtime'>
                                        <Checkbox
                                            value={commandState?.showVariableLabel !== false}
                                            onChange={(e) => handleOptionChange(command.name, 'showVariableLabel', e.currentTarget.checked, i)}
                                            label='Show variable name'
                                        />
                                    </Field>
                                    {commandState?.changeMode !== 'input' && (
                                        <Field label='Value' description='Value to use. You may write a custom value.'>
                                            <Combobox
                                                disabled={loading}
                                                options={((getTemplateSrv().getVariables().find(vr => vr.name === commandState?.variableToSet) as VariableWithMultiSupport)
                                                    ?.options || []).map(option => ({ label: option.text as string, value: option.value as string }))
                                                }
                                                value={commandState?.valueToSet || ''}
                                                createCustomValue
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'valueToSet', e.value, i);
                                                }}
                                            />
                                        </Field>
                                    )}
                                    <Field label='Comment' description='Optional comment'>
                                        <Input
                                            type='text'
                                            disabled={loading}
                                            value={commandState?.comment || ''}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                handleOptionChange(command.name, 'comment', e.target.value, i);
                                            }}
                                        />
                                    </Field>
                                    <Field label='Unit of Measurement' description='Unit to display next to the value (e.g., m/s, deg)'>
                                        <Input
                                            type='text'
                                            disabled={loading}
                                            value={commandState?.unit || ''}
                                            placeholder='Enter unit (e.g. m/s, deg)'
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                handleOptionChange(command.name, 'unit', e.target.value, i);
                                            }}
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                    {commandState?.changeMode !== 'input' && <>
                                        <Divider />
                                        <Field label='Button Label' description='Button label'>
                                            <Input
                                                type='text'
                                                disabled={loading}
                                                value={commandState?.label || ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                    handleOptionChange(command.name, 'label', e.target.value, i);
                                                }}
                                                style={{ width: '100%' }}
                                            />
                                        </Field>
                                        <Field label='Button Tooltip' description='Button tooltip'>
                                            <Input
                                                type='text'
                                                disabled={loading}
                                                value={commandState?.tooltip || ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                    handleOptionChange(command.name, 'tooltip', e.target.value, i);
                                                }}
                                                style={{ width: '100%' }}
                                            />
                                        </Field>
                                        <Field label='Icon' description='Icon name'>
                                            <Combobox
                                                disabled={loading}
                                                options={[
                                                    { label: 'None', value: '' },
                                                    ...getAvailableIcons().map(icon => ({ label: icon, value: icon }))
                                                ]}
                                                value={commandState?.icon || ''}
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'icon', e.value, i);
                                                }}
                                            />
                                        </Field>
                                        <Field label='Size' description='Button size'>
                                            <Combobox
                                                disabled={loading}
                                                options={[
                                                    { label: 'Mini', value: 'xs' },
                                                    { label: 'Small', value: 'sm' },
                                                    { label: 'Medium', value: 'md' },
                                                    { label: 'Large', value: 'lg' },
                                                ]}
                                                value={commandState?.size || 'md'}
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'size', e.value, i);
                                                }}
                                            />
                                        </Field>
                                        <Field label='Color' description='Button color'>
                                            <ColorPickerInput
                                                onChange={(color: string) => {
                                                    handleOptionChange(command.name, 'color', color, i);
                                                }}
                                                disabled={loading}
                                                color={commandState?.color || ''}
                                            />
                                        </Field>
                                        <Field label='Text Color' description='Text and icon color'>
                                            <ColorPickerInput
                                                onChange={(color: string) => {
                                                    handleOptionChange(command.name, 'textColor', color, i);
                                                }}
                                                disabled={loading}
                                                color={commandState?.textColor || ''}
                                            />
                                        </Field>
                                        <Field label='Transparent' description='Button transparency'>
                                            <Combobox
                                                disabled={loading}
                                                options={[
                                                    { label: 'Fill', value: 'solid' },
                                                    { label: 'Outline', value: 'outline' },
                                                    { label: 'Text', value: 'text' },
                                                ]}
                                                value={commandState?.transparent || 'solid'}
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'transparent', e.value, i);
                                                }}
                                            />
                                        </Field>
                                        <Field label='Shape' description='Button shape'>
                                            <Combobox
                                                disabled={loading}
                                                options={Object.keys(Shapes).map((shape) => ({
                                                    label: Shapes[shape as any].name,
                                                    value: shape,
                                                }))}
                                                value={commandState?.shape || 'rectangle'}
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'shape', e.value, i);
                                                }}
                                            />
                                        </Field>
                                        {commandState?.shape === 'svg' && <>
                                            <Field label="Custom SVG Shape">
                                                <FileUpload
                                                    accept=".svg"
                                                    onFileUpload={({ currentTarget: target }) => {
                                                        const file = target?.files?.[0];
                                                        if (!file) { return; }
                                                        const reader = new FileReader();
                                                        reader.onload = (event) => {
                                                            const svgContent = event.target?.result?.toString() || '';
                                                            handleOptionChange(command.name, 'customSVG', svgContent, i);
                                                        };
                                                        reader.readAsText(file);
                                                    }}
                                                    size="md"
                                                />
                                            </Field>
                                            <Field label="SVG Size" description="Controls how the background image is scaled.">
                                                <Combobox
                                                    options={[
                                                        { label: 'Contain', value: 'contain' },
                                                        { label: 'Cover', value: 'cover' },
                                                        { label: 'Auto', value: 'auto' },
                                                        { label: 'Stretch', value: '100% 100%' },
                                                    ]}
                                                    value={commandState?.bgSize || 'contain'}
                                                    createCustomValue
                                                    onChange={(v: SelectableValue<string>) =>
                                                        handleOptionChange(command.name, 'bgSize', v.value, i)
                                                    }
                                                />
                                            </Field>
                                            <Field label="SVG Position" description="Controls the position of the background image.">
                                                <Combobox
                                                    options={[
                                                        { label: 'Center', value: 'center' },
                                                        { label: 'Top Left', value: 'top left' },
                                                        { label: 'Top Right', value: 'top right' },
                                                        { label: 'Bottom Left', value: 'bottom left' },
                                                        { label: 'Bottom Right', value: 'bottom right' },
                                                    ]}
                                                    createCustomValue
                                                    value={commandState?.bgPosition || 'center'}
                                                    onChange={(v: SelectableValue<string>) =>
                                                        handleOptionChange(command.name, 'bgPosition', v.value, i)
                                                    }
                                                />
                                            </Field>
                                        </>}
                                        <Divider />
                                    </>}
                                    <Field label='Preview' description={commandState?.changeMode === 'input' ? 'Preview of the input box' : 'Preview of the button'}>
                                        <div style={{
                                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                                            height: '50px', width: '100%', objectFit: 'contain'
                                        }}>
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
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', height: '100%' }}>
                                                    {commandState?.showVariableLabel !== false && commandState?.variableToSet && (
                                                        <span style={{ whiteSpace: 'nowrap', fontWeight: 500 }}>
                                                            {(() => {
                                                                const variable = getTemplateSrv().getVariables().find(vr => vr.name === commandState?.variableToSet);
                                                                return variable ? (variable.label || variable.name) : commandState?.variableToSet;
                                                            })()}
                                                        </span>
                                                    )}
                                                    <div style={{ flex: 1, height: '100%' }}>{render()}</div>
                                                    {commandState?.unit && (
                                                        <span style={{ whiteSpace: 'nowrap' }}>{commandState.unit}</span>
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    </Field>
                                </> :
                                    <>
                                    {/* Button Type Selector */}
                                    <Field label='Button Type (Single or Dual)' description='Create a split button with separate commands'>
                                        <Combobox
                                            disabled={loading}
                                            options={[
                                                { label: 'Single Button', value: 'false' },
                                                { label: 'Dual Button', value: 'true' },
                                            ]}
                                            value={commandState?.isDualButton === true ? 'true' : 'false'}
                                            onChange={(e: SelectableValue<string> | null) => {
                                                if (!e || e.value === null || e.value === undefined) {
                                                    return;
                                                }
                                                handleOptionChange(command.name, 'isDualButton', e.value === 'true', i);
                                            }}
                                        />
                                    </Field>
                                    <Divider />

                                    {/* Single Button Configuration */}
                                    {!commandState?.isDualButton && <>
                                        {command.argument?.map((arg: any) => {
                                            const inputValue = commandState?.arguments?.[arg.name] ?? arg.initialValue;
                                            const errorMessage = errors[command.name]?.[arg.name];
                                            let inputField;

                                            if (arg.type.engType === 'enumeration') {
                                                inputField = (
                                                    <Combobox
                                                        disabled={loading}
                                                        value={inputValue}
                                                        onChange={(e: SelectableValue<any> | null) => {
                                                            if (!e || e.value === null || e.value === undefined) {
                                                                return;
                                                            }
                                                            handleInputChange(command.name, arg.name, e.value, i);
                                                            validateInput(command.name, arg, e.value);
                                                        }}
                                                        options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))}
                                                    />
                                                );
                                            } else if (arg.type.engType === 'boolean') {
                                                inputField = (
                                                    <Combobox
                                                        value={inputValue !== undefined && inputValue !== null ? String(inputValue) : ''}
                                                        disabled={loading}
                                                        onChange={(e: SelectableValue<any> | null) => {
                                                            if (!e || e.value === null || e.value === undefined) {
                                                                return;
                                                            }
                                                            const val = e.value === 'true' ? true : e.value === 'false' ? false : e.value;
                                                            handleInputChange(command.name, arg.name, val, i);
                                                            validateInput(command.name, arg, val);
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
                                                        value={inputValue}
                                                        placeholder={arg.type.engType === 'integer' || arg.type.engType === 'float' ? 'Enter value or $variable' : undefined}
                                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                            let val: any = e.target.value;
                                                            // Keep as string if it looks like a variable reference
                                                            if (!val.includes('$') && !val.includes('{')) {
                                                                if (arg.type.engType === 'integer') {
                                                                    const parsed = parseInt(val, 10);
                                                                    val = isNaN(parsed) ? val : parsed;
                                                                }
                                                                if (arg.type.engType === 'float') {
                                                                    const parsed = parseFloat(val);
                                                                    val = isNaN(parsed) ? val : parsed;
                                                                }
                                                            }
                                                            handleInputChange(command.name, arg.name, val, i);
                                                            validateInput(command.name, arg, e.target.value);
                                                        }}
                                                        style={{ width: '100%' }}
                                                    />
                                                );
                                            }

                                            return (
                                                <Field key={arg.name} label={arg.name} description={arg.description} style={{ width: '100%' }}>
                                                    <>
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                                                            {inputField}
                                                            <Badge text={`${arg.type.rangeMin ? `${arg.type.rangeMin} ≤` : ''} ${arg.type.engType} ${arg.type.rangeMax ? `≤ ${arg.type.rangeMax}` : ''}`} color="blue" />
                                                        </div>
                                                        {errorMessage && <Alert title="Invalid argument" severity="error">{errorMessage}</Alert>}
                                                    </>
                                                </Field>
                                            );
                                        })}
                                    </>}

                                    {/* Dual Button Configuration */}
                                    {commandState?.isDualButton && (() => {
                                        const commandKey = command.name + i;
                                        // The argument list shown for each side: prefer the separately fetched info if a different command was selected
                                        const onCommandInfo = dualCommandInfos[`${commandKey}-on`] ?? command;
                                        const offCommandInfo = dualCommandInfos[`${commandKey}-off`] ?? command;
                                        return (<>
                                        <Divider />
                                        <h5 style={{ marginTop: '10px', marginBottom: '10px' }}>Left Button Configuration</h5>

                                        {/* LEFT button: command picker */}
                                        <Field label='LEFT Command' description='Command to issue when the left button is clicked (leave empty to use the query command)'>
                                            <Combobox
                                                key={`on-cmd-${commandKey}`}
                                                options={async (q: string) => {
                                                    if (!commandInfo.endpoint) { return []; }
                                                    const results: Array<{name: string; description: string}> = await datasource!.getResource(
                                                        `endpoint/${commandInfo.endpoint}/commands`,
                                                        q ? { q } : undefined
                                                    );
                                                    return results.map(c => ({ label: c.name, value: c.name, description: c.description }));
                                                }}
                                                value={commandState?.onCommand?.commandName ?? null}
                                                isClearable
                                                onChange={(e: SelectableValue<string> | null) => {
                                                    const name = e?.value ?? '';
                                                    handleOptionChange(command.name, 'onCommand', {
                                                        ...commandState?.onCommand,
                                                        commandName: name,
                                                        arguments: {},
                                                    }, i);
                                                    if (name) {
                                                        fetchDualCommandInfo(commandKey, 'on', name, commandInfo.endpoint);
                                                    } else {
                                                        setDualCommandInfos(prev => { const n = {...prev}; delete n[`${commandKey}-on`]; return n; });
                                                    }
                                                }}
                                            />
                                        </Field>

                                        {/* LEFT button arguments (from per-button command or fallback to shared command) */}
                                        {onCommandInfo.argument?.map((arg: any) => {
                                            const inputValue = commandState?.onCommand?.arguments?.[arg.name] ?? arg.initialValue;
                                            let inputField;
                                            if (arg.type.engType === 'enumeration') {
                                                inputField = (
                                                    <Combobox disabled={loading} value={inputValue}
                                                        onChange={(e: SelectableValue<any>) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, arguments: { ...commandState?.onCommand?.arguments, [arg.name]: e.value } }, i)}
                                                        options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))} />
                                                );
                                            } else if (arg.type.engType === 'boolean') {
                                                inputField = (
                                                    <Combobox value={inputValue !== undefined && inputValue !== null ? String(inputValue) : ''} disabled={loading}
                                                        onChange={(e: SelectableValue<any>) => { const val = e.value === 'true' ? true : e.value === 'false' ? false : e.value; handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, arguments: { ...commandState?.onCommand?.arguments, [arg.name]: val } }, i); }}
                                                        options={[{ label: arg.type.zeroStringValue || 'False', value: 'false' }, { label: arg.type.oneStringValue || 'True', value: 'true' }]} />
                                                );
                                            } else {
                                                inputField = (
                                                    <Input disabled={loading} type="text" value={inputValue}
                                                        placeholder={arg.type.engType === 'integer' || arg.type.engType === 'float' ? 'Enter value or $variable' : undefined}
                                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                            let val: any = e.target.value;
                                                            if (!val.includes('$') && !val.includes('{')) {
                                                                if (arg.type.engType === 'integer') { const p = parseInt(val, 10); val = isNaN(p) ? val : p; }
                                                                if (arg.type.engType === 'float') { const p = parseFloat(val); val = isNaN(p) ? val : p; }
                                                            }
                                                            handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, arguments: { ...commandState?.onCommand?.arguments, [arg.name]: val } }, i);
                                                        }}
                                                        style={{ width: '100%' }} />
                                                );
                                            }
                                            return (
                                                <Field key={`on-${arg.name}`} label={`LEFT - ${arg.name}`} description={arg.description} style={{ width: '100%' }}>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                                                        {inputField}
                                                        <Badge text={`${arg.type.rangeMin ? `${arg.type.rangeMin} ≤` : ''} ${arg.type.engType} ${arg.type.rangeMax ? `≤ ${arg.type.rangeMax}` : ''}`} color="blue" />
                                                    </div>
                                                </Field>
                                            );
                                        })}

                                        <Field label='LEFT Comment' description='Optional comment for LEFT button'>
                                            <Input type='text' disabled={loading} value={commandState?.onCommand?.comment || ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, comment: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='LEFT Label' description='Label for LEFT button (supports $variable)'>
                                            <Input type='text' disabled={loading} value={commandState?.onCommand?.label ?? ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, label: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='LEFT Tooltip' description='Tooltip for LEFT button (supports $variable)'>
                                            <Input type='text' disabled={loading} value={commandState?.onCommand?.tooltip ?? ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, tooltip: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='LEFT Color' description='Button color for LEFT button'>
                                            <ColorPickerInput onChange={(color: string) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, color }, i)}
                                                disabled={loading} color={commandState?.onCommand?.color || ''} />
                                        </Field>
                                        <Field label='LEFT Text Color' description='Text color for LEFT button'>
                                            <ColorPickerInput onChange={(color: string) => handleOptionChange(command.name, 'onCommand', { ...commandState?.onCommand, textColor: color }, i)}
                                                disabled={loading} color={commandState?.onCommand?.textColor || ''} />
                                        </Field>

                                        <Divider />
                                        <h5 style={{ marginTop: '10px', marginBottom: '10px' }}>Right Button Configuration</h5>

                                        {/* RIGHT button: command picker */}
                                        <Field label='RIGHT Command' description='Command to issue when the right button is clicked (leave empty to use the query command)'>
                                            <Combobox
                                                key={`off-cmd-${commandKey}`}
                                                options={async (q: string) => {
                                                    if (!commandInfo.endpoint) { return []; }
                                                    const results: Array<{name: string; description: string}> = await datasource!.getResource(
                                                        `endpoint/${commandInfo.endpoint}/commands`,
                                                        q ? { q } : undefined
                                                    );
                                                    return results.map(c => ({ label: c.name, value: c.name, description: c.description }));
                                                }}
                                                value={commandState?.offCommand?.commandName ?? null}
                                                isClearable
                                                onChange={(e: SelectableValue<string> | null) => {
                                                    const name = e?.value ?? '';
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        commandName: name,
                                                        arguments: {},
                                                    }, i);
                                                    if (name) {
                                                        fetchDualCommandInfo(commandKey, 'off', name, commandInfo.endpoint);
                                                    } else {
                                                        setDualCommandInfos(prev => { const n = {...prev}; delete n[`${commandKey}-off`]; return n; });
                                                    }
                                                }}
                                            />
                                        </Field>

                                        {/* RIGHT button arguments */}
                                        {offCommandInfo.argument?.map((arg: any) => {
                                            const inputValue = commandState?.offCommand?.arguments?.[arg.name] ?? arg.initialValue;
                                            let inputField;
                                            if (arg.type.engType === 'enumeration') {
                                                inputField = (
                                                    <Combobox disabled={loading} value={inputValue}
                                                        onChange={(e: SelectableValue<any>) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, arguments: { ...commandState?.offCommand?.arguments, [arg.name]: e.value } }, i)}
                                                        options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))} />
                                                );
                                            } else if (arg.type.engType === 'boolean') {
                                                inputField = (
                                                    <Combobox value={inputValue !== undefined && inputValue !== null ? String(inputValue) : ''} disabled={loading}
                                                        onChange={(e: SelectableValue<any>) => { const val = e.value === 'true' ? true : e.value === 'false' ? false : e.value; handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, arguments: { ...commandState?.offCommand?.arguments, [arg.name]: val } }, i); }}
                                                        options={[{ label: arg.type.zeroStringValue || 'False', value: 'false' }, { label: arg.type.oneStringValue || 'True', value: 'true' }]} />
                                                );
                                            } else {
                                                inputField = (
                                                    <Input disabled={loading} type="text" value={inputValue}
                                                        placeholder={arg.type.engType === 'integer' || arg.type.engType === 'float' ? 'Enter value or $variable' : undefined}
                                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                            let val: any = e.target.value;
                                                            if (!val.includes('$') && !val.includes('{')) {
                                                                if (arg.type.engType === 'integer') { const p = parseInt(val, 10); val = isNaN(p) ? val : p; }
                                                                if (arg.type.engType === 'float') { const p = parseFloat(val); val = isNaN(p) ? val : p; }
                                                            }
                                                            handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, arguments: { ...commandState?.offCommand?.arguments, [arg.name]: val } }, i);
                                                        }}
                                                        style={{ width: '100%' }} />
                                                );
                                            }
                                            return (
                                                <Field key={`off-${arg.name}`} label={`RIGHT - ${arg.name}`} description={arg.description} style={{ width: '100%' }}>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                                                        {inputField}
                                                        <Badge text={`${arg.type.rangeMin ? `${arg.type.rangeMin} ≤` : ''} ${arg.type.engType} ${arg.type.rangeMax ? `≤ ${arg.type.rangeMax}` : ''}`} color="blue" />
                                                    </div>
                                                </Field>
                                            );
                                        })}

                                        <Field label='RIGHT Comment' description='Optional comment for RIGHT button'>
                                            <Input type='text' disabled={loading} value={commandState?.offCommand?.comment || ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, comment: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='RIGHT Label' description='Label for RIGHT button (supports $variable)'>
                                            <Input type='text' disabled={loading} value={commandState?.offCommand?.label ?? ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, label: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='RIGHT Tooltip' description='Tooltip for RIGHT button (supports $variable)'>
                                            <Input type='text' disabled={loading} value={commandState?.offCommand?.tooltip ?? ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, tooltip: e.target.value }, i)}
                                                style={{ width: '100%' }} />
                                        </Field>
                                        <Field label='RIGHT Color' description='Button color for RIGHT button'>
                                            <ColorPickerInput onChange={(color: string) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, color }, i)}
                                                disabled={loading} color={commandState?.offCommand?.color || ''} />
                                        </Field>
                                        <Field label='RIGHT Text Color' description='Text color for RIGHT button'>
                                            <ColorPickerInput onChange={(color: string) => handleOptionChange(command.name, 'offCommand', { ...commandState?.offCommand, textColor: color }, i)}
                                                disabled={loading} color={commandState?.offCommand?.textColor || ''} />
                                        </Field>
                                        </>);
                                    })()}

                                <Field label='Comment' description='Optional comment'>
                                    <Input
                                        type='text'
                                        disabled={loading}
                                        value={commandState?.comment || ''}
                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                            handleOptionChange(command.name, 'comment', e.target.value, i);
                                        }}
                                        style={{ width: '100%' }}
                                    />
                                </Field>
                                <Divider />
                                <Field label='Button Label' description='Button label'>
                                    <Input
                                        type='text'
                                        disabled={loading}
                                        value={commandState?.label || ''}
                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                            handleOptionChange(command.name, 'label', e.target.value, i);
                                        }}
                                        style={{ width: '100%' }}
                                    />
                                </Field>
                                <Field label='Button Tooltip' description='Button tooltip'>
                                    <Input
                                        type='text'
                                        disabled={loading}
                                        value={commandState?.tooltip || ''}
                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                            handleOptionChange(command.name, 'tooltip', e.target.value, i);
                                        }}
                                        style={{ width: '100%' }}
                                    />
                                </Field>
                                <Field label='Icon' description='Icon name'>
                                    <Combobox
                                        disabled={loading}
                                        options={
                                            [{ label: 'None', value: '' },
                                            ...getAvailableIcons().map(icon => ({ label: icon, value: icon }))]}
                                        value={commandState?.icon || ''}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'icon', e.value, i);
                                        }}
                                    />
                                </Field>
                                <Field label='Size' description='Button size'>
                                    <Combobox
                                        disabled={loading}
                                        options={[
                                            { label: 'Mini', value: 'xs' },
                                            { label: 'Small', value: 'sm' },
                                            { label: 'Medium', value: 'md' },
                                            { label: 'Large', value: 'lg' },
                                        ]}
                                        value={commandState?.size || 'md'}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'size', e.value, i);
                                        }}
                                    />
                                </Field>
                                <Field label='Color' description='Button color'>
                                    <ColorPickerInput
                                        onChange={(color: string) => {
                                            handleOptionChange(command.name, 'color', color, i);
                                        }}
                                        disabled={loading}
                                        color={commandState?.color || ''}
                                    />
                                </Field>
                                <Field label='Text Color' description='Text and icon color'>
                                    <ColorPickerInput
                                        onChange={(color: string) => {
                                            handleOptionChange(command.name, 'textColor', color, i);
                                        }}
                                        disabled={loading}
                                        color={commandState?.textColor || ''}
                                    />
                                </Field>
                                <Field label='Transparent' description='Button transparency'>
                                    <Combobox
                                        disabled={loading}
                                        options={[
                                            { label: 'Fill', value: 'solid' },
                                            { label: 'Outline', value: 'outline' },
                                            { label: 'Text', value: 'text' },
                                        ]}
                                        value={commandState?.transparent || 'solid'}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'transparent', e.value, i);
                                        }}
                                    />
                                </Field>
                                <Field label='Shape' description='Button shape'>
                                    <Combobox
                                        disabled={loading}
                                        options={Object.keys(Shapes).map((shape) => ({
                                            label: Shapes[shape as any].name,
                                            value: shape,
                                        }))}
                                        value={commandState?.shape || 'rectangle'}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'shape', e.value, i);
                                        }}
                                    />
                                </Field>
                                {commandState?.shape === 'svg' && <>
                                    <Field label="Custom SVG Shape">
                                        <FileUpload
                                            accept=".svg"
                                            onFileUpload={({ currentTarget: target }) => {
                                                const file = target?.files?.[0];
                                                if (!file) { return; }

                                                const reader = new FileReader();
                                                reader.onload = (event) => {
                                                    const svgContent = event.target?.result?.toString() || '';
                                                    // Store the SVG in formState and persist it in options
                                                    handleOptionChange(command.name, 'customSVG', svgContent, i);
                                                };
                                                reader.readAsText(file);
                                            }}
                                            size="md"
                                        />
                                    </Field>
                                    <Field label="SVG Size" description="Controls how the background image is scaled. You may write custom css backgroundSize value.">
                                        <Combobox
                                            options={[
                                                { label: 'Contain', value: 'contain' },
                                                { label: 'Cover', value: 'cover' },
                                                { label: 'Auto', value: 'auto' },
                                                { label: 'Stretch', value: '100% 100%' },
                                            ]}
                                            value={commandState?.bgSize || 'contain'}
                                            createCustomValue
                                            onChange={(v: SelectableValue<string>) =>
                                                handleOptionChange(command.name, 'bgSize', v.value, i)
                                            }
                                        />
                                    </Field>
                                    <Field label="SVG Position" description="Controls the position of the background image. You may write custom CSS backgroundPosition value.">
                                        <Combobox
                                            options={[
                                                { label: 'Center', value: 'center' },
                                                { label: 'Top Left', value: 'top left' },
                                                { label: 'Top Right', value: 'top right' },
                                                { label: 'Bottom Left', value: 'bottom left' },
                                                { label: 'Bottom Right', value: 'bottom right' },
                                            ]}
                                            createCustomValue
                                            value={commandState?.bgPosition || 'center'}
                                            onChange={(v: SelectableValue<string>) =>
                                                handleOptionChange(command.name, 'bgPosition', v.value, i)
                                            }
                                        />
                                    </Field>
                                </>}
                                <Field label='Preview' description='Preview of the button'>
                                    <div style={{
                                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                                        height: '50px', width: '100%', objectFit: 'contain'
                                    }}>
                                        {render()}
                                    </div>
                                </Field>
                                </>}
                            </FieldSet>
                        </Card.Description>
                    </Card>;
                }
                )}
            </div>
        </div>
    );
}
