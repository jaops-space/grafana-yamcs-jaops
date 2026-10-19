import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from 'commanding-panel/types';
import VariableSettingPanel from './components/VariableSettingPanel';
import { migrateCommandOptions } from 'commanding-panel/migrateOptions';

export const plugin = new PanelPlugin<PanelOptions>(VariableSettingPanel)
    .setMigrationHandler(migrateCommandOptions)
    .setNoPadding();
