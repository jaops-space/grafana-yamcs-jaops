## Local plugin development

Install Docker Compose 2.24.4 or newer (required for the canonical port override), Go matching [`go.mod`](../go.mod) (currently 1.27.3), Node matching [`.nvmrc`](../.nvmrc) (Node 22 on this branch), and Corepack/pnpm matching [`package.json`](../package.json). On Windows, use WSL2 with Docker integration. The Node 24/signing upgrade is maintained separately on `chore/node-24`; do not change this branch's toolchain just to run the demo.

From the plugin repository root:

```bash
corepack enable
pnpm install --frozen-lockfile
go mod download
mage -v buildAll
pnpm run build
```

`mage` can be installed following the [Mage instructions](https://magefile.org/); `go run github.com/magefile/mage -v buildAll` also builds without a global Mage command. The backend build must include a Linux binary for the Docker host's architecture. After changing frontend code, rebuild with `pnpm run build` (or use `pnpm run dev` to watch it); after changing backend code, rebuild the Linux binary and restart Grafana.

To run **only Grafana**, `pnpm run server` serves it at `http://localhost:3000/`. It does not start Yamcs or the simulator. The canonical dashboard below uses `/grafana` URLs for its local assets, so use the subpath mode for that demo.

## Run the provisioned canonical demo

The [canonical demo fixture](https://github.com/jaops-space/grafana-yamcs-demo/tree/chore/yamcs-5.13.7) supplies Yamcs 5.13.7 and **one** simulator process for telemetry, local feature parameters, events and generated images. The branch `chore/yamcs-5.13.7` contains the `/myproject`, `/features` and `/images` MDB expected by this dashboard. Clone it beside this repository if it is not already there:

```bash
git clone --branch chore/yamcs-5.13.7 \
  https://github.com/jaops-space/grafana-yamcs-demo.git ../grafana-yamcs-demo
```

From this repository root, start Yamcs on `localhost:8093` (the port configured in [`provisioning/datasources/yamcs-jaops.yaml`](../provisioning/datasources/yamcs-jaops.yaml)) and its simulator/image server on `localhost:8091`:

```bash
docker compose -f ../grafana-yamcs-demo/docker/docker-compose.yml \
  -f scripts/benchmarks/canonical-ports.yml up -d --build
curl -fsS http://localhost:8093/api/instances/demo
```

In another terminal, from this repository root, run the locally built plugin in Grafana at the subpath expected by the dashboard:

```bash
pnpm run server:nginx
```

Open `http://localhost:3000/grafana/d/0708823c-c106-407b-9861-f64283746167` to view **Yamcs Canonical Demo**. Grafana provisions its dashboard JSON from [`provisioning/dashboards`](../provisioning/dashboards) and the `demo_realtime` endpoint from [`provisioning/datasources`](../provisioning/datasources); no manual JSON import is required. Grafana stores UI edits in its database, **not** in the tracked JSON: export intended dashboard changes back into the repository.

The [drone quickstart](https://github.com/jaops-space/grafana-yamcs-quickstart-drone) is a separate, self-contained stack with its own Grafana, datasource, dashboard, simulator and image generator. Its Grafana also binds port 3000, so stop this plugin's Grafana before starting the drone stack. The drone stack remains pinned to released plugin v1.1.6 until a new release is available; it does not use this repository's unreleased build.

## Other commands you can run

### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins/grafana-plugin-sdk-for-go) dependency to the latest minor version:

    ```bash
    go get -u github.com/grafana/grafana-plugin-sdk-go
    go mod tidy
    ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

    ```bash
    mage -v
    ```

3. List all available Mage targets for additional commands:

    ```bash
    mage -l
    ```

4. Spin up a Grafana instance and run the plugin back-end inside it in dev mode

    ```bash
    pnpm run server:dev
    ```

### Frontend

1. Install dependencies

    ```bash
    pnpm install
    ```

2. Build plugin front-end in development mode and run in watch mode

    ```bash
    pnpm run dev
    ```

3. Build plugin in production mode

    ```bash
    pnpm run build
    ```

### Other

4. Run the tests (using Jest)

    ```bash
    # Runs the tests and watches for changes, requires git init first
    pnpm run test

    # Exits after running all the tests
    pnpm run test:ci
    ```

5. Run the E2E tests (using Playwright)

    ```bash
    # Start Grafana separately before running the browser tests
    pnpm run server

    # Run plugin E2E tests against that Grafana
    pnpm run e2e
    ```

6. Run the linter

    ```bash
    pnpm run lint

    # or

    pnpm run lint:fix
    ```
