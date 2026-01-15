import { AppEvents, PanelProps, SelectableValue, VariableWithMultiSupport } from '@grafana/data';
import { DataSourceWithBackend, getAppEvents, getTemplateSrv, locationService, useLocationService } from '@grafana/runtime';
import { Alert, Badge, Button, Card, ColorPickerInput, Divider, Field, FieldSet, FileUpload, getAvailableIcons, Input, LoadingPlaceholder, Select } from '@grafana/ui';
import { CommandForms, PanelOptions } from 'commanding-panel/types';
import React, { useState } from 'react';
import Shapes from './Shapes';

type CommandInfos = Array<{
    command: any,
    endpoint: string
}>

export interface CommandingPanelProps extends PanelProps<PanelOptions> {
    variableMode?: boolean;
}

export default function CommandingPanel({ variableMode = false, ...props }: CommandingPanelProps) {

    const { data, options, onOptionsChange } = props;
    const locService = useLocationService();
    const location = locService.getLocation();
    const editing = location.search.includes('editPanel=');
    const scopedVars = props.data.request?.scopedVars;

    const commandInfos: CommandInfos = [];
    if (variableMode) {
        commandInfos.push({ command: {}, endpoint: "" });
    } else {
        data.series.forEach((series) => {
            const commandField = series.fields.find(field => field.name === 'info');
            const endpointField = series.fields.find(field => field.name === 'endpoint');
            commandField?.values.forEach((command: any, index: number) => {
                const endpoint = endpointField?.values[index];
                if (command && endpoint) {
                    commandInfos.push({ command, endpoint });
                }
            });
        });
    }

    const [formState, setFormState] = useState<CommandForms>(options.commandForms || {});
    const [errors, setErrors] = useState<{ [command: string]: { [arg: string]: string } }>({});
    const [loading, setLoading] = useState<boolean>(false);
    
    // State to track which button (on/off) was last clicked
    // Persisted in panel data for consistency across multiple dashboards
    const dualButtonStates = options.dualButtonStates || {};
    const setDualButtonStates = (newState: { [key: string]: 'on' | 'off' }) => {
        onOptionsChange({ ...options, dualButtonStates: newState });
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
            }
            console.log(commandData.variableToSet);
            locationService.partial({[`var-${commandData.variableToSet}`]: newValue, replace: true})
            return;
        }

        setLoading(true);
        const datasource = data.request?.targets[0].datasource as DataSourceWithBackend;
        if (!datasource) {
            setLoading(false);
            throw new Error('Datasource UID not found');
        }
        Object.setPrototypeOf(datasource, DataSourceWithBackend.prototype);
        
        // Use off command configuration if this is the off button
        const argumentsToUse = isOffCommand && commandData?.offCommand?.arguments 
            ? commandData.offCommand.arguments 
            : commandData?.arguments;
        const commentToUse = isOffCommand && commandData?.offCommand?.comment 
            ? commandData.offCommand.comment 
            : commandData?.comment;

        datasource.postResource(`endpoint/${endpoint}/command/issue`, {
            name: command.qualifiedName,
            arguments: argumentsToUse,
            comment: commentToUse,
        })
            .then((_: any) => {
                setLoading(false);
                // Update dual button state to track which side was clicked
                if (commandData?.isDualButton) {
                    const newDualButtonStates = {
                        ...dualButtonStates,
                        [command.name + i]: isOffCommand ? 'off' : 'on'
                    };
                    setDualButtonStates(newDualButtonStates);
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
                        if (commandState?.isDualButton && !editing) {
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
                                        icon={commandState?.icon as any}
                                        fill={commandState?.transparent as any}
                                        tooltip={getTemplateSrv().replace(commandState?.tooltip, scopedVars)}
                                        onClick={withSubmit ? () => handleSubmit(commandInfo, i, false) : undefined}
                                    >
                                        {getTemplateSrv().replace(commandState?.label || 'ON', scopedVars)}
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
                                        icon={commandState?.offCommand?.icon || commandState?.icon as any}
                                        fill={commandState?.transparent as any}
                                        tooltip="Turn off"
                                        onClick={withSubmit ? () => handleSubmit(commandInfo, i, true) : undefined}
                                    >
                                        {commandState?.offCommand?.label || 'OFF'}
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
                                        <Select
                                            type='text'
                                            disabled={loading}
                                            options={getTemplateSrv().getVariables().map(vr => ({ label: vr.label || vr.name, value: vr.name }))}
                                            value={commandState?.variableToSet || ''}
                                            onChange={(e: SelectableValue<string>) => {
                                                handleOptionChange(command.name, 'variableToSet', e.value, i);
                                            }}
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                    <Field label='Change Mode' description='How to change the value'>
                                        <Select
                                            type='text'
                                            disabled={loading}
                                            options={[
                                                { label: "Set", value: 'change', description: "Set the variable to a value" },
                                                { label: "Add", value: 'add', description: "Add a number to the variable" },
                                                { label: "Multiply", value: 'multiply', description: "Multiply the variable by a number" },
                                            ]}
                                            value={commandState?.changeMode || ''}
                                            onChange={(e: SelectableValue<string>) => {
                                                handleOptionChange(command.name, 'changeMode', e.value, i);
                                            }}
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                    <Field label='Value' description='Value to use. You may write a custom value.'>
                                        <Select
                                            type='text'
                                            disabled={loading}
                                            options={((getTemplateSrv().getVariables().find(vr => vr.name === commandState?.variableToSet) as VariableWithMultiSupport)
                                                ?.options || []).map(option => ({ label: option.text as string, value: option.value as string }))
                                            }
                                            value={commandState?.valueToSet || ''}
                                            defaultValue={commandState?.valueToSet || ''}
                                            allowCustomValue
                                            onChange={(e: SelectableValue<string>) => {
                                                handleOptionChange(command.name, 'valueToSet', e.value, i);
                                            }}
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                </> :
                                    command.argument?.map((arg: any) => {
                                        const inputValue = commandState?.arguments?.[arg.name] || arg.initialValue;
                                        const errorMessage = errors[command.name]?.[arg.name];
                                        let inputField;

                                        if (arg.type.engType === 'enumeration') {
                                            inputField = (
                                                <Select
                                                    disabled={loading}
                                                    value={inputValue}
                                                    onChange={(e: SelectableValue<any>) => {
                                                        handleInputChange(command.name, arg.name, e.value, i);
                                                        validateInput(command.name, arg, e.value);
                                                    }}
                                                    options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))}
                                                />
                                            );
                                        } else if (arg.type.engType === 'boolean') {
                                            inputField = (
                                                <Select
                                                    value={inputValue}
                                                    disabled={loading}
                                                    style={{ width: '100%' }}
                                                    onChange={(e: SelectableValue<any>) => {
                                                        handleInputChange(command.name, arg.name, e.value, i);
                                                        validateInput(command.name, arg, e.value);
                                                    }}
                                                    options={[
                                                        { label: arg.type.zeroStringValue || 'False', value: false },
                                                        { label: arg.type.oneStringValue || 'True', value: true },
                                                    ]}
                                                    fullWidth
                                                />
                                            );
                                        } else {
                                            inputField = (
                                                <Input
                                                    disabled={loading}
                                                    type={arg.type.engType === 'integer' || arg.type.engType === 'float' ? 'number' : 'text'}
                                                    value={inputValue}
                                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                        let val: any = e.target.value;
                                                        if (arg.type.engType === 'integer') {
                                                            val = parseInt(val, 10);
                                                        }
                                                        if (arg.type.engType === 'float') {
                                                            val = parseFloat(val);
                                                        }
                                                        handleInputChange(command.name, arg.name, val, i);
                                                        validateInput(command.name, arg, e.target.value);
                                                    }}
                                                    min={arg.type.rangeMin}
                                                    max={arg.type.rangeMax}
                                                    style={{ width: '100%' }}
                                                    step={arg.type.engType === 'integer' ? 1 : undefined}
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
                                
                                {/* Dual Button Toggle */}
                                {!variableMode && <>
                                    <Divider />
                                    <Field label='Dual On/Off Button' description='Create a split button with separate on/off commands'>
                                        <Select
                                            disabled={loading}
                                            options={[
                                                { label: 'Single Button', value: false },
                                                { label: 'Dual On/Off Button', value: true },
                                            ]}
                                            value={commandState?.isDualButton || false}
                                            onChange={(e: SelectableValue<boolean>) => {
                                                handleOptionChange(command.name, 'isDualButton', e.value, i);
                                            }}
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                    
                                    {/* OFF Button Configuration Section */}
                                    {commandState?.isDualButton && <>
                                        <Divider />
                                        <h5 style={{ marginTop: '10px', marginBottom: '10px' }}>OFF Button Configuration</h5>
                                        
                                        {/* OFF Button Arguments */}
                                        {command.argument?.map((arg: any) => {
                                            const inputValue = commandState?.offCommand?.arguments?.[arg.name] || arg.initialValue;
                                            let inputField;

                                            if (arg.type.engType === 'enumeration') {
                                                inputField = (
                                                    <Select
                                                        disabled={loading}
                                                        value={inputValue}
                                                        onChange={(e: SelectableValue<any>) => {
                                                            handleOptionChange(command.name, 'offCommand', {
                                                                ...commandState?.offCommand,
                                                                arguments: {
                                                                    ...commandState?.offCommand?.arguments,
                                                                    [arg.name]: e.value
                                                                }
                                                            }, i);
                                                        }}
                                                        options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.label }))}
                                                    />
                                                );
                                            } else if (arg.type.engType === 'boolean') {
                                                inputField = (
                                                    <Select
                                                        value={inputValue}
                                                        disabled={loading}
                                                        style={{ width: '100%' }}
                                                        onChange={(e: SelectableValue<any>) => {
                                                            handleOptionChange(command.name, 'offCommand', {
                                                                ...commandState?.offCommand,
                                                                arguments: {
                                                                    ...commandState?.offCommand?.arguments,
                                                                    [arg.name]: e.value
                                                                }
                                                            }, i);
                                                        }}
                                                        options={[
                                                            { label: arg.type.zeroStringValue || 'False', value: false },
                                                            { label: arg.type.oneStringValue || 'True', value: true },
                                                        ]}
                                                        fullWidth
                                                    />
                                                );
                                            } else {
                                                inputField = (
                                                    <Input
                                                        disabled={loading}
                                                        type={arg.type.engType === 'integer' || arg.type.engType === 'float' ? 'number' : 'text'}
                                                        value={inputValue}
                                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                            let val: any = e.target.value;
                                                            if (arg.type.engType === 'integer') {
                                                                val = parseInt(val, 10);
                                                            }
                                                            if (arg.type.engType === 'float') {
                                                                val = parseFloat(val);
                                                            }
                                                            handleOptionChange(command.name, 'offCommand', {
                                                                ...commandState?.offCommand,
                                                                arguments: {
                                                                    ...commandState?.offCommand?.arguments,
                                                                    [arg.name]: val
                                                                }
                                                            }, i);
                                                        }}
                                                        min={arg.type.rangeMin}
                                                        max={arg.type.rangeMax}
                                                        style={{ width: '100%' }}
                                                        step={arg.type.engType === 'integer' ? 1 : undefined}
                                                    />
                                                );
                                            }

                                            return (
                                                <Field key={`off-${arg.name}`} label={`OFF - ${arg.name}`} description={arg.description} style={{ width: '100%' }}>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                                                        {inputField}
                                                        <Badge text={`${arg.type.rangeMin ? `${arg.type.rangeMin} ≤` : ''} ${arg.type.engType} ${arg.type.rangeMax ? `≤ ${arg.type.rangeMax}` : ''}`} color="blue" />
                                                    </div>
                                                </Field>
                                            );
                                        })}
                                        
                                        <Field label='OFF Comment' description='Optional comment for OFF command'>
                                            <Input
                                                type='text'
                                                disabled={loading}
                                                value={commandState?.offCommand?.comment || ''}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        comment: e.target.value
                                                    }, i);
                                                }}
                                                style={{ width: '100%' }}
                                            />
                                        </Field>
                                        
                                        <Field label='OFF Label' description='Label for OFF button'>
                                            <Input
                                                type='text'
                                                disabled={loading}
                                                value={commandState?.offCommand?.label || 'OFF'}
                                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        label: e.target.value
                                                    }, i);
                                                }}
                                                style={{ width: '100%' }}
                                            />
                                        </Field>
                                        
                                        <Field label='OFF Icon' description='Icon for OFF button'>
                                            <Select
                                                disabled={loading}
                                                options={
                                                    [{ label: 'None', value: '' },
                                                    ...getAvailableIcons().map(icon => ({ label: icon, value: icon, icon: icon }))]}
                                                value={commandState?.offCommand?.icon || ''}
                                                onChange={(e: SelectableValue<string>) => {
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        icon: e.value
                                                    }, i);
                                                }}
                                                style={{ width: '100%' }}
                                            />
                                        </Field>
                                        
                                        <Field label='OFF Color' description='Button color for OFF state'>
                                            <ColorPickerInput
                                                onChange={(color: string) => {
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        color: color
                                                    }, i);
                                                }}
                                                disabled={loading}
                                                color={commandState?.offCommand?.color || ''}
                                            />
                                        </Field>
                                        
                                        <Field label='OFF Text Color' description='Text color for OFF button'>
                                            <ColorPickerInput
                                                onChange={(color: string) => {
                                                    handleOptionChange(command.name, 'offCommand', {
                                                        ...commandState?.offCommand,
                                                        textColor: color
                                                    }, i);
                                                }}
                                                disabled={loading}
                                                color={commandState?.offCommand?.textColor || ''}
                                            />
                                        </Field>
                                    </>}
                                </>}
                                
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
                                    <Select
                                        disabled={loading}
                                        options={
                                            [{ label: 'None', value: '' },
                                            ...getAvailableIcons().map(icon => ({ label: icon, value: icon, icon: icon }))]}
                                        value={commandState?.icon || ''}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'icon', e.value, i);
                                        }}
                                        style={{ width: '100%' }}
                                    />
                                </Field>
                                <Field label='Size' description='Button size'>
                                    <Select
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
                                        style={{ width: '100%' }}
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
                                    <Select
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
                                        style={{ width: '100%' }}
                                    />
                                </Field>
                                <Field label='Shape' description='Button shape'>
                                    <Select
                                        disabled={loading}
                                        options={Object.keys(Shapes).map((shape) => ({
                                            label: Shapes[shape as any].name,
                                            value: shape,
                                        }))}
                                        value={commandState?.shape || 'rectangle'}
                                        onChange={(e: SelectableValue<string>) => {
                                            handleOptionChange(command.name, 'shape', e.value, i);
                                        }}
                                        style={{ width: '100%' }}
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
                                        <Select
                                            options={[
                                                { label: 'Contain', value: 'contain' },
                                                { label: 'Cover', value: 'cover' },
                                                { label: 'Auto', value: 'auto' },
                                                { label: 'Stretch', value: '100% 100%' },
                                            ]}
                                            value={commandState?.bgSize || 'contain'}
                                            allowCustomValue
                                            onChange={(v) =>
                                                handleOptionChange(command.name, 'bgSize', v.value, i)
                                            }
                                            style={{ width: '100%' }}
                                        />
                                    </Field>
                                    <Field label="SVG Position" description="Controls the position of the background image. You may write custom CSS backgroundPosition value.">
                                        <Select
                                            options={[
                                                { label: 'Center', value: 'center' },
                                                { label: 'Top Left', value: 'top left' },
                                                { label: 'Top Right', value: 'top right' },
                                                { label: 'Bottom Left', value: 'bottom left' },
                                                { label: 'Bottom Right', value: 'bottom right' },
                                            ]}
                                            allowCustomValue
                                            value={commandState?.bgPosition || 'center'}
                                            onChange={(v) =>
                                                handleOptionChange(command.name, 'bgPosition', v.value, i)
                                            }
                                            style={{ width: '100%' }}
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
                            </FieldSet>
                        </Card.Description>
                    </Card>;
                }
                )}
            </div>
        </div>
    );
}
