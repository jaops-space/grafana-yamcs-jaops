# Grafana reviewer feedback

Use these recurring review lessons when changing the plugin or preparing a release.

- **Protect saved dashboards.** Treat published panel IDs, query formats, stream addresses, and option keys as persistent interfaces. Test upgrades with existing dashboards; option migration cannot fix an ID Grafana no longer recognizes.
  - Grafana flagged the rename from `jaops-varcommanding-panel` to `jaops-variables-panel`: older dashboards would show "Panel plugin not found." They asked us to keep the ID or make the break a documented major release, and suggested migrating 1.0.8 panel settings.

- **Test behavior in Grafana.** A healthy connection check or mocked unit test does not prove that a dashboard renders; exercise real queries and upgrades on supported Grafana versions.
  - In 1.1.1, parameter graphs failed with an invalid Live channel address although the connection test and parameter search worked. The address used `stream` where Grafana expected `namespace`; mocks missed the regression.

- **Keep diagnostics safe.** Return fixed, useful failure categories from user-visible resources, and keep raw transport details in structured server logs.
  - Grafana found that `/fetch/health-details` showed the config editor raw connection errors, including private IPs and DNS resolver addresses. They requested fixed messages such as "Connection refused" or "Host name could not be resolved."

- **Authorize actions server-side.** Query access must not imply permission to issue commands or control alarms and links; enforce roles on the backend, not just in UI controls.
  - Grafana observed that Viewers could invoke telecommand, alarm, and link action resource routes. We restricted those routes to Editors and Admins and deny requests without a user identity.

- **Preserve connection configuration.** Use Grafana SDK HTTP options, reuse clients appropriately, and keep mutable connection state per instance or host rather than changing shared defaults.
  - Grafana noted that a client refactor dropped secure SOCKS proxy and custom-header settings, while editing `websocket.DefaultDialer` could race between hosts. An earlier review also objected to a global multiplexer.

- **Handle untrusted data at boundaries.** Render variables as data rather than HTML; sanitize unavoidable markup, constrain URL schemes, and escape dynamic API path segments individually.
  - Grafana flagged HTML constructed by `ImageRenderer` with unsanitized template values. They also identified unescaped link names, action IDs, and command names in Yamcs API paths.

- **Own and release resources.** Stop refresh loops when replaced or disposed, defer temporary-client cleanup immediately, and use component refs rather than manipulating the global DOM.
  - Earlier feedback found an auto-refresh goroutine that could outlive its client, a health-check multiplexer disposed without `defer`, and a ConfigEditor export that created an anchor through `document`.

- **Log deliberately.** Use the SDK logger and structured fields, not print or stray console statements; keep sensitive and high-volume operational data out of routine logs.
  - Grafana's validator rejected backend `fmt.Print` logging. Reviewers also called out full command-history entries logged at Info level, which could include arguments, comments, and usernames.

- **Validate the release artifact.** Scan the actual release inputs, including lockfiles and the Go standard library; align build and scanner toolchains and never treat a failed scan as clean.
  - Review checks caught vulnerable dependencies and 11 reachable standard-library vulnerabilities in the Go 1.26.9 build. A separate `govulncheck` run could not load packages because the scanner and Go on PATH used different versions.

- **Review the published presentation.** Inspect the README and metadata Grafana actually displays, keep the panel list accurate, and prefer Grafana theme tokens over fixed styling.
  - Grafana pointed out that the catalog reads the `src/` README, spotted duplicate panel includes, and suggested replacing hard-coded styles. A sponsorship link was optional advice, not a publication blocker.
