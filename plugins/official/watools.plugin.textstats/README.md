# watools.plugin.textstats

Official text statistics panel for explicit command-triggered analysis.

## Features

- explicit trigger only, no generic text matching
- command-context prefill support from the WaTools command panel
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

## Window Detachment

Window detachment (0.2.0) preserves exact text, selection, and scroll position, then recomputes the statistics. Press `Cmd + D` in the main window to detach.

Repackage and reinstall the plugin after updating its source; save the open document, refresh the plugin list (or restart WaTools), and reopen it to load the new revision. Updating the host alone keeps the old installed plugin.
