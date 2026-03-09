# watools.plugin.calculator

Official calculator plugin with two modes:

- executable mode for direct expressions from the command palette
- UI mode for repeated calculations and history

## Match Rules

- expressions like `2+3*4`
- prefixed expressions like `calc 2+3`
- keywords `calc`, `calculator`, `jsq`, `计算`, `计算器`

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.calculator
```
