# watools.plugin.textstats

Official text statistics panel for explicit command-triggered analysis.

## Features

- explicit trigger only, no generic text matching
- seed input support from the WaTools command panel
- live multidimensional text statistics
- Unicode-aware counts for letters, Han characters, punctuation, symbols, emoji, and more
- quick copy for a plain-text summary

## Match Rules

- `count ...`
- `chars ...`
- `textcount ...`
- `字符统计 ...`
- `字数统计 ...`
- `文本统计 ...`
- exact keywords without trailing content also open the panel
- when opened through a trigger phrase, only the trailing content is prefilled into the editor

## Shortcuts

- `Cmd/Ctrl + Shift + C`: copy summary
- `Cmd/Ctrl + L`: clear editor

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.textstats
```
