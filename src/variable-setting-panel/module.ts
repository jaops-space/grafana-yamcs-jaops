import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from 'commanding-panel/types';
import VariableSettingPanel from './components/VariableSettingPanel';

export const plugin = new PanelPlugin<PanelOptions>(VariableSettingPanel).setNoPadding();
