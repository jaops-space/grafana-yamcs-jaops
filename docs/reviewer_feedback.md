Feedback from Grafana review team. This feedback must be remembered and regressions avoided.

Required changes:

1. Frontend logging: Remove console.log statements
2. Backend logging: Replace usage of .info with .debug
3. When creating HTTP Clients (like in yamcs/core/http/http.go), please use the HTTP client provided by the Grafana plugin SDK. This will auto apply some recommended settings like timeouts and middlewares. Also, make sure to store these clients in the Datasource or App instance so connections can be reused for different queries.
4. Avoid using global variables like GlobalMultiplexer. This may cause that different instances of the plugin can collide with each other in some scenarios.
5. Auto-refresh goroutine leak
   The HTTPManager.Login method starts a new auto-refresh goroutine and replaces the RefreshStop channel without stopping any previously running loop. There’s also no cleanup during datasource/connection disposal. This can lead to leaked goroutines and tickers over time. Please ensure that: Any existing refresh loop is properly stopped before starting a new one (e.g., via context.WithCancel or safely closing the previous channel). The refresh loop is explicitly stopped during plugin shutdown (Datasource/ConnectionManager dispose path).
6. Direct DOM usage in ConfigEditor: The exportConfig function directly manipulates the global document to create and click an anchor element. This should be refactored to use a React ref (e.g., a hidden \<a> element in the component) and handle object URL cleanup through the component lifecycle.
7. The catalog displays the README from the src/ folder, not the root one. Your root README is mostly good for the catalog, but remove the "try it out plugin for yourself" section.
8. XSS Security Vulnerability in ImageRenderer. You're constructing HTML directly in JavaScript and using template variables directly in the src attribute without any sanitization. This is a security issue because template variables should be treated as user input - they can contain malicious JavaScript that could execute in dashboards. Locations:
   Main vulnerability: https://github.com/jaops-space/grafana-yamcs-jaops/blob/main/src/static-image-panel/components/ImageRenderer.tsx#L8-L23 Used by ImagePanel: https://github.com/jaops-space/grafana-yamcs-jaops/blob/main/src/image-panel/components/ImagePanel.tsx#L19 Used by StaticImagePanel: https://github.com/jaops-space/grafana-yamcs-jaops/blob/main/src/static-image-panel/components/StaticImagePanel.tsx#L6. Fix: Use DOMPurify to sanitize all user input, including template variables, before using them in HTML construction.
