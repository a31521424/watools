# CLAUDE.md

This file gives Claude Code a code-first map of this repository. It reflects the current implementation, not just the README.

## Project Summary

WaTools is a Wails desktop application with:

- a Go backend
- a React + TypeScript frontend
- a command-palette style UI
- app launching and system operation commands
- plugin loading through iframe-based UI plugins and executable plugins

The app is currently macOS-first in real functionality, even though the repository also contains Windows-specific files.

Current project metadata:

- app name: `watools`
- version: `0.1.0`
- Wails: `v2.11.0`
- Go: `1.24`
- React: `19`
- Vite: `7`
- Tailwind CSS: `4`

## What Actually Starts the App

`main.go` is the real entry point.

It:

- embeds `frontend/dist`
- parses `wails.json` into `config.ProjectInfo`
- initializes logging
- creates the singleton coordinator
- binds only `WaAppCoordinator` to Wails
- installs a custom asset handler for `/api/*`

Important implication:

- frontend calls do not talk to `internal/app`, `internal/plugin`, or `internal/command` directly
- all Wails-exposed methods must be added to `internal/coordinator/coordinator.go`

## High-Level Runtime Flow

1. A global hotkey shows the frameless Wails window.
2. The frontend loads application commands, operation commands, and plugin metadata.
3. User input is matched against:
   - application search results
   - operation commands
   - local app features
   - plugin entries
4. Triggering a result either:
   - launches an app
   - runs an operation
   - opens a plugin iframe
   - executes plugin code
5. Usage stats are buffered on the frontend and written back to SQLite in batches.

## Repository Map

### Backend

- `main.go`: Wails app setup and binding
- `config/config.go`: project metadata, cache dir, dev mode detection
- `internal/coordinator/`: the only Wails-bound API surface
- `internal/app/`: window lifecycle, hotkeys, clipboard integration
- `internal/command/`: app scanning, operation commands, filesystem watching
- `internal/plugin/`: plugin installation, loading, enable/disable, storage
- `internal/api/`: helper APIs exposed to frontend/plugins (`OpenFolder`, image save, HTTP proxy)
- `internal/handler/`: custom HTTP routes for icons and plugin assets
- `internal/app_menu/`: native menu with reload/refresh actions

### Frontend

- `frontend/src/app.tsx`: app root
- `frontend/src/components/watools/watools.tsx`: route shell
- `frontend/src/components/watools/wa-command.tsx`: main command palette
- `frontend/src/components/watools/wa-plugin.tsx`: iframe plugin host
- `frontend/src/components/watools/wa-plugin-management.tsx`: plugin management page
- `frontend/src/stores/`: Zustand stores for app input, applications, and plugins
- `frontend/src/api/`: thin wrappers over generated Wails bindings
- `frontend/src/schemas/`: shared frontend types
- `frontend/wailsjs/`: generated Wails bindings, usually do not hand-edit

### Persistence and Generated Code

- `pkg/db/`: sqlc-generated query layer and DB wrapper
- `pkg/db/migrations/`: embedded SQLite migrations
- `pkg/db/queries/`: SQL source files for sqlc
- `sqlc.yaml`: generation config

### Docs and Examples

- `README.md`: public project overview, partially outdated
- `DOCS/PLUGIN_DEVELOPMENT_INDEX.md`: plugin packaging/runtime entry index
- `fronted-plugin/`: sample/reference plugin assets, not the runtime plugin source of truth

## Real Architecture Details

### Coordinator Pattern

`internal/coordinator/coordinator.go` is the central bridge between frontend and backend.

It wires together:

- `app.GetWaApp()`
- `command.GetWaLaunch()`
- `plugin.GetWaPlugin()`
- `api.GetWaApi()`

When adding any frontend-callable method:

1. implement or reuse backend logic in the correct package
2. expose it from the coordinator
3. update `frontend/src/api/*` if needed
4. update frontend schemas/stores/components if payloads changed

### Window and Hotkey System

`internal/app/` owns:

- window show/hide behavior
- screen-aware resize/reposition
- global hotkey registration
- clipboard access

Current behavior worth knowing:

- default macOS hotkey is `cmd+Space`
- hotkey configs are persisted under `<cache>/hotkeys/config.json`
- in dev mode, the window is positioned for easier debugging instead of being centered normally
- on macOS, hiding the app tries to restore focus to the previously active app
- the main window auto-hides on blur in production mode

Hotkey APIs exist in `internal/app/app.go`, but only the coordinator is bound to Wails. If frontend needs hotkey management, coordinator methods must be added first.

### App Command System

`internal/command/command.go` manages application commands and operation commands.

Application command flow:

- app bundles are discovered from disk
- metadata is parsed into `models.ApplicationCommand`
- results are stored in SQLite
- a filesystem watcher tracks app directory changes
- `watools.applicationChanged` is emitted so the frontend can refresh

macOS application discovery currently scans:

- `/Applications`
- `/System/Applications`
- `/System/Applications/Utilities`
- `/System/Library/CoreServices`
- `~/Applications`

On macOS, app metadata comes from `Info.plist`, and display names/icons are resolved from bundle metadata when possible.

### Operation Commands

Built-in operation commands live in `internal/command/operator/`.

Current macOS operations include actions such as:

- system sleep
- lock screen
- empty trash
- show desktop
- toggle dark mode
- take screenshot
- mission control
- eject volumes

These are OS-script/command based and platform-specific.

### Plugin System

Plugins are managed by `internal/plugin/`.

Runtime model:

- installed plugins are extracted from `.wt` files
- plugin files are copied to `<cache>/plugins/<packageId>`
- plugin state is persisted in SQLite table `plugin_state`
- metadata is read from `manifest.json`
- plugin JS entry URLs are served through `/api/plugin/...`

There are two plugin entry types:

- `executable`: runs JS directly from the command palette
- `ui`: opens an iframe page in `/plugin`

Important implementation details:

- plugin metadata and enabled/storage/usage state are separate concerns
- plugin storage is persisted as JSON in SQLite
- plugin assets are served by the custom HTTP handler, not by Vite directly
- `fronted-plugin/` is only a local examples/reference directory; the app does not auto-load plugins from there

### Plugin Frontend API Exposure

The main window sets:

- `window.watools = WaApi`

The iframe plugin host sets:

- `iframeWindow.runtime = window.runtime`
- `iframeWindow.watools = createWaToolsApi(packageId)`
- `iframeWindow.pluginContext = PluginContext`
- `watools:context-ready` with `PluginContext` in `event.detail`

That means:

- plugins should use `window.watools`
- UI plugins should read launch data from `window.pluginContext`
- UI plugins should handle `watools:context-ready` for the authoritative context handoff
- storage calls are package-scoped only when the plugin is hosted through `createWaToolsApi(packageId)`
- direct assumptions about `window.go` are the wrong abstraction here

Supported plugin-facing helpers currently include:

- `OpenFolder`
- `SaveBase64Image`
- `HttpProxy`
- `StorageGet`
- `StorageSet`
- `StorageRemove`
- `StorageClear`
- `StorageKeys`

If plugin APIs change, update both:

- backend coordinator methods
- `frontend/src/api/api.ts`

### Custom HTTP Routes

`internal/handler/handler.go` intercepts `/api/*`.

Current routes:

- `/api/application-icon`: app icon serving
- `/api/plugin`: installed plugin asset serving

Anything outside `/api/*` falls back to the embedded frontend assets.

### Database Layer

SQLite lives under:

- `<user cache dir>/watools/data/watools.db`

Migrations are embedded with `go:embed` and applied automatically on startup.

Current schema includes:

- `application`
- `plugin_state`
- `metadata`

Usage stats for applications and plugins are persisted and updated in batches.

If you change SQL:

1. update `pkg/db/queries/*.sql` and/or migrations
2. run `sqlc generate`
3. check conversion code in `pkg/db/conversion.go`
4. check frontend schemas if payloads changed

## Frontend Behavior That Matters

### Routing

The frontend uses `wouter` with three routes:

- `/`: main command palette
- `/plugin`: iframe plugin host
- `/plugin-management`: plugin install/enable/uninstall UI

### State Management

Key Zustand stores:

- `appStore`: current input value, clipboard-derived content, image/file payloads
- `applicationCommandStore`: application command cache, Fuse instance, usage buffer, Wails event refresh
- `pluginStore`: plugin metadata, enabled filtering, usage buffer, install/uninstall/toggle actions

### Search and Ranking

Search is split by source, then merged:

- applications: Fuse search over app names, pinyin, initials, and path name
- operations: separate Fuse search
- local app features: separate Fuse search
- plugins: direct `entry.match(context)` evaluation

Final items are combined and then sorted by `usedCount`.

Practical consequence:

- if ranking changes are needed, check both per-source matching and the final merged sort
- plugin results are not Fuse-based; they depend entirely on plugin `match()`

### Clipboard/Input Model

The app supports more than plain text input.

`appStore` can hold:

- text
- clipboard text
- clipboard image as base64
- clipboard file paths

On window focus, the app reads clipboard content and may auto-fill the command bar.

When working on plugin matching or input UX, inspect:

- `frontend/src/components/watools/wa-command.tsx`
- `frontend/src/stores/appStore.ts`
- `frontend/src/schemas/app.ts`

## Commands To Use During Development

Preferred commands in this repo:

```bash
wails dev
wails build -clean
go test ./...
sqlc generate
cd frontend && npm install
cd frontend && npm run dev
cd frontend && npm run build
```

Notes:

- `wails.json` is configured with npm commands for frontend install/build/dev
- `frontend/package.json` also declares `yarn@1.22.22`, but Wails itself is wired to npm right now
- `frontend/dist` is generated output
- `frontend/wailsjs` is generated by Wails

## Files And Directories To Treat Carefully

- `frontend/wailsjs/`: generated bindings
- `frontend/dist/`: generated build output
- `.cache/`: local build/cache artifacts
- `<user cache dir>/watools/`: runtime data location outside the repo

## Practical Rules For Future Changes

### If You Add Backend Data To The Frontend

Update all of:

1. Go model / returned map shape
2. coordinator API
3. frontend `src/api/*`
4. frontend schema/types
5. any Zustand store or component that consumes it

### If You Add A New Plugin Capability

Check all of:

1. backend method in `internal/plugin` or `internal/api`
2. coordinator exposure
3. `frontend/src/api/api.ts`
4. iframe host setup in `frontend/src/components/watools/wa-plugin.tsx`
5. `DOCS/PLUGIN_DEVELOPMENT_INDEX.md` and the relevant plugin docs module if developer-facing behavior changed

### If You Change App Discovery Or Search

Check all of:

1. `internal/command/application/*`
2. `internal/command/watcher/*`
3. `pkg/db/*`
4. `frontend/src/api/command.ts`
5. `frontend/src/stores/applicationCommandStore.ts`

### If You Change Plugin Installation

Check all of:

1. `.wt` unzip and manifest validation
2. file copy destination under cache dir
3. DB registration and removal
4. `/api/plugin` asset serving
5. plugin metadata assumptions in frontend loading

## Known Realities / Caveats

- The public README is lighter and less exact than the codebase.
- The project is cross-platform in structure, but many polished behaviors are macOS-centered.
- `fronted-plugin/` is misspelled in the directory name and currently acts as example/reference material.
- Comments are not always perfectly aligned with literal values; trust the code path over comments.
- Plugin UI execution depends on iframe injection of `runtime` and `watools`, so plugin bugs often come from missing assumptions there.

## Recommended Reading Order For Any Non-Trivial Change

1. `main.go`
2. `internal/coordinator/coordinator.go`
3. the relevant backend package under `internal/`
4. the relevant DB/model files under `pkg/`
5. the matching frontend store/component/api wrapper

If the task is plugin-related, also read `DOCS/PLUGIN_DEVELOPMENT_INDEX.md`.
