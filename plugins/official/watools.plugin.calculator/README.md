# watools.plugin.calculator

Official calculator plugin with two modes:

- executable mode for direct expressions from the command palette
- UI mode for repeated calculations and history
- minimalist panel with one input and one history list
- pressing `Enter` appends `expression = result` into history and clears the input
- calculator history mirrored to plugin storage and localStorage fallback
- draft expression restored from localStorage when reopening the panel

## Features

- direct evaluation for inputs like `2+3*4`
- panel mode for `calc`, `calculator`, `jsq`, `计算`, `计算器`
- auto normalization for common operator variants like `x`, `X`, `×`, `÷`
- keyboard-first flow with minimal on-screen controls
- shortcuts for copy latest result, copy latest line, clear input, and history reuse

## Match Rules

- expressions like `2+3*4`
- prefixed expressions like `calc 2+3`
- keywords `calc`, `calculator`, `jsq`, `计算`, `计算器`

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.calculator
```
