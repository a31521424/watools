# macOS 本地开发与检查

[返回文档系统](../README.md) · [实施目标](../backlog/development-and-testing.md)

统一入口是 `node scripts/dev.mjs`，从任意目录调用脚本均可。当前宿主为 Wails `v3.0.0-beta.27`，优先 macOS 当前架构自用。Windows 与对外分发仍延期。

## 环境与首次启动

- Node.js `>=22.12.0`，pnpm `10.21.0`（取自前端 packageManager）。
- Go `1.26.8`（取自 go.mod 的 toolchain）；入口设置 GOTOOLCHAIN 与 GOWORK=off。
- Xcode Command Line Tools、macOS SDK。v3 的构建与 plist 最低系统版本统一为 macOS 12；当前实机覆盖以迁移验收记录为准。
- Wails CLI 不要求全局安装：入口运行 `go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27`，版本取自 go.mod。前端 `@wailsio/runtime` 同样精确固定 Beta.27。

```sh
node scripts/dev.mjs doctor
node scripts/dev.mjs setup
node scripts/dev.mjs plugins install
node scripts/dev.mjs dev
```

`setup` 下载 Go 模块，并用 frozen lockfile 安装前端；不安装系统依赖，不修改全局工具。`doctor` 检查工具、SDK、系统与路径。开发安装初始为空，可用 `plugins install watools.plugin.calculator` 只安装一个官方插件。

原生启动需要访问 macOS GUI 服务，开发 watcher 使用 FSEvents。执行沙箱可能允许编译却阻断 AppKit 初始化或文件监听；需在获准的本机执行环境启动，不应据此修改业务代码。系统版本查询改为 `sw_vers`，不再依赖 v2 CLI 的 sysctl 打包流程。

## 命令目录

| 命令 | 当前行为 |
| --- | --- |
| `doctor` | 检查固定工具、SDK、依赖和隔离路径 |
| `setup` | Go 模块下载与前端锁文件安装 |
| `dev [--port N]` | 原生宿主、Vite 热更新和 Go 重建；Vite 默认 9245 |
| `browser [--port N]` | Wails server 构建提供真实 Go 绑定、插件资源和事件；访问 `http://127.0.0.1:9246` |
| `bindings` | 为新检出准备 embed 占位资源，生成 v3 TypeScript 绑定，再构建前端 |
| `check` | 工作流回归、前端回归/构建、Go 测试/vet/production 编译 |
| `check-race` | 前端构建与 Go 全包 race 检查 |
| `build [--debug]` | 当前架构 `.app`，图标、plist 与 ad-hoc 签名，输出 `build/bin/`；不安装或启动 |
| `plugins list\|package\|install [package-id...]` | 列出、打包或安装官方插件；产物放入开发包目录 |

`dev/browser` 仅接受 `--port`（亦可用 `-port`），范围 1–65535 且不能占用宿主端口 9246；`build` 仅接受 `--debug`。v2 的 `-m`、`-browser`、`-appargs`、`-platform` 等 flags 不再透传。其他命令不接受额外 flags；插件命令仅接受 package ID，避免覆盖隔离目录。

`build/config.yml` 是 v3 watcher 模板，入口将端口和原生/浏览器目标渲染至 `.cache/dev/`。启动顺序是绑定及原生构建、Vite readiness、宿主；Go 修改重建并重启宿主，内存状态会丢失。`dev-build` 是 watcher 内部步骤，不是日常入口。

macOS 打包由 `scripts/mac-bundle.mjs` 按 v3 原生模板组装：Go 二进制、CLI 图标、plist 与 codesign。项目版本和作者仍由 `wails.json` 提供；该文件现在仅存项目元数据，v3 CLI 不读取其中的 v2 构建配置。模板保留原 bundle ID `com.wails.watools`。构建不自动 tidy；调整依赖后显式整理并审查 go.mod/go.sum。绑定生成文件位于 `frontend/bindings/`，修改源后重新生成，不能手改输出。

macOS 原生开发和 debug 构建带 `private_mac_apis`，生产构建带 `production,private_mac_apis`。固定 Beta.27 的公开 macOS 实现不会关闭 WKWebView 背景绘制；启动器同时配置 `MacBackdropTransparent`，通过该 tag 恢复透明边角。这沿用现有自用应用的 WebKit 私有属性能力，不能据此宣称符合 App Store 审核要求。浏览器构建只带 `server`，不启用该原生能力。

## 数据与取消

```text
.cache/
├── dev/
│   ├── data/        SQLite、插件、快捷键和本地凭据
│   ├── cache/       日志、图标和更新缓存
│   ├── packages/    官方插件 .wt 包
│   ├── bin/         原生/浏览器开发宿主
│   └── wails-*.yml  渲染后的 watcher 配置
└── tools/           Go 工具缓存
```

入口覆盖调用者的 `WATOOLS_DATA_DIR`、`WATOOLS_CACHE_DIR`。`dev/browser/plugins install` 共用开发安装；不要同时启动两个开发会话。`check/check-race/bindings/build` 使用一次性临时应用数据和缓存，结束或失败时清理。工具缓存可通过 GOCACHE、GOMODCACHE、GOPATH 复用，不改变应用隔离。显式数据目录不自动导入旧缓存；正式安装默认目录保留原迁移流程。

SIGINT/SIGTERM 转发到独立任务进程组，取消后即使子命令返回成功也不执行后续步骤。直属子进程退出后最多等待剩余任务组 5 秒；超时报告进程组并保留临时目录，不删除仍可能使用中的数据。v3 watcher 为自己的 Vite/宿主进程树设置退出策略与优雅关闭超时。生成 JS/TS/JSON 权限归一为 0644。

入口不清空真实安装、不提供自动重置；开发目录中的明文凭据同样不可提交。Wails framework 日志通过 slog 进入原旋转日志，过滤其自动记录的调用 `args/result` 字段，避免写入凭据或完整剪贴板内容。

## 调试边界

按用户 2026-10-04 的最新安排，继续沿用现有 dev flow，暂不实施低干扰能力适配或 VM runner，也不新增针对当前桌面使用影响的执行限制。`dev` 仍启动原生应用，`browser` 使用真实 Go 服务；数据隔离及下述能力边界保持原有约定。

`browser` 使用 v3 `server` build tag，不是固定数据模拟。页面和 `/wails/runtime` 必须来自 9246 宿主；仅打开 9245 Vite 页面无法调用 Go。服务器强制绑定 loopback，事件由 Wails WebSocket 桥接。浏览器可验证真实插件加载、存储、搜索、接口和热更新；框架在 server 模式的原生窗口、剪贴板和对话框能力不能作为桌面验收。server 的自动剪贴板读取返回空内容，宿主呼出/隐藏为空操作，图片剪贴板写入明确报不支持，避免进入没有 AppKit 主线程的 Cocoa 桥接。使用 `dev` 验证这些原生能力。

浏览器/桌面检查不混写：源码测试不等于 WebKit 交互，合成键盘事件不等于物理全局快捷键或真实输入法。迁移验收中可临时用 Wails 官方 `mcp` build tag 检查原生 WebView；默认产物没有 MCP 服务。完整 Go 断点流程、无宿主固定数据模拟、插件多窗口与远端 macOS CI 仍在对应 backlog。

实际版本、命令与验收边界见 [v3 迁移记录](../backlog/wails-v3-migration.md) 和 [开发工作流记录](../backlog/development-and-testing.md)。
