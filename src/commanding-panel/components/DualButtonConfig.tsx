import { DataSourceWithBackend } from '@grafana/runtime';
import { ColorPickerInput, Field, Input } from '@grafana/ui';
import React from 'react';
import { DualCommandInfos, UpdateFormOption } from '../types';
import { getCommandKey, getDualInfoKey } from '../utils/commandKeys';
import { ArgumentField } from './ArgumentField';
import { CommandSelector } from './CommandSelector';
import { FormSection } from './FormSection';

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

  const updateSide = (patch: Record<string, any>) => {
    onOptionChange(command.name, stateKey, { ...commandState?.[stateKey], ...patch }, index);
  };

  const updateSideArgument = (_commandName: string, argName: string, value: any) => {
    updateSide({ arguments: { ...commandState?.[stateKey]?.arguments, [argName]: value } });
  };

  return (
    <div style={{ border: '1px solid rgba(204, 204, 220, 0.16)', borderRadius: '4px', padding: '10px', minWidth: 0 }}>
      <FormSection title={title} separated={false}>
      <CommandSelector
        label={`${labelPrefix} Command`}
        description="Command issued by this side"
        endpoint={endpoint}
        datasource={datasource}
        value={commandState?.[stateKey]?.commandName ?? null}
        disabled={loading}
        commandInfo={sideCommandInfo}
        onChange={(name) => {
          updateSide({ commandName: name, arguments: {} });

          if (name) {
            fetchDualCommandInfo(commandKey, side, name, commandInfo.endpoint);
          } else {
            clearDualCommandInfo(commandKey, side);
          }
        }}
      />

      <Field label={`${labelPrefix} Label`} description="Supports $variable" style={{ marginBottom: 0 }}>
        <Input type="text" disabled={loading} value={commandState?.[stateKey]?.label ?? ''} onChange={(e) => updateSide({ label: e.currentTarget.value })} style={{ width: '100%' }} />
      </Field>
      <Field label={`${labelPrefix} Tooltip`} description="Supports $variable" style={{ marginBottom: 0 }}>
        <Input type="text" disabled={loading} value={commandState?.[stateKey]?.tooltip ?? ''} onChange={(e) => updateSide({ tooltip: e.currentTarget.value })} style={{ width: '100%' }} />
      </Field>
      <Field label={`${labelPrefix} Comment`} description="Optional issue comment" style={{ marginBottom: 0 }}>
        <Input type="text" disabled={loading} value={commandState?.[stateKey]?.comment || ''} onChange={(e) => updateSide({ comment: e.currentTarget.value })} style={{ width: '100%' }} />
      </Field>
      <Field label={`${labelPrefix} Color`} description="Button color" style={{ marginBottom: 0 }}>
        <ColorPickerInput onChange={(color: string) => updateSide({ color })} disabled={loading} color={commandState?.[stateKey]?.color || ''} />
      </Field>
      <Field label={`${labelPrefix} Text Color`} description="Text color" style={{ marginBottom: 0 }}>
        <ColorPickerInput onChange={(color: string) => updateSide({ textColor: color })} disabled={loading} color={commandState?.[stateKey]?.textColor || ''} />
      </Field>

      {sideCommandInfo.argument?.map((arg: any) => (
        <ArgumentField
          key={`${side}-${arg.name}`}
          commandName={command.name}
          index={index}
          arg={arg}
          value={commandState?.[stateKey]?.arguments?.[arg.name] ?? arg.initialValue}
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
