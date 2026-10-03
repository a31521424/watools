# macOS 本地开发与检查

[返回文档系统](../README.md) · [实施目标](../backlog/development-and-testing.md)

当前入口是仓库自带的 `scripts/dev.mjs`，使用 Node 标准库，不新增任务框架依赖。下述命令从仓库根目录运行；也可以在其他目录用脚本的绝对路径运行。当前适配 macOS 开发者自用，Windows 保留已有命令和 CI，统一入口的 Windows 适配延期。应用仍为 Wails v2。

## 环境与首次启动

- Node.js `>=22.12.0`，CI 使用 Node 22；旧说明中的 Node 18 不支持当前 Vite 7。
- Go 启动器：入口从 `go.mod` 读取 `toolchain`，设置 `GOTOOLCHAIN=go1.26.8`。本机更高版本的 Go 也使用这个固定工具链；首次可能下载。
- pnpm `10.21.0`：从 `frontend/package.json` 读取并检查，在前端目录执行命令。可使用 Corepack 或 pnpm 自身的版本管理；版本不匹配时入口报错。
- Xcode Command Line Tools 与 macOS SDK；通过 `xcrun --show-sdk-path` 检查，缺失时运行 `xcode-select --install` 并完成系统安装。
- Wails CLI 不要求全局安装：入口运行 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`，版本取自 `go.mod`，避免使用不同版本的全局 CLI。第一次运行会下载并编译 CLI 依赖。

```sh
node scripts/dev.mjs doctor
node scripts/dev.mjs setup
node scripts/dev.mjs plugins install
node scripts/dev.mjs dev
```

`doctor` 检查工具、SDK、前端依赖与路径；前端依赖未安装时提示 `setup`。`setup` 下载 Go 模块，并按 `pnpm-lock.yaml` 执行 `pnpm install --frozen-lockfile`。它不安装原生系统依赖，不修改全局工具设置。

Wails v2 的原生开发和打包会通过 `sysctl` 读取 macOS 系统版本。`doctor`、`dev`、`browser`、`build` 提前验证这个只读查询；agent 执行沙箱若禁止它，入口给出具体原因，需要为该原生任务授权系统信息查询后再运行。普通检查、插件工具和绑定生成不要求这个查询。本轮普通 Go 编译通过而 Wails 打包失败曾由此限制引起，不应据此修改应用代码或重装 SDK。

开发安装最初为空，`plugins install` 可打包并安装全部官方插件；也可只安装一个：

```sh
node scripts/dev.mjs plugins install watools.plugin.calculator
```

## 命令目录

| 命令 | 当前行为 |
| --- | --- |
| `doctor` | 检查固定 Go/pnpm、Node、SDK、依赖和目录；不启动 GUI |
| `setup` | Go 模块下载与前端锁文件安装 |
| `dev` | Wails 开发宿主、Vite 热更新和 Go 文件重建 |
| `browser` | 同时运行真实 Wails 宿主，打开浏览器调试入口 |
| `bindings` | 先构建嵌入资源，再通过固定 Wails CLI 重新生成绑定 |
| `check` | 工作流回归、前端回归和构建、Go 测试、vet 与本机生产标签编译；首个失败立即停止 |
| `check-race` | 构建前端后运行 `go test -race ./...` |
| `build` | 使用固定 CLI 构建当前平台原生应用，输出至 `build/bin/` |
| `plugins list` | 列出官方插件 |
| `plugins package [package-id...]` | 打包到开发包目录 |
| `plugins install [package-id...]` | 打包并安装到开发数据目录 |

`dev`、`browser`、`build` 可直接追加 Wails flags，例如 `node scripts/dev.mjs build -debug`。其他命令不接受额外 flags；插件命令仅接受 package ID，避免覆盖开发输出目录。Wails 开发和构建使用 `-m`，不自动执行 `go mod tidy`；调整 Go 依赖后应显式整理并审查锁定文件改动。前端安装始终使用 frozen lockfile。

Wails v2 会把部分生成 runtime 文件写成可执行权限；入口在 CLI 退出后将生成的 JS/TS/JSON 文件权限统一为 `0644`，避免绑定重新生成带来无意义的 Git 差异，不修改生成内容。

`check` 的 Go build 验证生产标签的编译，不生成完整 `.app` 包。需要原生包时另行执行 `build`。漏洞审查、插件打包和 Go race 仍由现有 CI 分别检查，`check` 不表示完整安全审查或原生 GUI 验收。

## 数据与缓存隔离

入口强制设置开发数据路径，覆盖调用者已有的 `WATOOLS_DATA_DIR` 与 `WATOOLS_CACHE_DIR`：

```text
.cache/
├── dev/
│   ├── data/       SQLite、插件、快捷键与本地凭据
│   ├── cache/      日志、图标与更新缓存
│   └── packages/   开发插件 .wt 包
└── tools/
    ├── go-build/
    ├── go-mod/
    └── gopath/
```

这些目录已在 `.gitignore` 覆盖范围内。`dev`、`browser` 与 `plugins install` 使用同一开发安装；修改官方插件后可重新打包安装，然后刷新开发宿主的插件列表。工具入口不是文件热重载器。

`check`、`check-race`、`bindings`、`build` 使用系统临时目录内的独立应用数据和缓存，正常结束或子命令失败时清理。它们不会使用或清空开发安装。开发模式日志默认 `debug`；可用 `WATOOLS_LOG_LEVEL` 指定其他级别，不记录凭据或完整剪贴板内容。

Ctrl-C/SIGINT 与 SIGTERM 转发给独立任务进程组，让 `go run` 背后的实际程序收到信号，并等待它完成清理。取消后即使子命令返回成功，也不会执行后续步骤。直属子进程退出后，最多再等待剩余任务进程组 5 秒；任务组退出后再清理临时数据和生成文件权限。若等待超时，入口报出进程组 ID、保留临时目录路径，不删除仍可能使用中的数据；这不是收到中断后整体保证 5 秒退出。Wails 启动的独立 Vite 进程组由 Wails 自身的退出流程清理。

Go 工具缓存默认在 `.cache/tools/`；允许显式设置 `GOCACHE`、`GOMODCACHE`、`GOPATH` 复用 CI 或 agent 的工具缓存，这不会改变应用数据隔离。入口设置 `GOWORK=off`，避免工作区上层的 Go workspace 改变构建依赖。

应用本身的目录语义也已明确：显式设置 `WATOOLS_DATA_DIR` 表示独立安装，只创建并使用该目录，不自动导入旧缓存数据。未设置此变量时，正式安装保留既有的旧缓存迁移流程。直接调用 `wails dev` 或 `go run ./cmd/pluginctl install` 不会获得统一入口的双目录隔离，仍可能操作正式安装。

入口不提供自动重置或迁移真实数据的命令。开发数据库、插件和 macOS 明文凭据也可能包含敏感内容，不要提交或分享整个 `.cache`。

## 调试边界

`browser` 使用 [Wails 的浏览器开发模式](https://wails.io/docs/reference/cli/#dev)，访问终端输出的宿主地址，通常为 `http://localhost:34115`。浏览器页面可以调用真实 Go 绑定；只运行 `frontend/pnpm dev` 的 Vite 页面无法独立提供宿主能力。浏览器调试仍需要原生宿主运行，不是无宿主的固定数据模拟模式。

React/样式修改由 Vite 热更新，Go 修改会重建并重启应用，进程内状态会丢失。开发构建开启 WebView DevTools，浏览器可直接使用自身 DevTools。使用源码断点调试 Go、固定数据 Host API、多窗口与插件会话端到端验收仍属于后续步骤，不能据此标记整个开发工作流目标完成。

## 本轮验证记录

实施日期：2026-10-03。本轮新增工作流集成测试使用临时检出目录和替代外部工具，验证目录带空格、不同工作目录、工具固定版本、环境覆盖、失败中止与临时数据清理；另覆盖多层子进程、独立服务组延迟退出，以及捕获信号后返回成功时取消后续任务。实际本机执行与原生验收结果见 [Backlog 实施记录](../backlog/development-and-testing.md)。
