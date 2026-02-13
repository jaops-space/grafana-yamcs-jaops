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
  'state',         // State
  'severity',      // Severity
  'triggerTime',   // Alarm time
  'name',          // Alarm name (Parameter)
  'type',          // Alarm type
  'triggerValue',  // Trip value
  'currentValue',  // Live value
  'violations',    // Violations
  'acknowledged',  // Ack
  'processOK',     // Status
  'actions',       // Actions
];

export const plugin = new PanelPlugin<AlarmsOptions>(AlarmsPanel).setPanelOptions(builder => {
  return builder
    .addMultiSelect({
      path: 'visibleFields',
      name: 'Visible Columns',
      description: 'Select which fields to display in the table',
      defaultValue: allFields,
      settings: {
        options: allFields.map(field => ({
          value: field as any,
          label: field.charAt(0).toUpperCase() + field.slice(1).replace(/([A-Z])/g, ' $1'),
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
