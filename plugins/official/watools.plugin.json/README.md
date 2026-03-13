# watools.plugin.json

Official JSON editor for explicit command-triggered editing.

## Features

- `json` or `json <payload>` trigger support
- launch-context passthrough into the editor
- Monaco-based primary editor with VS Code-like multi-cursor and word navigation behavior
- floating node preview panel that does not squeeze the editor
- auto-format on startup seed when the payload is a JSON object or array
- auto-format on paste when the pasted text is a JSON object or array
- click any node to extract its path and content
- one-click format, minify, and copy

## Match Rules

- `json`
- `json {"hello":"world"}`
- `json [1,2,3]`
- when trailing content is not valid JSON, it is still passed through into the editor unchanged

## Shortcuts

- `Cmd/Ctrl + Enter`: format current JSON
- `Cmd/Ctrl + Shift + C`: copy selected node content, or copy the current JSON when no node is selected
- `Cmd/Ctrl + L`: clear editor

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.json
```
