import React from 'react';
import { getTemplateSrv } from '@grafana/runtime';
import InputModeField from './InputModeField';
import { CommandButton } from './CommandButton';
import { CommandInfo, DualButtonStates } from '../types';

export function VariableRuntime(props: {
  commandInfo: CommandInfo;
  index: number;
  commandState: any;
  scopedVars: any;
  loading: boolean;
  dualButtonStates: DualButtonStates;
  onSubmit: (commandInfo: CommandInfo, index: number, isOffCommand?: boolean) => void;
}) {
  const { commandInfo, index, commandState, scopedVars, loading, dualButtonStates, onSubmit } = props;

  if (commandState?.changeMode === 'input') {
    return (
      <InputModeField
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

  const variableDisplayLabel = commandState?.variableToSet
    ? (() => {
        const variable = getTemplateSrv().getVariables().find((vr) => vr.name === commandState?.variableToSet);
        return variable ? variable.label || variable.name : commandState?.variableToSet;
      })()
    : '';

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', height: '100%', overflow: 'hidden' }}>
      {commandState?.showVariableLabel !== false && variableDisplayLabel && (
        <span style={{ whiteSpace: 'nowrap', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', flexShrink: 1, minWidth: 0 }}>{variableDisplayLabel}</span>
      )}
      <div style={{ flex: 1, height: '100%', minWidth: 0 }}>
        <CommandButton commandInfo={commandInfo} index={index} commandState={commandState} scopedVars={scopedVars} loading={loading} dualButtonStates={dualButtonStates} onSubmit={onSubmit} />
      </div>
      {commandState?.unit && <span style={{ whiteSpace: 'nowrap', flexShrink: 0 }}>{commandState.unit}</span>}
    </div>
  );
}
