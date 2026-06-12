import { useEffect, useState } from 'react';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { PanelOptions } from 'commanding-panel/types';
import { CommandInfos } from '../types';
import { getCommandKey } from '../utils/commandKeys';

export function useCommandInfos(params: {
  datasource: DataSourceWithBackend | null;
  targets: any[];
  scopedVars: any;
  variableMode: boolean;
  options: PanelOptions;
  fetchDualCommandInfo: (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => void;
}) {
  const { datasource, targets, scopedVars, variableMode, options, fetchDualCommandInfo } = params;
  const [commandInfos, setCommandInfos] = useState<CommandInfos>([]);

  useEffect(() => {
    if (variableMode) {
      setCommandInfos([{ command: {}, endpoint: '' }]);
      return;
    }

    if (!datasource || targets.length === 0) {
      return;
    }

    Promise.all(
      targets.map(async (target: any) => {
        const endpoint: string = target.asVariable
          ? getTemplateSrv().replace(target.endpointVariable, scopedVars)
          : target.endpoint;
        const command: string = getTemplateSrv().replace(target.command, scopedVars);

        if (!endpoint || !command) {
          return null;
        }

        try {
          const info = await datasource.getResource(`endpoint/${endpoint}/command/info`, { name: command });
          return { command: info, endpoint };
        } catch (err) {
          console.error('Failed to fetch command info', err);
          return null;
        }
      })
    ).then((results) => {
      const infos = results.filter(Boolean) as CommandInfos;
      setCommandInfos(infos);

      infos.forEach((info, index) => {
        const commandKey = getCommandKey(info.command.name ?? '', index);
        const savedState = (options.commandForms ?? {})[commandKey];
        if (!savedState?.isDualButton) {
          return;
        }
        if (savedState.onCommand?.commandName) {
          fetchDualCommandInfo(commandKey, 'on', savedState.onCommand.commandName, info.endpoint);
        }
        if (savedState.offCommand?.commandName) {
          fetchDualCommandInfo(commandKey, 'off', savedState.offCommand.commandName, info.endpoint);
        }
      });
    });
  }, [datasource, targets, scopedVars, variableMode, options.commandForms, fetchDualCommandInfo]);

  return commandInfos;
}
