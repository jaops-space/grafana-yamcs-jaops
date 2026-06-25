# Provisoning

Grafana has an active provisioning system that uses configuration files. You can define data sources and dashboards to be automatically set up by Grafana.

Reference: [Provision Grafana](https://grafana.com/docs/grafana/latest/administration/provisioning/)

## Folder structure

Grafana documents these provisioning folders and responsibilities:

1. `provisioning/datasources`
   Each YAML file contains datasource configuration.
   Grafana expects datasource definitions under the top-level key: datasources.
   Optional top-level keys include `deleteDatasources` and `prune`.

2. `provisioning/dashboards`
   Each YAML file contains dashboard providers.
   Grafana expects provider definitions under the top-level key: providers.
   Providers point to dashboard files on disk (for example via options.path).

3. `provisioning/plugins`
   Each YAML file contains plugin application configuration.
   Grafana expects plugin app entries under the top-level key: apps.
   This provisions plugin configuration, not plugin installation.

4. `provisioning/alerting`
   Used for Grafana Alerting provisioning resources.
   See:
   [Provision Grafana Alerting resources](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/)

### Datasource UIDs

By default, Grafana generates UIDs for datasources to be used reliably within dashboards, and that happens if you omit the `uid` in the `.yaml` file.

If you do that and want to provision a dashboard (or use it in another instance), do not manually copy its JSON, that will require some manual fixing and might add overhead on complex dashboards.

Instead, when outside of Edit Mode in a dashboard (you can exit using `Exit Edit`) then click on `Export` > `Export as JSON` and tick `Export the dashboard to use in another instance`

> [!NOTE]
> If you manually set a `uid` in the `.yaml` file, the `uid` chosen will be used and won't cause any issues.

## In this repository

- App provisioning file: [provisioning/plugins/apps.yaml](provisioning/plugins/apps.yaml)
- Datasource instances folder: [provisioning/datasources](provisioning/datasources)
- Dashboard provider file: [provisioning/dashboards/yamcs-dashboards.yaml](provisioning/dashboards/yamcs-dashboards.yaml)
