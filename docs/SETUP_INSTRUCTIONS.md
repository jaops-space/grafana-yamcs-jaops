## Try out the plugin for yourself

Local installation:

- use to contribute to the development of the plugin
- use while it is not officially available from the Grafana marketplace (in progress)
### Pre-requisites

- Docker

```bash
# Install Docker using the official convenience script
curl -fsSL https://get.docker.com | sudo sh

# Allow running docker without sudo
sudo usermod -aG docker $USER
newgrp docker
```

> [!IMPORTANT]  
> If you want to set-up the plugin in a Windows environment, you will need to run the plugin inside WSL2, so you will need WSL2 installed, and have the tools above all installed in the WSL2 environment

### Set up and run the plugin easily (with pixi)

[pixi](https://pixi.sh) installs Node.js, pnpm and Go for you

```bash
# Build the front-end and the back-end. Pixi skips a build when its source did not change.
pixi run build

# Build, then start Grafana (same as pnpm run server)
pixi run server
```

Note: `pixi run server-dev` is the same as `pnpm run server:dev`. 
See `pixi.toml` for details.


### Set up the plugin (manually)
For power users / when not using pixi:

1. Install back-end and front-end dependencies using the following:

    ```bash
    go mod download
    pnpm install
    ```

2. Compile the back-end using Mage:

    ```bash
    mage build:backend
    ```

3. Build the front-end:

    ```bash
    pnpm run build
    ```

> [!NOTE]  
> for development, it's convenient to run the front-end in dev mode (watches for changes)
> use `pnpm run dev` instead

### Run the plugin (manually)
For power users / when not using pixi:


Run the back-end and front-end with a Grafana instance

```bash
pnpm run server
```

That's it, you should have a grafana instance running at port `3000`, head inside to find further instructions on how to use the plugin (click on **More Apps > Grafana-Yamcs Integration** on the side bar).

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
    # Spins up a Grafana instance first that we tests against
    pnpm run server

    # Starts the tests
    pnpm run e2e
    ```

6. Run the linter

    ```bash
    pnpm run lint

    # or

    pnpm run lint:fix
    ```
