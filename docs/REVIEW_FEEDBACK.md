# Grafana reviewer feedback

Use these recurring review lessons when changing the plugin or preparing a release.

- **Protect saved dashboards.** Treat every published panel ID, query format, stream address, and option key as a persistent interface. Test upgrades with existing dashboards, including those created by intermediate releases. Option migrations preserve settings only after a panel loads; an ID change requires continued support or a major release with migration guidance.

- **Test behavior in Grafana.** Healthy connection checks and mocks cannot prove that dashboards render. Exercise real-time queries, Live channel addresses, and stream subscriptions against a provisioned data source on supported Grafana versions; include representative old dashboards and focused regression tests.

- **Keep diagnostics safe.** Health details, resource responses, and connection-test summaries are user-visible. Map failures to fixed, actionable categories rather than exposing raw transport errors, internal addresses, resolvers, credentials, or upstream responses. Log original errors with diagnostic context on the server.

- **Authorize actions server-side.** Query access must not imply permission to issue commands or operate alarms and links. Gate every mutating resource route with an explicit role or permission, deny missing identities, and check Viewer versus authorized behavior; hiding UI controls is not authorization.

- **Preserve connection configuration.** Build outbound clients from Grafana SDK HTTP options so proxies, headers, TLS, timeouts, and middleware survive refactors. Reuse clients where appropriate, isolate mutable per-host settings, and never modify shared defaults such as a global WebSocket dialer.

- **Handle untrusted data at boundaries.** Render template variables and remote values as data, not HTML; sanitize unavoidable HTML and restrict URL schemes. Escape dynamic API path segments individually, especially names containing slashes, rather than interpolating them into URLs.

- **Own and release resources.** Keep connection state per data source instance, not in globals. Stop refresh goroutines and tickers when replaced or disposed; defer temporary-client cleanup immediately. Use component refs instead of global DOM manipulation and revoke temporary object URLs.

- **Log deliberately.** Use the Grafana SDK logger with structured fields, not backend print statements or stray frontend console logs. Keep per-entry command history and other user or operational data out of routine Info logs; Debug is not a safe place for secrets.

- **Validate the release artifact.** Run Grafana's validator and dependency/source scans on the actual release inputs, including lockfiles and the Go standard library. Align Go versions across the module, CI, build, and scanners; investigate scanner/toolchain mismatches rather than treating failed scans as clean. Separate blockers from suggestions.

- **Review the published presentation.** Check the README Grafana displays, deduplicate panel metadata, and keep documentation and screenshots accurate. Prefer Grafana theme tokens over fixed styles; distinguish optional catalog suggestions, such as sponsorship links, from publication blockers.
