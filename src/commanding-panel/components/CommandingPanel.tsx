import { AppEvents, PanelProps, SelectableValue } from '@grafana/data';
import { DataSourceWithBackend, getAppEvents, getTemplateSrv, useLocationService } from '@grafana/runtime';
import { Alert, Badge, Button, Card, ColorPickerInput, Divider, Field, FieldSet, getAvailableIcons, Input, LoadingPlaceholder, Select } from '@grafana/ui';
import { CommandForms, PanelOptions } from 'commanding-panel/types';
import React, { useState } from 'react';
import Shapes from './Shapes';

type CommandInfos = Array<{
    command: any,
    endpoint: string
}>

export default function CommandingPanel(props: PanelProps<PanelOptions>) {

    const { data, options, onOptionsChange } = props;
    const locService = useLocationService();
    const location = locService.getLocation();
    const editing = location.search.includes('editPanel=');

    const commandInfos: CommandInfos = [];
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

    const [formState, setFormState] = useState<CommandForms>(options.commandForms || {});
    const [errors, setErrors] = useState<{ [command: string]: { [arg: string]: string } }>({});
    const [loading, setLoading] = useState<boolean>(false);

    const handleInputChange = (commandName: string, argName: string, value: any, i: number) => {
        setFormState(prevState => {
            const newState ={
                ...prevState,
                [commandName + i]: {
                    ...prevState[commandName + i],
                    arguments: {
                        ...prevState[commandName]?.arguments,
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
            const newState ={
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

    const handleSubmit = (commandInfo: CommandInfos[number], i: number) => {
        const command = commandInfo.command;
        const endpoint = commandInfo.endpoint;
        const commandData = formState[command.name + i];
        console.log(formState);
        setLoading(true);
        const datasource = data.request?.targets[0].datasource as DataSourceWithBackend;
        if (!datasource) {
            setLoading(false);
            throw new Error('Datasource UID not found');
        }
        Object.setPrototypeOf(datasource, DataSourceWithBackend.prototype);
        datasource.postResource(`endpoint/${endpoint}/command/issue`, {
            name: command.qualifiedName,
            arguments: commandData?.arguments,
            comment: commandData?.comment,
        })
            .then((_: any) => {
                setLoading(false);
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
            editing ? { display: 'grid', gridTemplateColumns: `repeat(auto-fit, minmax(300px, 2fr))`, gap: '2px', padding: '10px', width: '100%'}
            : { display: 'flex', flexDirection: 'column', gap: '2px', padding: '10px', width: '100%', height: '100%'}}>
            {commandInfos.map((commandInfo, i) => {
                const command = commandInfo.command;
                const commandState = formState[command.name + i];
                if (!editing) {
                    return <Button 
                        key={command.name}
                        style={{ 
                            ...Shapes[formState[command.name + i]?.shape as any]?.css,
                            width: '100%', height: '100%', objectFit: 'contain', display: 'flex', alignItems: 'center', justifyContent: 'center',
                            backgroundColor: formState[command.name + i]?.transparent === 'solid' 
                            || !formState[command.name + i]?.transparent ?
                            formState[command.name + i]?.color as any : '#00000000',
                            color: formState[command.name + i]?.textColor as any,
                            borderColor: formState[command.name + i]?.transparent === 'outline' ?
                            formState[command.name + i]?.color as any : null,                          
                        }}
                        size={formState[command.name + i]?.size as any}
                        icon={loading ? 'spinner' :formState[command.name + i]?.icon as any}
                        fill={formState[command.name + i]?.transparent as any}
                        tooltip={getTemplateSrv().replace(formState[command.name + i]?.tooltip)}
                        onClick={() => handleSubmit(commandInfo, i)}
                        disabled={loading}
                    >{getTemplateSrv().replace(commandState?.label)}</Button>
                }

                return <Card key={command.name} style={{ width: '100%', padding: '20px' }}>
                    <Card.Heading>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                            <h4>{command.name}</h4>
                            <Button
                                disabled={loading}
                                onClick={() => handleSubmit(commandInfo, i)} style={{ marginLeft: '20px'}} size='sm'>
                                {loading ? <LoadingPlaceholder text="Issuing..." /> : "Issue Command"}
                            </Button>
                        </div>
                    </Card.Heading>
                    <Card.Meta>{command.shortDescription || command.longDescription}</Card.Meta>
                    <Card.Description>
                    <FieldSet style={{ display: 'flex', flexDirection: 'column', gap: '3px', width: '100%' }}>
                        {command.argument?.map((arg: any) => {
                            const inputValue = formState[command.name + i]?.arguments?.[arg.name] || arg.initialValue;
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
                                        options={arg.type.enumValue.map((ev: any) => ({ label: ev.label, value: ev.value }))}
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
                        <Field label='Comment' description='Optional comment'>
                            <Input
                                type='text'
                                disabled={loading}
                                value={formState[command.name + i]?.comment || ''}
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
                                value={formState[command.name + i]?.label || ''}
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
                                value={formState[command.name + i]?.tooltip || ''}
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
                                    [{label: 'None', value: ''},
                                        ...getAvailableIcons().map(icon => ({ label: icon, value: icon, icon: icon }))]}
                                value={formState[command.name + i]?.icon || ''}
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
                                value={formState[command.name + i]?.size || 'md'}
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
                                color={formState[command.name + i]?.color || ''}
                            />
                        </Field>
                        <Field label='Text Color' description='Text and icon color'>
                            <ColorPickerInput 
                                onChange={(color: string) => {
                                    handleOptionChange(command.name, 'textColor', color, i);
                                }}
                                disabled={loading}
                                color={formState[command.name + i]?.textColor || ''}
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
                                value={formState[command.name + i]?.transparent || 'solid'}
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
                                value={formState[command.name + i]?.shape || 'rectangle'}
                                onChange={(e: SelectableValue<string>) => {
                                    handleOptionChange(command.name, 'shape', e.value, i);
                                }}
                                style={{ width: '100%' }}
                            />
                        </Field>
                        <Field label='Preview' description='Preview of the button'>
                            <div style={{
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                            height: '50px', width: '100%', objectFit: 'contain'}}>
                            <Button
                                disabled={loading}
                                style={{ 
                                    ...Shapes[formState[command.name + i]?.shape as any]?.css,
                                    width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center',
                                    backgroundColor: formState[command.name + i]?.transparent === 'solid' 
                                    || !formState[command.name + i]?.transparent ?
                                    formState[command.name + i]?.color as any : '#00000000',
                                    color: formState[command.name + i]?.textColor as any,
                                    borderColor: formState[command.name + i]?.transparent === 'outline' ?
                                    formState[command.name + i]?.color as any : null,                          
                                }}
                                size={formState[command.name + i]?.size as any}
                                icon={formState[command.name + i]?.icon as any}
                                fill={formState[command.name + i]?.transparent as any}
                                tooltip={formState[command.name + i]?.tooltip}
                            >{formState[command.name + i]?.label}</Button>
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
