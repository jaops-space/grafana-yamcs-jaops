import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from './types';
import CommandingPanel from './components/CommandingPanel';

export const plugin = new PanelPlugin<PanelOptions>(CommandingPanel).setNoPadding();
