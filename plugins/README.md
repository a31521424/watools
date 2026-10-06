# Official Plugins

This directory is the source of truth for WaTools official plugins.

## Layout

- `official/<package-id>/README.md`: plugin-specific notes
- `official/<package-id>/plugin/`: runtime files that are packaged into `.wt`
- `dist/`: generated `.wt` archives

## Official Plugins

- `watools.plugin.common`: open URLs and file paths, copy file paths, save clipboard images
- `watools.plugin.calculator`: executable calculator + calculator panel with history
- `watools.plugin.json`: JSON editor with bounded object/array candidate matching, seed/paste auto-formatting, expanded preview, and minified copy
- `watools.plugin.qr`: two-pane QR workspace for seeded text generation and clipboard image decoding
- `watools.plugin.translate`: translation panel with host-managed API key storage
- `watools.plugin.textstats`: explicit-trigger text statistics panel with multidimensional counts

## Local application installation

`go run ./cmd/watoolsctl install` builds and installs WaTools and all official plugins into the normal installation. After `git pull --ff-only`, rerun it to update. Use `--scope plugins --plugin watools.plugin.json` for one plugin. The built-in management center runs the same workflow. Storage, enabled state and third-party plugins are preserved; running windows retain their old revision.

`go run ./cmd/watoolsctl plugins install` remains isolated to development data. The Go commands below are lower-level packaging/install tools.

## Commands

List official plugins:

```bash
go run ./cmd/pluginctl list
```

Package all official plugins:

```bash
go run ./cmd/pluginctl package
```

Package a specific plugin:

```bash
go run ./cmd/pluginctl package watools.plugin.calculator
```

Install all official plugins into the local WaTools data directory:

```bash
go run ./cmd/pluginctl install
```

Install a specific official plugin:

```bash
go run ./cmd/pluginctl install watools.plugin.translate
```

## Development Notes

- For an isolated macOS development installation, use `go run ./cmd/watoolsctl plugins list|package|install [package-id...]`. Packages go to `.cache/dev/packages/`; installation shares `.cache/dev/data/` with `go run ./cmd/watoolsctl dev`. See the [local workflow](../docs/development/local-workflow.md). Direct `go run ./cmd/pluginctl install` uses the normal installation unless data/cache overrides are supplied.
- Runtime files come from each plugin's `plugin/` directory. UI packages also receive the canonical `frontend/public/watools-ui.css` at archive root; that name is reserved by the packager.
- The generated `.wt` archive contains the files at the root of `plugin/`, not the parent directory.
- `go run ./cmd/pluginctl install ...` uses the same backend installer logic as the app.
- `fronted-plugin/` is now legacy reference material. New official plugins belong here.
- UI plugins should read launch data from `window.pluginContext` and `watools:context-ready`.
- `seed` query params and `window.inputValue` are compatibility fallbacks only. The host does not put input text in URLs.
- Repackage and reinstall changed official plugins; updating the host alone does not replace installed plugin files.
- All five official UI plugins use the shared desktop design from version 0.3.0. Include `watools-ui.css` and use `<body class="wt-plugin">`; keep tool-specific styles to workspace layout. See the [UI guide](../docs/design/ui-style.md).
- All five official UI plugins provide `watoolsSession.capture/restore` from version 0.2.0. JSON preserves the edited model and panel; QR preserves its current image; translation preserves results without repeating requests; calculator preserves its current view; textstats preserves selection/scroll.
- The host injects `window.watools.session.windowMode` before `watools:context-ready`. Detached pages restore their snapshot instead of reapplying launch input or starting automatic work.
- After upgrading a plugin, save any open document, return to the launcher, refresh the plugin list (or restart), and reopen it. Existing pages keep their old immutable revision.
- Current official entries declare only `matchPreview` and require the current host. Third-party entries using legacy `match` remain supported through the host adapter.
- Translation requires the new Secret API. macOS saves credentials in owner-only local plaintext files; Windows uses DPAPI. Host storage failures are reported without copying keys to browser storage.
