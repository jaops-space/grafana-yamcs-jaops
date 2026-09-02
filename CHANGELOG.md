# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project follows [Semantic Versioning].

## [Unreleased]

## [1.1.0] - 2026-08-20

### Added

- Added the Yamcs Alarms panel for live Yamcs alarm monitoring, including parameter and event alarms, severity/status display, expandable details, and acknowledge, clear, shelve, and unshelve actions.
- Added the Yamcs Links panel for Yamcs link monitoring and control, including enable, disable, reset counters, custom link actions, filtering, and activity indicators.
- Added the Yamcs Time Sync panel used to synchronize Grafana dashboard time ranges with Yamcs processor time for simulations and replays.
- Added configurable command-button variables, dual-command button support, richer command argument handling, and reusable command-panel components/hooks.
- Added "Expired" notice to panels when Yamcs parameter data has expired.
- Added backend unit tests testing specific utility functions.
- Added Yamcs integration tests for websocket connection, search methods and authentication.
- Added performance benchmark CI reporting with plots, pull-request base comparison, optional long-term baseline comparison.
- Added automatic color toggle for discrete parameter types.
- Added more robust tutorials and on-demand documentation.

### Changed

- Improved datasource configuration diagnostics with host and endpoint health details.
- Reworked the commanding panel into smaller components with clearer editor sections, command selectors, layout controls, button styling controls, validation, and runtime variable handling.
- Renamed the old variable-commanding panel to the variable-setting panel to better match its purpose.
- Updated the datasource query editor with clearer query categories, query help text, and simplified parameter aggregate options.
- Changed Grafana Live stream path construction to use deterministic path segments and include the query shape where needed, such as time range, max data points, min/max fields.
- Refactored the backend source layer from a single endpoint/connection manager into focused host and endpoint handlers for alarms, command history, events, links, parameters, and time.
- Refactored Yamcs client WebSocket and HTTP handling for better context propagation, state checks, request cancellation, and subscription management.
- Improved datasource health checks so invalid hosts/endpoints are reported explicitly instead of failing generically.
- Updated frontend dependencies, backend dependencies, pnpm, ESLint, TypeScript, Webpack, and Grafana package versions.
- Updated Docker, nginx, supervisor, development, and CI configuration for the newer toolchain.
- Reorganized documentation under `docs/`, refreshed README and plugin documentation, updated screenshots/logos, and clarified setup, testing, signing, submission, provisioning, and benchmarking instructions.
- Centralized per-host connection management and WebSocket stream health checks instead of duplicating them per endpoint.
- Replaced per-stream value buffers/channels with a single shared ring buffer per parameter and a lightweight per-stream read cursor, removing lock contention and reducing allocations under many concurrent panels.
- Optimized per-parameter-value processing to avoid rebuilding a full listener slice on every incoming value.

### Fixed

- Fixed parameter query re-rendering behavior and simplified aggregate querying.
- Fixed backend validation and error reporting around malformed datasource configuration.
- Fixed static image rendering safety and image-panel rendering edge cases found during reviewer/security work.
- Fixed alarms, links, and command-history panels not updating live after their initial load.
- Fixed error handling in the Links stream and added Links integration tests.
- Fixed a data race that could crash the alarms stream, and added detection for stale connections.
- Fixed retry-storm timeouts and zero-point-count validation errors during parameter queries.
- Fixed data races in WebSocket connection state and subscription listener wiring.
- Fixed unsynchronized concurrent access to Yamcs client subscription maps.
- Fixed stale processor snapshots and connection/ordering bugs in per-host connection setup.
- Fixed the datasource holding a shared HTTP query object across concurrent requests, which could leak one request's parameters into another's.

### Security

- Added Dependabot configuration for GitHub Actions, Go modules, npm/pnpm dependencies, and OSV scanning.
- Added security scanning CI reporting detected vulnerabilities.
- Hardened resource proxy behavior and static-image URL/style sanitization.
- Updated frontend dependency overrides for known vulnerable transitive packages.
- Migrated pnpm dependency overrides from package.json's now-ignored `pnpm` field to `pnpm-workspace.yaml`.
- Patched known vulnerabilities in websocket-driver, brace-expansion, fast-uri, nanoid, postcss, js-yaml, and react-router.
- Updated the Go toolchain and bumped grpc/opentelemetry dependencies to resolve known Go standard-library and module vulnerabilities.

### Removed

- Removed the legacy `connectionmanager` source implementation after replacing it with host/endpoint-oriented source management.

## [1.0.8] - 2026-07-02

### Fixed

- Fixed the plugin metadata to declare the required Grafana dependency as `>=12.1.1`.

## [1.0.7] - 2026-07-02

### Changed

- Reworked the Yamcs Go client and datasource backend paths to propagate Go contexts through HTTP calls, streaming, command execution, resource requests, and paginated iterators.
- Updated plugin submission documentation to validate locally with `-sourceCodeUri`, matching Grafana marketplace CI behavior.

### Fixed

- Improved request cancellation behavior for datasource API, resource, iterator, and subscription paths.

## [1.0.6] - 2026-07-01

### Changed

- Added release documentation notes for gosec static analysis and the Grafana SDK Go-version requirement.

### Security

- Changed Yamcs WebSocket subscription call IDs to `int32` to address gosec integer-overflow findings.

## [1.0.5] - 2026-07-01

### Added

- Added datasource provisioning examples and release/submission documentation for version bumping and tagging.

### Changed

- Updated command-related panel styling to use Grafana styling APIs.
- Updated CI Go version to match the upgraded Grafana plugin SDK requirement.

### Fixed

- Fixed Grafana submission review issues and improved backend Yamcs client error handling.

### Security

- Upgraded Grafana plugin SDK and patched pnpm dependency vulnerabilities, including gRPC/OpenTelemetry-related findings.

## [1.0.4] - 2026-03-02

### Security

- Patched a high-severity `serialize-javascript` vulnerability through dependency overrides.

## [1.0.3] - 2026-03-02

### Changed

- Updated README and catalog publishing documentation.

### Fixed

- Fixed login auto-refresh locking to avoid races while replacing refresh state.
- Fixed a missing newline in generated Webpack plugin type declarations.

### Security

- Sanitized templated static-image URLs and style values in the image renderer.

## [1.0.2] - 2026-02-26

### Security

- Patched high-severity dependency vulnerabilities in the Go and Node dependency trees.

## [1.0.1] - 2026-02-25

### Fixed

- Fixed an HTTP manager auto-refresh goroutine leak.
- Replaced direct DOM manipulation in datasource configuration export with a React ref and object URL cleanup.

## [1.0.0] - 2026-02-11

### Added

- Initial stable release of the JAOPS Grafana Yamcs plugin.
- Added a Yamcs datasource for connecting Grafana dashboards to Yamcs telemetry and command data.
- Added multiplexed endpoint support so multiple Grafana clients can share Yamcs connections efficiently.
- Added telemetry visualization and static image panels for live Yamcs data.
- Added a commanding panel for sending Yamcs commands from Grafana dashboards.
- Added command history and variable-oriented panel support.
- Added app navigation, setup documentation, demo dashboard assets, and contribution guidance.

[Unreleased]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.8...v1.1.0
[1.0.8]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/jaops-space/grafana-yamcs-jaops/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/jaops-space/grafana-yamcs-jaops/releases/tag/v1.0.0
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
