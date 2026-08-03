import { useCallback, useRef, useState } from 'react';
import { DataSourceWithBackend } from '@grafana/runtime';
import { DualCommandInfos, DualSide } from '../types';
import { getDualInfoKey } from '../utils/commandKeys';

export function useDualCommandInfos(datasource: DataSourceWithBackend | null) {
    const [dualCommandInfos, setDualCommandInfos] = useState<DualCommandInfos>({});
    const lastFetchedRef = useRef<Record<string, string>>({});

    const fetchDualCommandInfo = useCallback(
        async (commandKey: string, side: DualSide, commandName: string, endpoint: string) => {
            if (!datasource || !commandName || !endpoint) {
                return;
            }

            const infoKey = getDualInfoKey(commandKey, side);
            const fetchKey = `${endpoint}::${commandName}`;
            if (lastFetchedRef.current[infoKey] === fetchKey) {
                return;
            }

            try {
                const info = await datasource.getResource(`endpoint/${endpoint}/command/info`, { name: commandName });
                setDualCommandInfos((prev) => ({ ...prev, [infoKey]: info }));
                lastFetchedRef.current[infoKey] = fetchKey;
            } catch {
                setDualCommandInfos((prev) => {
                    const next = { ...prev };
                    delete next[infoKey];
                    return next;
                });
            }
        },
        [datasource]
    );

    const clearDualCommandInfo = useCallback((commandKey: string, side: DualSide) => {
        setDualCommandInfos((prev) => {
            const next = { ...prev };
            const infoKey = getDualInfoKey(commandKey, side);
            delete next[infoKey];
            delete lastFetchedRef.current[infoKey];
            return next;
        });
    }, []);

    return { dualCommandInfos, fetchDualCommandInfo, clearDualCommandInfo };
}
