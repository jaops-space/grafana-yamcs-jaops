import { useEffect, useState } from 'react';
import { DataSourceWithBackend, getDataSourceSrv } from '@grafana/runtime';

export function useDatasource(datasourceUid?: string) {
    const [datasource, setDatasource] = useState<DataSourceWithBackend | null>(null);

    useEffect(() => {
        if (!datasourceUid) {
            setDatasource(null);
            return;
        }

        getDataSourceSrv()
            .get(datasourceUid)
            .then((ds) => setDatasource(ds as DataSourceWithBackend))
            .catch(() => setDatasource(null));
    }, [datasourceUid]);

    return datasource;
}
