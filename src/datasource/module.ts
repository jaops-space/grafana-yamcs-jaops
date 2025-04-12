import { DataSourcePlugin } from '@grafana/data';
import { QueryEditor } from './components/QueryEditor';
import { DataSource } from './datasource';
import { Configuration, Query } from './types';
import ConfigEditor from './config/ConfigEditor';

export const plugin = new DataSourcePlugin<DataSource, Query, Configuration>(DataSource)
    .setQueryEditor(QueryEditor)
    .setConfigEditor(ConfigEditor);
