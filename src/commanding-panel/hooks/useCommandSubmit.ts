import { useCallback } from 'react';
import { AppEvents } from '@grafana/data';
import { DataSourceWithBackend, getAppEvents, getTemplateSrv, locationService } from '@grafana/runtime';
import { CommandForms, PanelOptions } from 'commanding-panel/types';
import { CommandInfo, DualButtonStates, DualCommandInfos } from '../types';
import { getCommandKey, getDualInfoKey } from '../utils/commandKeys';
import { resolveArguments } from '../utils/commandArguments';

export function useCommandSubmit(params: {
  datasource: DataSourceWithBackend | null;
  formState: CommandForms;
  scopedVars: any;
  variableMode: boolean;
  options: PanelOptions;
  setLoading: (loading: boolean) => void;
  dualCommandInfos: DualCommandInfos;
  dualButtonStates: DualButtonStates;
  updateDualButtonStates: (newStates: DualButtonStates) => void;
}) {
  const { datasource, formState, scopedVars, variableMode, setLoading, dualCommandInfos, dualButtonStates, updateDualButtonStates } = params;
  const appEvents = getAppEvents();

  return useCallback(
    (commandInfo: CommandInfo, index: number, isOffCommand = false) => {
      const command = commandInfo.command;
      const endpoint = commandInfo.endpoint;
      const commandKey = getCommandKey(command.name, index);
      const commandData = formState[commandKey];

      if (variableMode) {
        const variableValueBefore = getTemplateSrv().replace(`$${commandData.variableToSet}`, scopedVars);
        const valueToSet = getTemplateSrv().replace(commandData.valueToSet, scopedVars);
        let newValue: any = variableValueBefore;

        switch (commandData.changeMode) {
          case 'change':
          case 'input':
            newValue = valueToSet;
            break;
          case 'add':
            newValue = parseFloat(variableValueBefore) + parseFloat(valueToSet);
            break;
          case 'multiply':
            newValue = parseFloat(variableValueBefore) * parseFloat(valueToSet);
            break;
        }

        locationService.partial({ [`var-${commandData.variableToSet}`]: newValue, replace: true });
        return;
      }

      setLoading(true);
      if (!datasource) {
        setLoading(false);
        appEvents.publish({ type: AppEvents.alertError.name, payload: ['Datasource not available'] });
        return;
      }

      let argumentsToUse = commandData?.arguments;
      let commentToUse = commandData?.comment;
      let commandNameToUse: string = getTemplateSrv().replace(commandData?.commandName || command.qualifiedName || command.name || '', scopedVars);

      if (isOffCommand) {
        if (commandData?.offCommand?.commandName) {
          commandNameToUse = getTemplateSrv().replace(commandData.offCommand.commandName, scopedVars);
        }
        argumentsToUse = commandData?.offCommand?.arguments ?? argumentsToUse;
        commentToUse = commandData?.offCommand?.comment ?? commentToUse;
      } else if (commandData?.isDualButton) {
        if (commandData?.onCommand?.commandName) {
          commandNameToUse = getTemplateSrv().replace(commandData.onCommand.commandName, scopedVars);
        }
        argumentsToUse = commandData?.onCommand?.arguments ?? argumentsToUse;
        commentToUse = commandData?.onCommand?.comment ?? commentToUse;
      }

      if (!commandNameToUse) {
        setLoading(false);
        appEvents.publish({ type: AppEvents.alertError.name, payload: ['Select a command before issuing'] });
        return;
      }

      const activeCommandInfo = isOffCommand
        ? dualCommandInfos[getDualInfoKey(commandKey, 'off')] ?? commandInfo.command
        : commandData?.isDualButton
          ? dualCommandInfos[getDualInfoKey(commandKey, 'on')] ?? commandInfo.command
          : commandInfo.command;

      datasource
        .postResource(`endpoint/${endpoint}/command/issue`, {
          name: commandNameToUse,
          arguments: resolveArguments(argumentsToUse, activeCommandInfo, scopedVars),
          comment: getTemplateSrv().replace(commentToUse || '', scopedVars),
        })
        .then(() => {
          setLoading(false);
          if (commandData?.isDualButton) {
            updateDualButtonStates({ ...dualButtonStates, [commandKey]: isOffCommand ? 'off' : 'on' });
          }
          appEvents.publish({ type: AppEvents.alertSuccess.name, payload: [`Command ${commandNameToUse} issued successfully`] });
        })
        .catch(() => setLoading(false));
    },
    [appEvents, datasource, dualButtonStates, dualCommandInfos, formState, scopedVars, setLoading, updateDualButtonStates, variableMode]
  );
}
