# watools.plugin.translate

Official translation panel plugin.

## Features

- keyword-triggered translation workspace
- source and target language selection
- persisted language preferences
- copy result and quick source/target swap
- translation requests routed through WaTools `HttpProxy`

## Match Rules

- `fy ...`
- `translate ...`
- `翻译 ...`
- exact keywords `fy`, `translate`, `翻译`

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.translate
```
