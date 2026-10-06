# watools.plugin.json

Official JSON editor for explicit command-triggered editing.

## Features

- `json` or `json <payload>` trigger support
- `url2params <url-or-query>` trigger support
- `query2params <url-or-query>` and `qs2params <url-or-query>` aliases
- launch-context passthrough into the editor
- long JSON object/array candidates are recommended through bounded head/tail matching; selecting the UI entry imports the complete raw text without requiring a `json` prefix
- Monaco-based primary editor with VS Code-like multi-cursor and word navigation behavior
- path panel for current cursor path and batch extraction
- auto-format on startup seed when the payload is a JSON object or array
- auto-format on paste when the pasted text is a JSON object or array
- auto parse for `url2params` input with decode heuristics
- current JSON object can be serialized back to a hostname-free query string
- current editor content can be parsed as URL / query string via toolbar button
- one-click format, minify, and copy
- opens inside the launcher by default; `Cmd + D` detaches the edited document into an independent native window, allowing multiple JSON instances
- `watoolsSession` exports and restores editor text, cursor/view state, path input, and extraction output during detachment

## Match Rules

- `json`
- `json {"hello":"world"}`
- `json [1,2,3]`
- `url2params https://example.com?a=1&b=2`
- `url2params a%3D1%26b%3D%257B%2522x%2522%253A1%257D`
- `query2params foo=1&bar=2`
- `qs2params https%3A%2F%2Fexample.com%3Fa%3D1`
- when trailing content is not valid JSON, it is still passed through into the editor unchanged
- `matchPreview` detects candidate objects/arrays from bounded head/tail brackets and explicit command prefixes; it never parses JSON during launcher matching
- selecting “打开 JSON 编辑器” imports complete original clipboard text, then use the format button or `Cmd/Ctrl + Enter`; scalars remain supported by the editor but do not automatically match the bracket heuristic
- clipboard text is literal input, including JSON scalar values and command-like strings; command prefixes are parsed only for typed commands and legacy seeds

## Shortcuts

- `Cmd + D`: detach the current launcher document (or use the host's 分离 button)
- `Cmd/Ctrl + Enter`: format current JSON
- `Cmd/Ctrl + Shift + C`: copy extracted result, or copy current JSON when no extracted result exists
- `Cmd/Ctrl + L`: clear editor

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.json
```

## Window Detachment

Window detachment (0.2.0) preserves the edited model, view, path, extraction output, and panel visibility. Press `Cmd + D` in the main window to detach.

Repackage and reinstall the plugin after updating its source; save the open document, refresh the plugin list (or restart WaTools), and reopen it to load the new revision. Updating the host alone keeps the old installed plugin.
