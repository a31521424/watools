> 2026-10-07 当前自用安装入口为 `go run ./cmd/watoolsctl install`，一次构建并安装本体与官方插件，内置管理中心复用该流程。见[本地指南](./local-workflow.md)。本次按用户简化后的范围实施本机源码安装，没有启用 GitHub 商店或恢复远端 release workflow。

> 2026-10-03 v3 迁移说明：本机包的当前构建命令见本地指南；本页下方的公开分发流程保留为 v2 历史基线。release workflow 在 prepare 阶段明确停止，避免用 v2 CLI 构建 v3 源码或发布未经验证的 Windows/universal 包。恢复分发时须迁移整条构建、安装器和发布验收流程；本次未推送、发布或触发远端 CI。

# Release CD

[Docs index](../README.md) · [Agent entry](../../AGENT.md)

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
