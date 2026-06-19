import React from 'react';
import { SelectableValue } from '@grafana/data';
import { ColorPickerInput, Combobox, Field, FileUpload, getAvailableIcons, Input } from '@grafana/ui';
import Shapes from '../shapes';
import { UpdateFormOption } from '../types';

export function ButtonStyleFields(props: {
    commandName: string;
    index: number;
    commandState: any;
    loading: boolean;
    onOptionChange: UpdateFormOption;
    includeIcon?: boolean;
}) {
    const { commandName, index, commandState, loading, onOptionChange, includeIcon = true } = props;
    const set = (option: string, value: any) => onOptionChange(commandName, option, value, index);

    return (
        <>
            <Field label="Button Label" description="Button label" style={{ marginBottom: 0 }}>
                <Input
                    type="text"
                    disabled={loading}
                    value={commandState?.label || ''}
                    onChange={(e) => set('label', e.currentTarget.value)}
                    style={{ width: '100%' }}
                />
            </Field>
            <Field label="Button Tooltip" description="Button tooltip" style={{ marginBottom: 0 }}>
                <Input
                    type="text"
                    disabled={loading}
                    value={commandState?.tooltip || ''}
                    onChange={(e) => set('tooltip', e.currentTarget.value)}
                    style={{ width: '100%' }}
                />
            </Field>
            {includeIcon && (
                <Field label="Icon" description="Icon name" style={{ marginBottom: 0 }}>
                    <Combobox
                        disabled={loading}
                        options={[
                            { label: 'None', value: '' },
                            ...getAvailableIcons().map((icon) => ({ label: icon, value: icon })),
                        ]}
                        value={commandState?.icon || ''}
                        onChange={(e: SelectableValue<string>) => set('icon', e.value)}
                    />
                </Field>
            )}
            <Field label="Size" description="Button size" style={{ marginBottom: 0 }}>
                <Combobox
                    disabled={loading}
                    options={[
                        { label: 'Mini', value: 'xs' },
                        { label: 'Small', value: 'sm' },
                        { label: 'Medium', value: 'md' },
                        { label: 'Large', value: 'lg' },
                    ]}
                    value={commandState?.size || 'md'}
                    onChange={(e: SelectableValue<string>) => set('size', e.value)}
                />
            </Field>
            <Field label="Color" description="Button color" style={{ marginBottom: 0 }}>
                <ColorPickerInput
                    onChange={(color: string) => set('color', color)}
                    disabled={loading}
                    color={commandState?.color || ''}
                />
            </Field>
            <Field label="Text Color" description="Text and icon color" style={{ marginBottom: 0 }}>
                <ColorPickerInput
                    onChange={(color: string) => set('textColor', color)}
                    disabled={loading}
                    color={commandState?.textColor || ''}
                />
            </Field>
            <Field label="Transparent" description="Button transparency" style={{ marginBottom: 0 }}>
                <Combobox
                    disabled={loading}
                    options={[
                        { label: 'Fill', value: 'solid' },
                        { label: 'Outline', value: 'outline' },
                        { label: 'Text', value: 'text' },
                    ]}
                    value={commandState?.transparent || 'solid'}
                    onChange={(e: SelectableValue<string>) => set('transparent', e.value)}
                />
            </Field>
            <Field label="Shape" description="Button shape" style={{ marginBottom: 0 }}>
                <Combobox
                    disabled={loading}
                    options={Object.keys(Shapes).map((shape) => ({ label: Shapes[shape as any].name, value: shape }))}
                    value={commandState?.shape || 'rectangle'}
                    onChange={(e: SelectableValue<string>) => set('shape', e.value)}
                />
            </Field>
            {commandState?.shape === 'svg' && (
                <>
                    <Field label="Custom SVG Shape" style={{ marginBottom: 0 }}>
                        <FileUpload
                            accept=".svg"
                            onFileUpload={({ currentTarget: target }) => {
                                const file = target?.files?.[0];
                                if (!file) {
                                    return;
                                }
                                const reader = new FileReader();
                                reader.onload = (event) => set('customSVG', event.target?.result?.toString() || '');
                                reader.readAsText(file);
                            }}
                            size="md"
                        />
                    </Field>
                    <Field
                        label="SVG Size"
                        description="Controls how the background image is scaled."
                        style={{ marginBottom: 0 }}
                    >
                        <Combobox
                            options={[
                                { label: 'Contain', value: 'contain' },
                                { label: 'Cover', value: 'cover' },
                                { label: 'Auto', value: 'auto' },
                                { label: 'Stretch', value: '100% 100%' },
                            ]}
                            value={commandState?.bgSize || 'contain'}
                            createCustomValue
                            onChange={(v: SelectableValue<string>) => set('bgSize', v.value)}
                        />
                    </Field>
                    <Field
                        label="SVG Position"
                        description="Controls the position of the background image."
                        style={{ marginBottom: 0 }}
                    >
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
                            onChange={(v: SelectableValue<string>) => set('bgPosition', v.value)}
                        />
                    </Field>
                </>
            )}
        </>
    );
}
