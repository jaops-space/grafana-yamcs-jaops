import { PanelPlugin } from '@grafana/data';
import CommandHistoryPanel from './components/CommandHistoryPanel';

export interface CommandHistoryOptions {
  visibleFields: string[];
  showArguments: boolean;
  pagination: boolean;
  pageSize: number;
}

const allFields = [
  'time',
  'command',
  'comment',
  'queued',
  'released',
  'sent',
  'extraAcks',
  'completion',
];

export const plugin = new PanelPlugin<CommandHistoryOptions>(CommandHistoryPanel).setPanelOptions(builder => {
  return builder
    .addMultiSelect({
      path: 'visibleFields',
      name: 'Visible Columns',
      description: 'Select which fields to display in the table',
      defaultValue: allFields,
      settings: {
        options: allFields.map(field => ({
          value: field as any,
          label: field.charAt(0).toUpperCase() + field.slice(1),
        })),
      },
    })
    .addBooleanSwitch({
      path: 'showArguments',
      name: 'Show Arguments on Expand',
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

