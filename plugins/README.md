# Official Plugins

This directory is the source of truth for WaTools official plugins.

## Layout

- `official/<package-id>/README.md`: plugin-specific notes
- `official/<package-id>/plugin/`: runtime files that are packaged into `.wt`
- `dist/`: generated `.wt` archives

## Official Plugins

- `watools.plugin.common`: open URLs and file paths, copy file paths, save clipboard images
- `watools.plugin.calculator`: executable calculator + calculator panel with history
- `watools.plugin.json`: explicit-trigger JSON editor with seed/paste auto-formatting, expanded preview, and minified copy
- `watools.plugin.qr`: two-pane QR workspace for seeded text generation and clipboard image decoding
- `watools.plugin.translate`: translation panel with persisted language preferences
- `watools.plugin.textstats`: explicit-trigger text statistics panel with multidimensional counts

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

Install all official plugins into the local WaTools cache:

```bash
go run ./cmd/pluginctl install
```

Install a specific official plugin:

```bash
go run ./cmd/pluginctl install watools.plugin.translate
```

## Development Notes

- Package contents come only from each plugin's `plugin/` directory.
- The generated `.wt` archive contains the files at the root of `plugin/`, not the parent directory.
- `go run ./cmd/pluginctl install ...` uses the same backend installer logic as the app.
- `fronted-plugin/` is now legacy reference material. New official plugins belong here.
- UI plugins should read launch data from `window.pluginContext` and `watools:context-ready`.
- `seed` query params and `window.inputValue` are compatibility fallbacks only.
