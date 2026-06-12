import React from 'react';
import { Button, Card, FieldSet, LoadingPlaceholder } from '@grafana/ui';
import { CommandEditor } from './CommandEditor';
import { VariableEditor } from './VariableEditor';
import { CommandErrors, CommandInfo, DualButtonStates, DualCommandInfos, UpdateArgument, UpdateFormOption, ValidateArgument } from '../types';

export function CommandCard(props: {
  commandInfo: CommandInfo;
  index: number;
  commandState: any;
  variableMode: boolean;
  scopedVars: any;
  loading: boolean;
  datasource: any;
  errors: CommandErrors;
  dualCommandInfos: DualCommandInfos;
  dualButtonStates: DualButtonStates;
  onSubmit: (commandInfo: CommandInfo, index: number, isOffCommand?: boolean) => void;
  onArgumentChange: UpdateArgument;
  onOptionChange: UpdateFormOption;
  onValidate: ValidateArgument;
  fetchDualCommandInfo: (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => void;
  clearDualCommandInfo: (commandKey: string, side: 'on' | 'off') => void;
}) {
  const { commandInfo, index, commandState, variableMode, scopedVars, loading, datasource, errors, dualCommandInfos, dualButtonStates, onSubmit, onArgumentChange, onOptionChange, onValidate, fetchDualCommandInfo, clearDualCommandInfo } = props;
  const command = commandInfo.command;

  return (
    <Card key={`${command.name}${index}`} style={{ width: '100%', padding: '12px 14px' }}>
      <Card.Heading>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '12px', width: '100%' }}>
          <h4 style={{ margin: 0 }}>{variableMode ? 'Variable Panel' : `Command Button ${index + 1}`}</h4>
          {!variableMode && (
            <Button disabled={loading} onClick={() => onSubmit(commandInfo, index)}  size="sm">
              {loading ? <LoadingPlaceholder text="Issuing..." /> : 'Issue Command'}
            </Button>
          )}
        </div>
      </Card.Heading>
      <Card.Meta>
        {variableMode ? 'Configure Grafana variables through buttons' : 'Configure a command button'}
      </Card.Meta>
      <Card.Description>
        <FieldSet style={{ display: 'flex', flexDirection: 'column', gap: '0', width: '100%' }}>
          {variableMode ? (
            <VariableEditor commandInfo={commandInfo} index={index} commandState={commandState} scopedVars={scopedVars} loading={loading} dualButtonStates={dualButtonStates} onOptionChange={onOptionChange} />
          ) : (
            <CommandEditor
              commandInfo={commandInfo}
              index={index}
              commandState={commandState}
              scopedVars={scopedVars}
              loading={loading}
              datasource={datasource}
              errors={errors}
              dualCommandInfos={dualCommandInfos}
              dualButtonStates={dualButtonStates}
              onArgumentChange={onArgumentChange}
              onOptionChange={onOptionChange}
              onValidate={onValidate}
              fetchDualCommandInfo={fetchDualCommandInfo}
              clearDualCommandInfo={clearDualCommandInfo}
            />
          )}
        </FieldSet>
      </Card.Description>
    </Card>
  );
}
