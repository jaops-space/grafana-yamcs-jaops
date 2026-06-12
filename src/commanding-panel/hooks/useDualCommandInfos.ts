import { useCallback, useState } from 'react';
import { DataSourceWithBackend } from '@grafana/runtime';
import { DualCommandInfos, DualSide } from '../types';
import { getDualInfoKey } from '../utils/commandKeys';

export function useDualCommandInfos(datasource: DataSourceWithBackend | null) {
  const [dualCommandInfos, setDualCommandInfos] = useState<DualCommandInfos>({});

  const fetchDualCommandInfo = useCallback(
    async (commandKey: string, side: DualSide, commandName: string, endpoint: string) => {
      if (!datasource || !commandName || !endpoint) {
        return;
      }

      try {
        const info = await datasource.getResource(`endpoint/${endpoint}/command/info`, { name: commandName });
        setDualCommandInfos((prev) => ({ ...prev, [getDualInfoKey(commandKey, side)]: info }));
      } catch (err) {
        console.error('Failed to fetch dual command info', err);
      }
    },
    [datasource]
  );

  const clearDualCommandInfo = useCallback((commandKey: string, side: DualSide) => {
    setDualCommandInfos((prev) => {
      const next = { ...prev };
      delete next[getDualInfoKey(commandKey, side)];
      return next;
    });
  }, []);

  return { dualCommandInfos, fetchDualCommandInfo, clearDualCommandInfo };
}
