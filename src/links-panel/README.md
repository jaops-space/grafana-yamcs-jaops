# Yamcs Link Management

Adds the ability to view and control Yamcs data links directly from Grafana dashboards. Related to [Issue #7](https://github.com/jaops-space/grafana-yamcs-jaops/issues/7).

## What it does

A new **JAOPS Links Panel** lets operators see all Yamcs links for a given endpoint and enable, disable, or reset their counters with a click. The panel polls the backend over REST on a configurable interval (default 5 s), it does not use the WebSocket streaming pipeline that parameter queries use.

## How it works

The panel reads the datasource UID and endpoint from its query configuration (`data.request.targets`). Because links don't need streaming, the datasource returns an empty observable for `LINKS` queries to preserve the request metadata:

```typescript
if (query.type === QueryType.LINKS) {
    return new Observable<DataQueryResponse>((subscriber) => {
        subscriber.next({ data: [], state: LoadingState.Done });
        subscriber.complete();
    });
}
```

The panel then fetches data directly using `getBackendSrv()`:

```typescript
const url = `/api/datasources/uid/${dsUid}/resources/endpoint/${endpoint}/links`;
const result = await getBackendSrv().get(url);
```

Dashboard variables are supported: when "As variable" is checked, the endpoint is resolved via `getTemplateSrv().replace(target.endpointVariable)`.

## Backend

Six new resource routes are registered in `resources.go`, backed by corresponding client methods in `link_endpoints.go` that call the Yamcs REST API using protobuf:

```
GET  /endpoint/{endpointID}/links                        — list all links
GET  /endpoint/{endpointID}/links/{linkName}              — get a single link
POST /endpoint/{endpointID}/links/{linkName}/enable       — enable
POST /endpoint/{endpointID}/links/{linkName}/disable      — disable
POST /endpoint/{endpointID}/links/{linkName}/reset        — reset counters
POST /endpoint/{endpointID}/links/{linkName}/action/{id}  — run a link action
```

## Panel options

- **Auto-refresh interval**: polling frequency in seconds (0 = manual only, default 5)
- **Show details**: show link type and data in/out counters (default true)
- **Filter by name**: regex to filter which links are displayed

## Status badges

- 🟢 Green: link is OK
- 🟠 Orange: link is disabled
- 🔴 Red: link has failed or status is not OK