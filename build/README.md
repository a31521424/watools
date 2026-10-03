# Build files

`node scripts/dev.mjs build` builds the current macOS architecture using Wails v3. The Node helper in `scripts/mac-bundle.mjs` follows the official v3 macOS template and does not require a global CLI or Task installation.

- `appicon.png`: source icon; the pinned v3 CLI generates icons into the output directory.
- `darwin/Info.plist` and `darwin/Info.dev.plist`: metadata templates rendered from `wails.json`; minimum macOS 12 and the existing bundle ID are retained.
- `config.yml`: watcher template for native and real-host browser development, rendered by the unified entry.
- `bin/`: ignored output for local `.app` builds.

Windows manifest, icon and v2 installer sources are retained as migration references. Their v3 packaging and public release verification are deferred. The release workflow stops before the old v2 build/publish steps until distribution is explicitly resumed and migrated.
