# Release CD

This repository now includes a GitHub Actions release workflow at `.github/workflows/release.yml`.

## Trigger

- Push a tag like `v0.1.0`
- Or run `workflow_dispatch` manually and provide `release_tag`

## What It Builds

- Windows: `watools_<version>_windows_amd64_installer.exe`
- macOS: `watools_<version>_macos_universal.zip`

The workflow uses the current Wails CLI build flow:

- `wails build -clean -platform windows/amd64 -nsis`
- `wails build -clean -platform darwin/universal`

## What It Publishes

The release job uploads these files to the GitHub Release page:

- Platform installers
- `watools_latest.json`
- `SHA256SUMS.txt`

`watools_latest.json` is the self-update manifest consumed by the app. The application reads it from:

`https://github.com/a31521424/watools/releases/latest/download/watools_latest.json`

## Version Rules

Before a release starts, the workflow validates:

- `wails.json.version`
- `wails.json.info.productVersion`

Both must match the release tag without the `v` prefix.

## Self Update Behavior

- Windows: the app downloads the NSIS installer and launches it.
- macOS: the app downloads the zip, tries to replace the current `.app`, and falls back to manual replacement if the install location is not writable.

## Current Limitation

The macOS build is unsigned and not notarized. Users may still need to acknowledge Gatekeeper warnings depending on how they open the app.
