# watools.plugin.json

Official JSON editor for explicit command-triggered editing.

## Features

- `json` or `json <payload>` trigger support
- `url2params <url-or-query>` trigger support
- `query2params <url-or-query>` and `qs2params <url-or-query>` aliases
- launch-context passthrough into the editor
- Monaco-based primary editor with VS Code-like multi-cursor and word navigation behavior
- path panel for current cursor path and batch extraction
- auto-format on startup seed when the payload is a JSON object or array
- auto-format on paste when the pasted text is a JSON object or array
- auto parse for `url2params` input with decode heuristics
- current JSON object can be serialized back to a hostname-free query string
- current editor content can be parsed as URL / query string via toolbar button
- one-click format, minify, and copy

## Match Rules

- `json`
- `json {"hello":"world"}`
- `json [1,2,3]`
- `url2params https://example.com?a=1&b=2`
- `url2params a%3D1%26b%3D%257B%2522x%2522%253A1%257D`
- `query2params foo=1&bar=2`
- `qs2params https%3A%2F%2Fexample.com%3Fa%3D1`
- when trailing content is not valid JSON, it is still passed through into the editor unchanged

## Shortcuts

- `Cmd/Ctrl + Enter`: format current JSON
- `Cmd/Ctrl + Shift + C`: copy extracted result, or copy current JSON when no extracted result exists
- `Cmd/Ctrl + L`: clear editor

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.json
```
