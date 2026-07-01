export interface PanelOptions {
    enabled: boolean;
    showStatus: boolean;
    onlyWhenRelativeRange: boolean;
    offsetStepMs: number;
    minWriteIntervalMs: number;
    normalizeToNowThresholdMs: number;
}

export const defaultPanelOptions: PanelOptions = {
    enabled: true,
    showStatus: true,
    onlyWhenRelativeRange: true,
    offsetStepMs: 15000,
    minWriteIntervalMs: 10000,
    normalizeToNowThresholdMs: 1500,
};
