import { PanelPlugin } from '@grafana/data';
import { defaultPanelOptions, PanelOptions } from './types';
import { TimeSyncPanel } from './TimeSyncPanel';

export const plugin = new PanelPlugin<PanelOptions>(TimeSyncPanel).setNoPadding().setPanelOptions((builder) =>
    builder
        .addBooleanSwitch({
            path: 'enabled',
            name: 'Enable Yamcs time sync',
            defaultValue: defaultPanelOptions.enabled,
        })
        .addBooleanSwitch({
            path: 'showStatus',
            name: 'Show status card',
            defaultValue: defaultPanelOptions.showStatus,
        })
        .addBooleanSwitch({
            path: 'onlyWhenRelativeRange',
            name: 'Only apply when range is relative (contains now)',
            defaultValue: defaultPanelOptions.onlyWhenRelativeRange,
        })
        .addNumberInput({
            path: 'offsetStepMs',
            name: 'Offset step (ms)',
            description: 'Quantize Yamcs skew to this step to avoid frequent dashboard range rewrites.',
            defaultValue: defaultPanelOptions.offsetStepMs,
            settings: { min: 1000, max: 300000, step: 1000 },
        })
        .addNumberInput({
            path: 'minWriteIntervalMs',
            name: 'Minimum write interval (ms)',
            description: 'Minimum time between dashboard range rewrites.',
            defaultValue: defaultPanelOptions.minWriteIntervalMs,
            settings: { min: 1000, max: 120000, step: 1000 },
        })
        .addNumberInput({
            path: 'normalizeToNowThresholdMs',
            name: 'Normalize-to-now threshold (ms)',
            description: 'If Yamcs skew magnitude is under this threshold, snap back to plain now.',
            defaultValue: defaultPanelOptions.normalizeToNowThresholdMs,
            settings: { min: 100, max: 10000, step: 100 },
        })
);
