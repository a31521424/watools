# watools.plugin.common

Official utility plugin for path, URL, and clipboard asset workflows.

## Features

- Open URLs directly from the command bar
- Reveal local paths with the host `OpenFolder` API
- Copy pasted file paths from clipboard content
- Save clipboard images to Downloads and reveal the saved file

## Match Rules

- URL or path input: `https://...`, `www...`, `~/...`, `/...`
- Clipboard files: shows `Copy File Path`
- Clipboard image: shows `Save Clipboard Image`

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.common
```
