export interface PanelOptions {
    enabled: boolean;
    showStatus: boolean;
    onlyWhenRelativeRange: boolean;
    offsetStepMs: number;
    minWriteIntervalMs: number;
    normalizeToNowThresholdMs: number;
    maxAcceptedSkewMs: number;
}

export const defaultPanelOptions: PanelOptions = {
    enabled: true,
    showStatus: true,
    onlyWhenRelativeRange: true,
    offsetStepMs: 15000,
    minWriteIntervalMs: 10000,
    normalizeToNowThresholdMs: 1500,
    maxAcceptedSkewMs: 24 * 60 * 60 * 1000,
};
