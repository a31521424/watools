# watools.plugin.translate

Official translation panel plugin, now built around DeepL Free.

## Features

- keyword-triggered translation workspace
- DeepL Free translation through WaTools `HttpProxy`
- header-based DeepL authentication compatible with the 2025 auth change
- plugin-scoped storage for the user's DeepL auth key
- auto source detection by DeepL
- target language inferred from the left text:
  - Chinese input -> English output
  - non-Chinese input -> Simplified Chinese output
- quick swap and copy workflow
- shortcut-first UI with minimal visible controls
- seeded input is accepted from the unified WaTools plugin context and auto-prefilled

## Match Rules

- `fy ...`
- `translate ...`
- `翻译 ...`
- exact keywords `fy`, `translate`, `翻译`
- generic text containing Chinese or Latin letters also surfaces the translation panel
- when opened through `fy/translate/翻译`, the prefix is stripped and only the trailing text is prefilled
- when opened with seeded content from the WaTools command panel, the source text is auto-filled directly into the left editor through the host plugin context

## Shortcuts

- `Cmd/Ctrl + Enter`: translate
- `Cmd/Ctrl + Shift + C`: copy translation
- `Cmd/Ctrl + Shift + X`: swap source/translation and translate back
- `Cmd/Ctrl + ,`: open the API key input

## Notes

- The plugin keeps the outer UI frameless; WaTools already provides the surrounding border and window chrome.
- The DeepL key uses the host Secret API: an owner-only plaintext file in the application data directory on macOS, or Windows DPAPI. macOS does not access Keychain. This requires a host version with Secret API support.
- Existing keys in plugin storage or `localStorage` are removed only after they have been successfully migrated to the host credential store. Repackage and reinstall this plugin to receive the updated behavior.

## Packaging

```bash
go run ./cmd/pluginctl package watools.plugin.translate
```

## Window Detachment

Window detachment (0.2.0) preserves source/result and view state, pauses pending translation, and restores without repeating API requests. Credentials reload from host storage and are excluded from the snapshot. Press `Cmd + D` in the main window to detach.

Repackage and reinstall the plugin after updating its source; save the open document, refresh the plugin list (or restart WaTools), and reopen it to load the new revision. Updating the host alone keeps the old installed plugin.
