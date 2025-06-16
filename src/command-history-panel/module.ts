import { PanelPlugin } from '@grafana/data';
import CommandHistoryPanel from './components/CommandHistoryPanel';

export const plugin = new PanelPlugin<{}>(CommandHistoryPanel);
