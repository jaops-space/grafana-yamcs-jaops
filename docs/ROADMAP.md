# Roadmap

## Authorization for Yamcs actions

Current behavior is unchanged: anyone who can use the data source, including Grafana Viewers, can invoke telecommand, alarm, and link actions. The actions execute through the data source's configured Yamcs credentials; the backend does not currently enforce a Grafana role. This keeps shared and view-oriented dashboards working, but means dashboard access can include operational control.

Revisit whether to support a configurable minimum Grafana role or separate permissions for command issuance and alarm/link actions. Any change should make the effective permissions clear to dashboard users, account for shared dashboards, and preserve an explicit compatibility path for existing deployments. No role restriction is being introduced in this release.
