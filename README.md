# WaTools

[![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/a31521424/watools/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/a31521424/watools)](https://goreportcard.com/report/github.com/a31521424/watools)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)](https://wails.io)

[中文说明](./README.zh-CN.md)

WaTools is an open-source desktop productivity toolbox inspired by uTools and Alfred. It combines a Wails desktop shell, a Go backend, and a React command-palette UI to make local tools, system actions, and plugins available from a single hotkey-driven surface.

## Overview

WaTools is currently organized around four core capabilities:

- A transparent, frameless command-palette style desktop UI
- Local application search and launch
- Built-in system operation commands
- A trusted local plugin system with `executable` and `ui` entries

The repository is cross-platform in structure, but the current implementation is still most complete on macOS.

## Project Story

WaTools started as a manually built desktop tool project. The early implementation, architecture, and core workflows were designed and written in a traditional hand-coded way.

As the project evolved, the workflow expanded into AI-assisted development. Recent iterations increasingly use Codex, Claude Code, and related vibe coding practices for repo mapping, documentation restructuring, implementation acceleration, and plugin-oriented iteration.

That means this project is best understood as:

- Initially human-designed and human-implemented
- Later accelerated with AI pair-programming and vibe coding style workflows
- Still grounded in concrete repo structure, explicit modules, and reviewable code paths

## Current Architecture

At a high level, the app is split into:

- `main.go`
  Wails entrypoint, asset embedding, lifecycle wiring, native window setup
- `internal/`
  Backend modules for app behavior, command scanning, plugins, APIs, handlers, updates, and menus
- `frontend/`
  React 19 + TypeScript + Tailwind 4 frontend for the command palette, plugin host, and management pages
- `pkg/`
  Shared models, SQLite access, and logging
- `plugins/`
  Official plugin source tree and packaging output
- `docs/`
  Architecture, UI, implementation mechanism, feature module, release, and plugin development docs

The only Wails-bound API surface is the coordinator in `internal/coordinator/`, which acts as the stable bridge between frontend and backend modules.

## Key Features

- Global hotkey access to the main panel
- Application discovery and launch
- System operation commands
- Plugin installation from trusted `.wt` packages
- UI plugins hosted through iframe pages with a shared `PluginContext`
- Executable plugins that run directly from the command palette
- Local logging and update management screens
- Official plugin set for common utilities, calculator, JSON, QR, translation, and text statistics workflows

## Technology Stack

- Backend: Go `1.26`
- Desktop framework: Wails `v2`
- Frontend: React `19`, TypeScript, Vite `7`
- Styling: Tailwind CSS `4`
- State management: Zustand
- Search and command UI: Fuse.js, cmdk
- Package manager: `pnpm`
- Persistence: SQLite

## Docs Map

Start with the entry that matches the task instead of reading everything end-to-end.

- [`AGENT.md`](./AGENT.md)
  Repository map, runtime summary, and fast orientation for developers and agents
- [`docs/README.md`](./docs/README.md)
  Documentation index
- [`docs/architecture.md`](./docs/architecture.md)
  System layers, runtime boundaries, and data flow
- [`docs/ui-style.md`](./docs/ui-style.md)
  Command palette structure, routes, and UI direction
- [`docs/implementation-mechanism.md`](./docs/implementation-mechanism.md)
  Wails binding, command, plugin, update, and resource-serving mechanisms
- [`docs/feature-modules.md`](./docs/feature-modules.md)
  User-facing functional modules
- [`docs/PLUGIN_DEVELOPMENT_INDEX.md`](./docs/PLUGIN_DEVELOPMENT_INDEX.md)
  Plugin development reading path

## Repository Layout

```text
.
|-- main.go
|-- config/
|-- internal/
|-- frontend/
|-- pkg/
|-- plugins/
|-- docs/
|-- cmd/pluginctl/
`-- build/
```

## Getting Started

### Prerequisites

- Go `1.26.1+`
- Node.js `18+`
- `pnpm`
- Wails CLI

### Install and Run

1. Clone the repository.

```sh
git clone https://github.com/a31521424/watools.git
cd watools
```

2. Install frontend dependencies.

```sh
cd frontend
pnpm install
cd ..
```

3. Start the development app.

```sh
wails dev
```

4. Build a production binary.

```sh
wails build
```

The output binary is written to `build/bin/`.

## Official Plugins

Official plugin source code lives in [`plugins/official`](./plugins/official).

Current official plugins include:

- `watools.plugin.common`
- `watools.plugin.calculator`
- `watools.plugin.json`
- `watools.plugin.qr`
- `watools.plugin.translate`
- `watools.plugin.textstats`

Useful plugin commands:

```sh
go run ./cmd/pluginctl list
go run ./cmd/pluginctl package
go run ./cmd/pluginctl install
```

More plugin details:

- [`plugins/README.md`](./plugins/README.md)
- [`docs/PLUGIN_DEVELOPMENT_INDEX.md`](./docs/PLUGIN_DEVELOPMENT_INDEX.md)

## Release and Updates

The repository includes a GitHub Actions release workflow and app update support.

- Workflow: [`/.github/workflows/release.yml`](./.github/workflows/release.yml)
- Notes: [`docs/release-cd.md`](./docs/release-cd.md)

## Trust Model

Installed plugins are currently treated as trusted local code chosen by the user. WaTools is not positioned as a hardened sandbox for an untrusted public marketplace.

## Contributing

Issues and pull requests are welcome. If you are changing architecture, plugin behavior, or developer-facing workflows, update the matching docs under [`docs/`](./docs/) together with the code.

## License

This project is licensed under the MIT License.
