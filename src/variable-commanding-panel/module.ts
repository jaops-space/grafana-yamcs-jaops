import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from 'commanding-panel/types';
import VariableCommandingPanel from './components/CommandingPanel';

export const plugin = new PanelPlugin<PanelOptions>(VariableCommandingPanel).setNoPadding();
