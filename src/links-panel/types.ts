export interface PanelOptions {
  /** Refresh interval in seconds (0 = manual only) */
  refreshInterval: number;
  /** Show detailed link information */
  showDetails: boolean;
  /** Filter by link name (regex pattern) */
  nameFilter: string;
}

export interface LinkInfo {
  name: string;
  type?: string;
  class?: string;
  disabled?: boolean;
  status?: string;
  dataInCount?: string;
  dataOutCount?: string;
  detailedStatus?: string;
  parentName?: string;
  actions?: LinkAction[];
  extra?: Record<string, unknown>;
}

export interface LinkAction {
  id: string;
  label: string;
  style?: string;
  enabled?: boolean;
  checked?: boolean;
  spec?: Record<string, unknown>;
}
