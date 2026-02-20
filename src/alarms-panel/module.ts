import { PanelPlugin } from '@grafana/data';
import AlarmsPanel from './components/AlarmsPanel';

export interface AlarmsOptions {
  visibleFields: string[];
  showDetails: boolean;
  pagination: boolean;
  pageSize: number;
}


// Default columns to match Yamcs Web order
const allFields = [
  'state',            // State
  'severity',         // Severity
  'triggerTime',      // Alarm time
  'triggerTimestamp', // Trigger Timestamp
  'name',             // Alarm name (Parameter)
  'type',             // Alarm type
  'triggerValue',     // Trigger value
  'mostSevereValue',  // Most severe value
  'currentValue',     // Live value
  'violations',       // Violations
  'acknowledged',     // Ack
  'actions',          // Actions
];

// Default visible columns for new panels (matching Yamcs Web)
const defaultVisibleFields = [
  'severity',         // Severity
  'triggerTime',      // Alarm time
  'name',             // Alarm name
  'type',             // Alarm type
  'triggerValue',     // Trigger value
  'mostSevereValue',  // Most severe value
  'currentValue',     // Live value
  'actions',          // Actions
];

// Field labels for better UX
const fieldLabels: Record<string, string> = {
  state: 'State',
  severity: 'Severity',
  triggerTime: 'Alarm time',
  triggerTimestamp: 'Trigger Timestamp',
  name: 'Alarm name',
  type: 'Alarm type',
  triggerValue: 'Trigger value',
  mostSevereValue: 'Most severe value',
  currentValue: 'Live value',
  violations: 'Violations',
  acknowledged: 'Ack',
  actions: 'Actions',
};

export const plugin = new PanelPlugin<AlarmsOptions>(AlarmsPanel).setPanelOptions(builder => {
  return builder
    .addMultiSelect({
      path: 'visibleFields',
      name: 'Visible Columns',
      description: 'Select which fields to display in the table',
      defaultValue: defaultVisibleFields,
      settings: {
        options: allFields.map(field => ({
          value: field as any,
          label: fieldLabels[field] || field,
        })),
      },
    })
    .addBooleanSwitch({
      path: 'showDetails',
      name: 'Show Details on Expand',
      defaultValue: true,
    })
    .addBooleanSwitch({
      path: 'pagination',
      name: 'Enable Pagination',
      defaultValue: false,
    })
    .addNumberInput({
      path: 'pageSize',
      name: 'Page Size',
      defaultValue: 10,
      settings: {
        min: 1,
        max: 100,
      },
      showIf: (config) => config.pagination,
    });
});
