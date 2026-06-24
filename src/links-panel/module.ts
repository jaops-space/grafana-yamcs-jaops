import { PanelPlugin } from '@grafana/data';
import { LinksPanel } from './components/LinksPanel';
import { PanelOptions } from './types';

export const plugin = new PanelPlugin<PanelOptions>(LinksPanel)
  .setPanelOptions((builder) => {
    return builder
      .addBooleanSwitch({
        path: 'showDetails',
        name: 'Show details',
        description: 'Show detailed link information (type, class, data counts)',
        defaultValue: true,
      })
      .addTextInput({
        path: 'nameFilter',
        name: 'Filter by name',
        description: 'Regex pattern to filter links by name (leave empty for all)',
        defaultValue: '',
      });
  });
