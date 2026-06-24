export interface PanelOptions {
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
  dataInCount?: number | string;
  dataOutCount?: number | string;
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
