import React from 'react';
import { SelectableValue } from '@grafana/data';
import { ColorPickerInput, Combobox, Field, FileUpload, getAvailableIcons, Input } from '@grafana/ui';
import { DataSourceWithBackend } from '@grafana/runtime';
import Shapes from '../shapes';
import { ArgumentField } from './ArgumentField';
import { FormSection } from './FormSection';
import { DualCommandInfos, UpdateFormOption } from '../types';
import { getCommandKey, getDualInfoKey } from '../utils/commandKeys';

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
  const { side, title, labelPrefix, command, commandInfo, index, endpoint, datasource, commandState, dualCommandInfos, loading, onOptionChange, fetchDualCommandInfo, clearDualCommandInfo } = props;
  const stateKey = side === 'on' ? 'onCommand' : 'offCommand';
  const commandKey = getCommandKey(command.name, index);
  const sideCommandInfo = dualCommandInfos[getDualInfoKey(commandKey, side)] ?? command;
  const sideState = commandState?.[stateKey] ?? {};

  const updateSide = (patch: Record<string, any>) => {
    onOptionChange(command.name, stateKey, { ...sideState, ...patch }, index);
  };

  const updateSideArgument = (_commandName: string, argName: string, value: any) => {
    updateSide({ arguments: { ...sideState.arguments, [argName]: value } });
  };

  return (
    <div style={{ border: '1px solid rgba(204, 204, 220, 0.16)', borderRadius: '4px', padding: '10px', minWidth: 0 }}>
      <FormSection title={title} separated={false}>
        <Field label={`${labelPrefix} Command`} description="Leave empty to use the query command" style={{ marginBottom: 0 }}>
          <Combobox
            key={`${side}-cmd-${commandKey}`}
            options={async (q: string) => {
              if (!endpoint || !datasource) {
                return [];
              }
              const results: Array<{ name: string; description: string }> = await datasource.getResource(`endpoint/${endpoint}/commands`, q ? { q } : undefined);
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

        <Field label={`${labelPrefix} Label`} description="Supports $variable" style={{ marginBottom: 0 }}>
          <Input type="text" disabled={loading} value={sideState.label ?? ''} onChange={(e) => updateSide({ label: e.currentTarget.value })} style={{ width: '100%' }} />
        </Field>
        <Field label={`${labelPrefix} Tooltip`} description="Supports $variable" style={{ marginBottom: 0 }}>
          <Input type="text" disabled={loading} value={sideState.tooltip ?? ''} onChange={(e) => updateSide({ tooltip: e.currentTarget.value })} style={{ width: '100%' }} />
        </Field>
        <Field label={`${labelPrefix} Comment`} description="Optional issue comment" style={{ marginBottom: 0 }}>
          <Input type="text" disabled={loading} value={sideState.comment || ''} onChange={(e) => updateSide({ comment: e.currentTarget.value })} style={{ width: '100%' }} />
        </Field>
      </FormSection>

      <FormSection title={`${labelPrefix} appearance`}>
        <Field label={`${labelPrefix} Icon`} description="Icon name" style={{ marginBottom: 0 }}>
          <Combobox
            disabled={loading}
            options={[{ label: 'None', value: '' }, ...getAvailableIcons().map((icon) => ({ label: icon, value: icon }))]}
            value={sideState.icon || ''}
            onChange={(e: SelectableValue<string>) => updateSide({ icon: e.value })}
          />
        </Field>

        <Field label={`${labelPrefix} Size`} description="Button size" style={{ marginBottom: 0 }}>
          <Combobox
            disabled={loading}
            options={[
              { label: 'Mini', value: 'xs' },
              { label: 'Small', value: 'sm' },
              { label: 'Medium', value: 'md' },
              { label: 'Large', value: 'lg' },
            ]}
            value={sideState.size || commandState?.size || 'md'}
            onChange={(e: SelectableValue<string>) => updateSide({ size: e.value })}
          />
        </Field>

        <Field label={`${labelPrefix} Color`} description="Button color" style={{ marginBottom: 0 }}>
          <ColorPickerInput onChange={(color: string) => updateSide({ color })} disabled={loading} color={sideState.color || ''} />
        </Field>
        <Field label={`${labelPrefix} Text Color`} description="Text color" style={{ marginBottom: 0 }}>
          <ColorPickerInput onChange={(color: string) => updateSide({ textColor: color })} disabled={loading} color={sideState.textColor || ''} />
        </Field>

        <Field label={`${labelPrefix} Transparent`} description="Button transparency" style={{ marginBottom: 0 }}>
          <Combobox
            disabled={loading}
            options={[
              { label: 'Fill', value: 'solid' },
              { label: 'Outline', value: 'outline' },
              { label: 'Text', value: 'text' },
            ]}
            value={sideState.transparent || commandState?.transparent || 'solid'}
            onChange={(e: SelectableValue<string>) => updateSide({ transparent: e.value })}
          />
        </Field>

        <Field label={`${labelPrefix} Shape`} description="Button shape" style={{ marginBottom: 0 }}>
          <Combobox
            disabled={loading}
            options={Object.keys(Shapes).map((shape) => ({ label: Shapes[shape as any].name, value: shape }))}
            value={sideState.shape || commandState?.shape || 'rectangle'}
            onChange={(e: SelectableValue<string>) => updateSide({ shape: e.value })}
          />
        </Field>

        {(sideState.shape || commandState?.shape) === 'svg' && (
          <>
            <Field label={`${labelPrefix} Custom SVG Shape`} style={{ marginBottom: 0 }}>
              <FileUpload
                accept=".svg"
                onFileUpload={({ currentTarget: target }) => {
                  const file = target?.files?.[0];
                  if (!file) {
                    return;
                  }
                  const reader = new FileReader();
                  reader.onload = (event) => updateSide({ customSVG: event.target?.result?.toString() || '' });
                  reader.readAsText(file);
                }}
                size="md"
              />
            </Field>
            <Field label={`${labelPrefix} SVG Size`} description="Controls how the background image is scaled." style={{ marginBottom: 0 }}>
              <Combobox
                options={[
                  { label: 'Contain', value: 'contain' },
                  { label: 'Cover', value: 'cover' },
                  { label: 'Auto', value: 'auto' },
                  { label: 'Stretch', value: '100% 100%' },
                ]}
                value={sideState.bgSize || commandState?.bgSize || 'contain'}
                createCustomValue
                onChange={(v: SelectableValue<string>) => updateSide({ bgSize: v.value })}
              />
            </Field>
            <Field label={`${labelPrefix} SVG Position`} description="Controls the position of the background image." style={{ marginBottom: 0 }}>
              <Combobox
                options={[
                  { label: 'Center', value: 'center' },
                  { label: 'Top Left', value: 'top left' },
                  { label: 'Top Right', value: 'top right' },
                  { label: 'Bottom Left', value: 'bottom left' },
                  { label: 'Bottom Right', value: 'bottom right' },
                ]}
                createCustomValue
                value={sideState.bgPosition || commandState?.bgPosition || 'center'}
                onChange={(v: SelectableValue<string>) => updateSide({ bgPosition: v.value })}
              />
            </Field>
          </>
        )}

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
      </FormSection>
    </div>
  );
}

export function DualButtonConfig(props: Omit<React.ComponentProps<typeof DualSideConfig>, 'side' | 'title' | 'labelPrefix'>) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '12px', width: '100%' }}>
      <DualSideConfig {...props} side="on" title="Left button" labelPrefix="LEFT" />
      <DualSideConfig {...props} side="off" title="Right button" labelPrefix="RIGHT" />
    </div>
  );
}
