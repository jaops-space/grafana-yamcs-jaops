import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import { PLUGIN_BASE_URL } from '../constants';

// Prefixes the route with the base URL of the plugin
export function prefixRoute(route: string): string {
  return `${PLUGIN_BASE_URL}/${route}`;
}

export async function getDatasourceResource(uid: string, path: string) {
    const response = getBackendSrv().fetch({
        url: `/api/datasources/uid/${uid}/resources/${path}`,
        method: 'GET',
    });

    return lastValueFrom(response);
}

export async function postDatasourceResource(uid: string, path: string, data: any) {
    const response = getBackendSrv().fetch({
        url: `/api/datasources/uid/${uid}/resources/${path}`,
        method: 'POST',
        data
    });

    return lastValueFrom(response);
}
