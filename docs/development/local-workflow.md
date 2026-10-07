# macOS 本地开发与检查

[返回文档系统](../README.md) · [实施目标](../backlog/development-and-testing.md)

统一入口是 `go run ./cmd/watoolsctl`，在仓库根目录执行；编译后的 CLI 也可从源码子目录或自身缓存目录定位仓库。当前宿主为 Wails `v3.0.0-beta.27`，优先 macOS 当前架构自用。Windows 与对外分发仍延期。

## 环境与首次启动

- Node.js `>=22.12.0`，pnpm `10.21.0`（取自前端 packageManager）。
- Go `1.26.8`（取自 go.mod 的 toolchain）；入口设置 GOTOOLCHAIN 与 GOWORK=off。
- Xcode Command Line Tools、macOS SDK。v3 的构建与 plist 最低系统版本统一为 macOS 12；当前实机覆盖以迁移验收记录为准。
- Wails CLI 不要求全局安装：入口运行 `go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27`，版本取自 go.mod。前端 `@wailsio/runtime` 同样精确固定 Beta.27。

```sh
go run ./cmd/watoolsctl doctor
go run ./cmd/watoolsctl setup
go run ./cmd/watoolsctl plugins install
go run ./cmd/watoolsctl dev
```

`setup` 下载 Go 模块，并用 frozen lockfile 安装前端；不安装系统依赖，不修改全局工具。`doctor` 检查工具、SDK、系统与路径。开发安装初始为空，可用 `plugins install watools.plugin.calculator` 只安装一个官方插件。

原生启动需要访问 macOS GUI 服务，开发 watcher 使用 FSEvents。执行沙箱可能允许编译却阻断 AppKit 初始化或文件监听；需在获准的本机执行环境启动，不应据此修改业务代码。Go 入口不依赖 v2 CLI 的 sysctl 打包流程。

## 安装到本机与覆盖更新

已有编译环境时，clone 后直接执行，不需要先运行 setup：

```sh
go run ./cmd/watoolsctl install
```

该入口通过固定 Wails v3 CLI 和 Go 构建本体，同时打包全部官方 `.wt` 插件。本体安装到 `/Applications/watools.app`；插件安装到正式数据目录，保留配置、插件存储、启用状态和第三方插件。首次安装自动打开应用；覆盖更新保留正在运行的应用和小窗，保存内容后退出重开。

```sh
git pull --ff-only
go run ./cmd/watoolsctl install
# 可选：只更新插件、指定插件，或安装到个人目录
go run ./cmd/watoolsctl install --scope plugins
go run ./cmd/watoolsctl install --scope plugins --plugin watools.plugin.json
go run ./cmd/watoolsctl install --scope application --app-dir "$HOME/Applications" --no-launch
```

`--scope` 默认 `all`，`--plugin` 可重复指定；不指定 ID 时覆盖全部官方插件。同版本可重装，插件降级被拒绝。`--app-dir` 指定应用父目录，`--no-launch` 禁止首次安装自动打开。统一工作流由 Go 执行，本体由 Go/Wails 编译，Node／pnpm 仅用于前端任务。只更新插件时不调用 Node／pnpm。

搜索“管理”“插件”“更新”进入内置管理中心。安装命令把源码路径、Go 路径、终端 PATH 和安装位置记录到正式数据目录的 `installation.json`，使 Finder 启动的应用也能找到编译工具。管理中心先用记录的 Go 编译当前源码中的轻量 CLI，再直接执行同一个入口，支持本体、全部官方插件或单个官方插件更新；已有 `.wt` 安装、启用、禁用和卸载功能仍可用。源码迁移后可修改设置；只执行自己信任的源码。管理中心不会自动拉取代码，先在终端 `git pull --ff-only`。

安装前使用临时数据完成依赖准备、打包、构建和签名。签名校验后在目标目录暂存应用、备份旧 bundle，再原子切换应用和插件版本指针；安装或收据写入失败会恢复应用和已切换的插件指针。旧应用备份为 `.watools-backup-*.app`，旧插件 revision 保留以供运行中小窗继续使用。安装锁同时覆盖源码构建、正式数据和应用目标目录；并发安装明确报错。

构建阶段可取消；进入安装提交阶段后延迟取消，等待成功或恢复。该恢复覆盖程序正常报告的错误，不承诺断电或强制 SIGKILL 下的跨应用／SQLite 事务原子性。源码锁现在使用内核文件锁；进程退出自动释放，保留锁文件不代表仍有任务运行。备份和旧 revision 当前不自动清理。

## 命令目录

| 命令 | 当前行为 |
| --- | --- |
| `doctor` | 检查固定工具、SDK、依赖和隔离路径 |
| `setup` | Go 模块下载与前端锁文件安装 |
| `dev [--port N]` | 原生宿主、Vite 热更新和 Go 重建；Vite 默认 9245 |
| `browser [--port N]` | Wails server 构建提供真实 Go 绑定、插件资源和事件；访问 `http://127.0.0.1:9246` |
| `bindings` | 为新检出准备 embed 占位资源，生成 v3 TypeScript 绑定，再构建前端 |
| `check` | 前端回归/构建、Go 工作流及全包测试/vet/production 编译 |
| `check-race` | 前端构建与 Go 全包 race 检查 |
| `install [--scope all\|application\|plugins] [--plugin ID] [--app-dir DIR] [--no-launch]` | 本体和官方插件安装／覆盖更新，使用正式数据 |
| `build [--debug]` | 当前架构 `.app`，图标、plist 与 ad-hoc 签名，输出 `build/bin/`；不安装或启动 |
| `plugins list\|package\|install [package-id...]` | 列出、打包或安装官方插件；产物放入开发包目录 |

`dev/browser` 仅接受 `--port`（亦可用 `-port`），范围 1–65535 且不能占用宿主端口 9246；`build` 仅接受 `--debug`。v2 的 `-m`、`-browser`、`-appargs`、`-platform` 等 flags 不再透传。除 `install` 外，其他命令不接受额外 flags；插件命令仅接受 package ID，避免覆盖隔离目录。

`build/config.yml` 是 v3 watcher 模板，入口将端口和原生/浏览器目标渲染至 `.cache/dev/`。启动顺序是绑定及原生构建、Vite readiness、宿主；Go 修改重建并重启宿主，内存状态会丢失。`dev-build` 是 watcher 内部步骤，不是日常入口。

macOS 打包由 `internal/workflow/bundle.go` 按 v3 原生模板组装：Go 二进制、CLI 图标、plist 与 codesign。项目版本和作者仍由 `wails.json` 提供；该文件现在仅存项目元数据，v3 CLI 不读取其中的 v2 构建配置。模板保留原 bundle ID `com.wails.watools`。构建不自动 tidy；调整依赖后显式整理并审查 go.mod/go.sum。绑定生成文件位于 `frontend/bindings/`，修改源后重新生成，不能手改输出。

macOS 原生开发和 debug 构建带 `private_mac_apis`，生产构建带 `production,private_mac_apis`。固定 Beta.27 的公开 macOS 实现不会关闭 WKWebView 背景绘制；启动器同时配置 `MacBackdropTransparent`，通过该 tag 恢复透明边角。这沿用现有自用应用的 WebKit 私有属性能力，不能据此宣称符合 App Store 审核要求。浏览器构建只带 `server`，不启用该原生能力。

## 数据与取消

```text
.cache/
├── dev/
│   ├── data/        SQLite、插件、快捷键和本地凭据
│   ├── cache/       日志、图标和更新缓存
│   ├── packages/    官方插件 .wt 包
│   ├── bin/         原生/浏览器开发宿主
│   ├── vite/        当前检出的前端优化缓存
│   └── wails-*.yml  渲染后的 watcher 配置
└── tools/           Go 工具缓存
```

开发／检查入口覆盖调用者的 `WATOOLS_DATA_DIR`、`WATOOLS_CACHE_DIR`。`install` 构建阶段同样隔离，应用安装阶段恢复调用者显式覆盖（用于独立验收），未覆盖时使用正式目录。`dev/browser/plugins install` 共用开发安装；不要同时启动两个开发会话。`check/check-race/bindings/build` 使用一次性临时应用数据和缓存，结束或失败时清理。工具缓存可通过 GOCACHE、GOMODCACHE、GOPATH 复用，不改变应用隔离。显式数据目录不自动导入旧缓存；正式安装默认目录保留原迁移流程。

CLI 持续监听 SIGINT/SIGTERM，并检测 `go run` 外层退出；取消转发到独立任务进程组，取消后即使子命令返回成功也不执行后续步骤。非提交任务先发送 TERM，给任务组最多 10 秒退出；超时后 KILL 整组并等待收尾，确认退出后才释放源码锁和临时数据。继承的输出管道也有 5 秒收尾上限。若系统仍无法终止进程则报告并保留临时目录，不删除可能使用中的数据。安装提交不走强制终止，等待完成或恢复。v3 watcher 为自己的 Vite/宿主进程树设置退出策略与优雅关闭超时。生成 JS/TS/JSON 权限归一为 0644。

`build` 不更新已安装插件。遇到“当前插件版本不支持编辑器状态迁移”，开发模式运行 `go run ./cmd/watoolsctl plugins install watools.plugin.json`；正式安装运行 `go run ./cmd/watoolsctl install --scope plugins --plugin watools.plugin.json`，或在管理中心更新。五个官方 UI 插件的完整迁移接口从 0.2.0 起提供。安装后先保存旧页面编辑内容，返回启动器并刷新插件列表，或退出旧应用后启动新构建，再重新打开插件；已有页面不会自动切换 revision。

入口不清空真实安装、不提供自动重置；开发目录中的明文凭据同样不可提交。Wails framework 日志通过 slog 进入原旋转日志，过滤其自动记录的调用 `args/result` 字段，避免写入凭据或完整剪贴板内容。

## 调试边界

按用户 2026-10-04 的最新安排，继续沿用现有 dev flow，暂不实施低干扰能力适配或 VM runner，也不新增针对当前桌面使用影响的执行限制。`dev` 仍启动原生应用，`browser` 使用真实 Go 服务；数据隔离及下述能力边界保持原有约定。

`browser` 使用 v3 `server` build tag，不是固定数据模拟。页面和 `/wails/runtime` 必须来自 9246 宿主；仅打开 9245 Vite 页面无法调用 Go。服务器强制绑定 loopback，事件由 Wails WebSocket 桥接。浏览器可验证真实插件加载、存储、搜索、接口和热更新；框架在 server 模式的原生窗口、剪贴板和对话框能力不能作为桌面验收。server 的自动剪贴板读取返回空内容，宿主呼出/隐藏为空操作，图片剪贴板写入明确报不支持，避免进入没有 AppKit 主线程的 Cocoa 桥接。使用 `dev` 验证这些原生能力。

可选的无 Go 固定场景：`VITE_WATOOLS_HOST=mock pnpm --dir frontend dev --host 127.0.0.1 --port 9259`，打开该 Vite 地址。默认 `dev` / `browser` 仍按原流程连接真实 Go。mock 实例提供固定命令/插件/更新/HTTP 结果、独立 JSON/凭据内存存储、合成文本剪贴板和 launcher 尺寸，可通过 `createMockHost(options)` 注入空列表、禁用等数据，`setHostAPI` 替换测试调用实现。`applicationQuery` 可注入应用查询失败、空结果和恢复场景；成功查询克隆并更新实例的命令快照，供 trigger/usage 使用，失败保留旧快照。文本写入更新版本，已取得的快照仍保留原文；不读写系统剪贴板。mock 不载入或执行真实插件模块，原生安装、文件对话框、图片剪贴板、下载/安装更新返回 `UNAVAILABLE`；旧 runtime 的其余原生窗口、ClipboardGetText 和 BrowserOpenURL 仍是兼容 Wails API，未纳入 mock。固定场景不能作为原生能力或权限隔离验收。

浏览器/桌面检查不混写：源码测试不等于 WebKit 交互，合成键盘事件不等于物理全局快捷键或真实输入法。迁移验收中可临时用 Wails 官方 `mcp` build tag 检查原生 WebView；默认产物没有 MCP 服务。插件默认主窗口打开，Command+D 分离独立小窗；该能力仅原生桌面可用，browser/mock 明确报不支持。完整 Go 断点、物理输入法／多屏窗口体验与远端 macOS CI 仍在对应 backlog。

实际版本、命令与验收边界见 [v3 迁移记录](../backlog/wails-v3-migration.md) 和 [开发工作流记录](../backlog/development-and-testing.md)。

验证记录见 [Go 工作流与原生验收](../reviews/evidence/2026-10-07/README.md)。

## 工作流实现

`cmd/watoolsctl` 与 `internal/workflow` 使用 Go 标准库及内核文件锁，不加载桌面服务。`cmd/pluginctl` 是 `.wt` 的底层打包／安装工具，`cmd/localinstall` 是准备完成后的安装提交助手；两者由统一 CLI 调度，没有第二套 Node 编排。Wails watcher 使用 `.cache/tools/watoolsctl`；管理中心在缓存内临时编译当前源码的 CLI，任务结束删除该可执行文件。旧收据的 `toolPath` 用于定位 Go，保存后移除旧 `nodeExecutable` 字段。

旧 `scripts/dev.mjs`、`scripts/mac-bundle.mjs`、`scripts/install.mjs` 和对应 Node 工作流测试已经移除。前端 `.cjs` 回归仍由 Node 执行，因为它们测试 JavaScript／插件行为。失败构建保留上次可用 bundle，成功构建替换完整 bundle 并清理过时资源。
