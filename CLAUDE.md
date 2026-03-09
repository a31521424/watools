# CLAUDE.md

This file gives Claude Code a code-first map of this repository. It reflects the current implementation.

## Project Summary

WaTools is a Wails desktop app with:

- Go backend
- React + TypeScript frontend
- command-palette UI
- application launcher and system operations
- plugin support for executable plugins and iframe-based UI plugins

Current stack:

- app name: `watools`
- version: `0.1.0`
- Go: `1.24`
- Wails: `v2.11.0`
- React: `19`
- Vite: `7`
- Tailwind CSS: `4`

The codebase is cross-platform in structure, but the real feature depth is currently macOS-first.

## Entry Point

`main.go` is the app entry point. It:

- embeds `frontend/dist`
- parses `wails.json`
- initializes logging
- creates the singleton coordinator
- binds only `WaAppCoordinator` to Wails
- installs a custom asset handler for `/api/*`

Practical consequence:

- frontend must call backend through `internal/coordinator/coordinator.go`
- adding a frontend-callable backend feature means updating the coordinator

## Repository Map

### Backend

- `main.go`: Wails setup and binding
- `config/config.go`: project metadata, cache dir, dev-mode detection
- `internal/coordinator/`: Wails API surface
- `internal/app/`: window lifecycle, hotkeys, clipboard
- `internal/command/`: app discovery, operation commands, filesystem watching
- `internal/plugin/`: plugin install/load/enable/storage
- `internal/api/`: helper APIs exposed to frontend/plugins
- `internal/handler/`: `/api/application-icon` and `/api/plugin` routes
- `internal/app_menu/`: native reload/refresh menu

### Frontend

- `frontend/src/components/watools/watools.tsx`: route shell
- `frontend/src/components/watools/wa-command.tsx`: main command palette
- `frontend/src/components/watools/wa-plugin.tsx`: iframe plugin host
- `frontend/src/components/watools/wa-plugin-management.tsx`: install/enable/uninstall UI
- `frontend/src/stores/`: Zustand stores
- `frontend/src/api/`: Wails wrappers
- `frontend/src/schemas/`: frontend-side shared types
- `frontend/wailsjs/`: generated bindings

### Persistence

- `pkg/db/`: SQLite wrapper + sqlc output
- `pkg/db/migrations/`: embedded migrations
- `pkg/db/queries/`: sqlc source SQL
- `sqlc.yaml`: sqlc config

### Docs and Examples

- `README.md`: public overview
- `PLUGIN_DEVELOPMENT.md`: plugin packaging/runtime rules
- `fronted-plugin/`: local example/reference plugins, not auto-loaded runtime plugins

## Runtime Flow

1. Global hotkey shows the frameless window.
2. Frontend loads applications, operations, and plugin metadata.
3. User input is matched against apps, operations, local app features, and plugin entries.
4. Triggering a result launches an app, runs an operation, opens a plugin iframe, or executes plugin code.
5. Usage stats are buffered on the frontend and persisted to SQLite.

## Key Architecture Notes

### Coordinator

`internal/coordinator/coordinator.go` wires together:

- `app.GetWaApp()`
- `command.GetWaLaunch()`
- `plugin.GetWaPlugin()`
- `api.GetWaApi()`

When payloads change, update:

1. Go backend logic
2. coordinator API
3. frontend `src/api/*`
4. frontend schemas/stores/components

### Window and Hotkeys

`internal/app/` owns:

- show/hide behavior
- screen-aware sizing/repositioning
- global hotkeys
- clipboard access

Current details:

- default macOS hotkey is `cmd+Space`
- hotkey config is persisted under `<cache>/hotkeys/config.json`
- production auto-hides on blur
- macOS hide tries to restore focus to the previous app

### Command System

`internal/command/command.go` manages application and operation commands.

macOS application scanning currently reads:

- `/Applications`
- `/System/Applications`
- `/System/Applications/Utilities`
- `/System/Library/CoreServices`
- `~/Applications`

Applications are parsed into `models.ApplicationCommand`, stored in SQLite, and refreshed through filesystem watcher events.

### Plugin System

Plugins are installed from `.wt` archives into `<cache>/plugins/<packageId>` and tracked in SQLite `plugin_state`.

There are two entry types:

- `executable`: runs from the command palette
- `ui`: opens inside `/plugin` as an iframe page

Important implementation details:

- plugin metadata comes from `manifest.json`
- plugin entry JS is loaded dynamically from `/api/plugin/...`
- plugin storage is package-scoped JSON persisted in SQLite
- disabled plugins are not loaded
- plugin assets and package IDs are path-validated before serving/loading

Trust model:

- plugins are treated as trusted code chosen by the user
- the current implementation is not a hardened sandbox boundary
- do not describe plugins as fully isolated or safe for untrusted marketplace code

### Plugin API Exposure

Main window:

- `window.watools = WaApi`

Iframe plugin host:

- `iframeWindow.runtime = window.runtime`
- `iframeWindow.watools = createWaToolsApi(packageId)`

Currently exposed plugin helpers:

- `OpenFolder`
- `SaveBase64Image`
- `HttpProxy`
- `StorageGet`
- `StorageSet`
- `StorageRemove`
- `StorageClear`
- `StorageKeys`

### Custom Asset Routes

`internal/handler/handler.go` handles:

- `/api/application-icon`
- `/api/plugin`

Everything else is served by the embedded frontend build.

### Database

SQLite path:

- `<user cache dir>/watools/data/watools.db`

Schema currently includes:

- `application`
- `plugin_state`
- `metadata`

Migrations are embedded and run automatically on startup.

## Frontend Notes

Routes:

- `/`: command palette
- `/plugin`: iframe plugin host
- `/plugin-management`: plugin management UI

Main stores:

- `appStore`: current input + clipboard content
- `applicationCommandStore`: app cache, Fuse search, usage buffer
- `pluginStore`: plugin metadata, enabled filtering, usage buffer

Search behavior:

- applications use Fuse with pinyin support
- operations use their own Fuse search
- app features use their own Fuse search
- plugins depend on `entry.match(context)`
- merged results are then sorted by `usedCount`

## Development Commands

Use:

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

- `wails.json` is wired to npm for frontend install/build/dev
- `frontend/package.json` still declares `yarn@1.22.22`
- `frontend/dist` and `frontend/wailsjs` are generated output

## Change Checklist

If you change backend data exposed to frontend, update:

1. Go logic/model
2. coordinator method
3. frontend API wrapper
4. frontend schema
5. consuming store/component

If you change plugin behavior, check:

1. `internal/plugin/*`
2. `internal/handler/plugin.go`
3. `frontend/src/api/plugin.ts`
4. `frontend/src/api/api.ts`
5. `frontend/src/components/watools/wa-plugin.tsx`
6. `PLUGIN_DEVELOPMENT.md`

If you change app discovery or ranking, check:

1. `internal/command/application/*`
2. `internal/command/watcher/*`
3. `pkg/db/*`
4. `frontend/src/api/command.ts`
5. `frontend/src/stores/applicationCommandStore.ts`
