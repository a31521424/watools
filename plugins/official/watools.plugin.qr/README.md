# watools.plugin.qr

QR workspace for two trigger paths:

- `qr <text>` or `二维码 <text>`: seed text and render a QR image immediately
- clipboard image + `qr`: seed the clipboard image and decode QR content into text

UI notes:

- two-pane text/image workspace
- text edits regenerate the QR image in place
- imported or pasted images decode back into text
- text supports copy, image supports copy and download
- shortcuts use both `Ctrl` and `Cmd`
