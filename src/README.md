<p align="center">
  <img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/img/logo.png" alt="Grafana Yamcs JAOPS logo" width="120" height="120">
</p>

<h1 align="center">Grafana Plugin for the Yamcs Mission Control Software</h1>

<p align="center">
  A comprehensive Grafana plugin to connect to <a href="https://yamcs.org/">Yamcs</a> servers for real-time telemetry visualization and commanding capabilities. Designed for mission control centers, aerospace applications and beyond.
</p>

<p align="center">
  <a href="https://github.com/jaops-space/grafana-yamcs-jaops/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/jaops-space/grafana-yamcs-jaops/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/jaops-space/grafana-yamcs-jaops/actions/workflows/benchmark.yml"><img alt="Benchmark" src="https://github.com/jaops-space/grafana-yamcs-jaops/actions/workflows/benchmark.yml/badge.svg"></a>
  <a href="https://grafana.com/grafana/plugins/jaops-yamcs-app/"><img alt="Grafana catalog version" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.version&label=version&color=F46800&logo=grafana"></a>
  <a href="https://grafana.com/grafana/plugins/jaops-yamcs-app/"><img alt="Downloads" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.downloads&label=downloads&color=5794F2&logo=grafana"></a>
  <a href="https://github.com/jaops-space/grafana-yamcs-jaops/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/badge/license-MIT-2EA043"></a>
</p>

## Overview

A plugin that connects Grafana directly to the Yamcs server to display telemetry, send commands, inspect command history, monitor alarms, control Yamcs links, and synchronize dashboards with Yamcs processor time.

This plugin is engineered for high reliability to be used in Mission Control Centers and anywhere Yamcs is used.

The current version has already been tested in real-world deployments but active development continues and community feedback and contributions are very welcome.

Development is led by [JAOPS](https://www.jaops.com/): providing Mission Control software, tools and training for spacecraft in orbit and rovers on the Moon!

### Install and Try

Install the signed plugin directly from the [Grafana Catalog](https://grafana.com/grafana/plugins/jaops-yamcs-app/) (**Plugins and data** > **Plugins** on your Grafana Instnace). After installation, configure a Yamcs host and endpoint in the datasource settings.

The plugin includes helpful tutorials for each panel. Access them from the main navigation menu on the left side of Grafana.

<table>
  <tr>
    <td><strong>Grafana Catalog</strong></td>
    <td><a href="https://grafana.com/grafana/plugins/jaops-yamcs-app/">jaops-yamcs-app</a></td>
  </tr>
  <tr>
    <td><strong>Latest catalog version</strong></td>
    <td><img alt="Latest catalog version" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.version&label=version&color=F46800&logo=grafana"></td>
  </tr>
  <tr>
    <td><strong>Status</strong></td>
    <td><img alt="Catalog status" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.status&label=status&color=2EA043&logo=grafana"></td>
  </tr>
  <tr>
    <td><strong>Signature</strong></td>
    <td>
      <img alt="Signature type" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.signatureType&label=signature&color=7B61FF&logo=grafana">
      <img alt="Signed by" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.versionSignedByOrgName&label=signed%20by&color=7B61FF&logo=grafana">
    </td>
  </tr>
  <tr>
    <td><strong>Grafana dependency</strong></td>
    <td><img alt="Grafana dependency" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.grafanaDependency&label=grafana&color=F46800&logo=grafana"></td>
  </tr>
  <tr>
    <td><strong>Catalog downloads</strong></td>
    <td><img alt="Catalog downloads" src="https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgrafana.com%2Fapi%2Fplugins%2Fjaops-yamcs-app%3Fversion%3Dlatest&query=%24.downloads&label=downloads&color=5794F2&logo=grafana"></td>
  </tr>
</table>

## Plugin Parts

<table>
  <tr>
    <th width="80">Icon</th>
    <th>Plugin Part</th>
    <th>Capability</th>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/datasource/img/logo-small.png" alt="Datasource icon" width="42" height="42"></td>
    <td><strong>Yamcs Datasource</strong></td>
    <td>Connects Grafana to Yamcs endpoints, queries telemetry, streams realtime values, exposes backend resources, and reports endpoint health.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/commanding-panel/img/logo-small.png" alt="Commanding icon" width="42" height="42"></td>
    <td><strong>Commanding Panel</strong></td>
    <td>Issues commands via a Grafana panel with fully customizable buttons, supporting arguments, comments, and endpoint targeting.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/command-history-panel/img/logo-small.png" alt="Command history icon" width="42" height="42"></td>
    <td><strong>Command History Panel</strong></td>
    <td>Keeps track of commands sent, arguments, acknowledgements, and command execution state.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/image-panel/img/logo-small.png" alt="Telemetric image icon" width="42" height="42"></td>
    <td><strong>Telemetric Image Panel</strong></td>
    <td>Visualizes realtime image data from Yamcs telemetry.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/static-image-panel/img/logo-small.png" alt="Static image icon" width="42" height="42"></td>
    <td><strong>Static Image Panel</strong></td>
    <td>Overlays telemetry data on static images such as spacecraft layouts, maps, or operator schematics.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/variable-setting-panel/img/logo-small.png" alt="Variables icon" width="42" height="42"></td>
    <td><strong>Variable Setting Panel</strong></td>
    <td>Provides operator-friendly dashboard variable controls for commanding and filtering workflows.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/alarms-panel/img/logo-small.png" alt="Alarms icon" width="42" height="42"></td>
    <td><strong>Alarms Panel</strong></td>
    <td>Displays active Yamcs alarms and supports alarm-focused operations from Grafana.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/links-panel/img/logo-small.png" alt="Links icon" width="42" height="42"></td>
    <td><strong>Links Panel</strong></td>
    <td>Displays Yamcs data links and lets operators enable or disable them from Grafana.</td>
  </tr>
  <tr>
    <td align="center"><img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/src/time-sync-panel/img/logo-small.png" alt="Time sync icon" width="42" height="42"></td>
    <td><strong>Yamcs Time Sync</strong></td>
    <td>Visualizes replays and future simulations through synchronization between Grafana and the Yamcs clock.</td>
  </tr>
</table>

## Features

- **Multiplexed Endpoint Support** - Designed to handle complex setups with multiple Yamcs endpoints through a robust multiplexer system. Supports scaling to many Grafana clients efficiently by multiplexing the connections to Yamcs: the same data is only requested once.

- **Modular and Scalable Architecture** - Clean separation of concerns and a solid backend structure built for reliability and flexibility.

- **Telemetry & Static Image Panel** - Visualize real-time telemetry data or overlay data on static images, such as spacecraft layouts and maps.

- **Commanding Panel** - Issue commands via a Grafana panel with fully customizable buttons, supporting arguments, comments, and endpoint targeting.

- **Intuitive UI/UX** - Clean and simple user interface designed to be easy to use, even for non-experts.

- **Live Status Feedback** - Displays endpoint availability and WebSocket status in real-time, ensuring quick diagnostics.

- **Fully Configurable** - Every aspect of the plugin, from endpoint configuration to command structure and visual layout, is configurable through Grafana's settings.

- **Replay/Simulation support** - Visualize replays and future simulations through synchronization between Grafana and Yamcs clock.

<p align="center">
  <img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/screenshots/DesignDocument.png" alt="Design document" width="900">
</p>

## Example Grafana Dashboard Connected to Yamcs

Demo dashboards showcase the main capabilities of the plugin. They are made to use data from the [Yamcs Quickstart](https://github.com/jaops-space/yamcs-quickstart).

To try it with a local Yamcs demo, clone the [JAOPS Yamcs Quickstart](https://github.com/jaops-space/yamcs-quickstart) repository and run:

```bash
./mvn yamcs:run
```

In another terminal, start the simulator:

```bash
python3 simulator.py
```

For image telemetry panels, you can also run the optional image simulator:

```bash
python3 simulator/images/generate_images.py
```

For local development with the repository demo dashboards, follow the [setup instructions](https://github.com/jaops-space/grafana-yamcs-jaops/blob/main/docs/SETUP_INSTRUCTIONS.md).

<p align="center">
  <img src="https://raw.githubusercontent.com/jaops-space/grafana-yamcs-jaops/main/screenshots/full_demo_dash.png" alt="Grafana dashboard connected to Yamcs" width="900">
</p>

## Acknowledgements

Since October 2024, the plugin has been tested and improved with feedback from the Space Robotics Lab of Tohoku University in Sendai, Japan.

## Documentation and Contributing

Check out the open-source [GitHub repository](https://github.com/jaops-space/grafana-yamcs-jaops) for the latest features and documentation. 
Community feedback and contributions are very welcome!

If you find a bug, have a feature request, or want to improve the project, feel free to [open an issue](https://github.com/jaops-space/grafana-yamcs-jaops/issues) or submit a pull request.
