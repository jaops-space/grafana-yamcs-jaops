import { useMemo, useState } from 'react';
import { PanelOptions } from 'commanding-panel/types';
import { DualButtonStates } from '../types';

export function useDualButtonStates(
    panelId: number,
    options: PanelOptions,
    onOptionsChange: (options: PanelOptions) => void
) {
    const storageKey = useMemo(() => `jaops-yamcs-app.commanding-panel-state-${panelId}`, [panelId]);

    const [dualButtonStates, setDualButtonStates] = useState<DualButtonStates>(() => {
        try {
            const stored = localStorage.getItem(storageKey);
            return stored ? JSON.parse(stored) : options.dualButtonStates || {};
        } catch {
            return options.dualButtonStates || {};
        }
    });

    const updateDualButtonStates = (newStates: DualButtonStates) => {
        setDualButtonStates(newStates);
        localStorage.setItem(storageKey, JSON.stringify(newStates));
        onOptionsChange({ ...options, dualButtonStates: newStates });
    };

    return { dualButtonStates, updateDualButtonStates };
}
