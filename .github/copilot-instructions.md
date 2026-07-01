# Copilot Instructions for grafana-yamcs-jaops

## Project Overview
This is a Grafana plugin for connecting to Yamcs. It includes:
- **Frontend**: TypeScript/React panels in `src/`
- **Backend**: Go plugin backend in `pkg/`
Never capitalize Yamcs as YAMCS (it is not an acronym anymore)

## Terminal & Working Directory
- Keep track of the current working directory via your context
- Do NOT prefix commands with `cd` unnecessarily — check your context first. if two commands in a row cd to the same directory you have failed.
- If a command requires a different directory, change once and note the new context
- Don't pipe output to /dev/null unless sure it's huge. in general avoid suppressing output: context is important and tokens are cheap.
- Don't sleep uncessarily at the begining of a command. Count on me to allow the command at the right time.

## Package Manager
- Use `pnpm` for all frontend operations (not npm or yarn)

## Code Style

### TypeScript/React (Frontend)
- Source files are in `src/`
- Use functional React components with hooks
- Follow existing ESLint configuration
- Run `pnpm run lint:fix` before committing

### Go (Backend)
- Source files are in `pkg/`
- Follow standard Go conventions
- Run `go test ./pkg/...` for backend tests
- Use `gofmt` for formatting

## Testing
- Frontend: Jest (`pnpm run test`)
- Backend: Go test (`pnpm run test:backend` or `go test ./pkg/...`)
- E2E: Playwright (`pnpm run e2e`)

## Git Workflow
- Write clear, conventional commit messages (e.g., `feat:`, `fix:`, `ci:`, `docs:`). see CONTRIBUTING.md
- Do not be overly verbose
- Current branch workflow uses feature branches and tags for releases
- when releasing: make sure that package.json and go.mod versions match the tag before committing. make sure to run pnpm audit and osv-scanner before version bumping and pushing (only address high and critical, ignore others). 
- Don't push unecessarily, it trigers the CI each time. Push at end of process or when need the CI to trigger a release. 