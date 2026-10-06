# watools.plugin.qr

QR workspace for two trigger paths:

- `qr <text>` or `二维码 <text>`: receive seeded text from WaTools plugin context and render a QR image immediately
- clipboard image + `qr`: receive the clipboard image from WaTools plugin context and decode QR content into text

UI notes:

- two-pane text/image workspace
- text edits regenerate the QR image in place
- imported or pasted images decode back into text
- text supports copy, image supports copy and download
- shortcuts use both `Ctrl` and `Cmd`

## Window Detachment

Window detachment (0.2.0) preserves text, image pixels, imported/generated mode, and selection without regenerating or decoding the image. Press `Cmd + D` in the main window to detach.

Repackage and reinstall the plugin after updating its source; save the open document, refresh the plugin list (or restart WaTools), and reopen it to load the new revision. Updating the host alone keeps the old installed plugin.
