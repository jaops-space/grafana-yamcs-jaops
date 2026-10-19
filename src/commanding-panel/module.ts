import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from './types';
import CommandingPanel from './CommandingPanel';
import { migrateCommandOptions } from './migrateOptions';

export const plugin = new PanelPlugin<PanelOptions>(CommandingPanel)
    .setMigrationHandler(migrateCommandOptions)
    .setNoPadding();
