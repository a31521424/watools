# 开发工作流与测试

[返回 Backlog](./README.md)

状态：in-progress。优先级：P0。基础入口已实施；可替换的类型化宿主适配依赖 [服务接口](./services-and-host-api.md)，原生多窗口检查依赖 [小窗实现](./plugin-windows-and-sessions.md)。目标是开发者和 agent 能稳定启动、复现、检查与定位问题。

按 [自用需求](../requirements/developer-self-use.md)，首先保证本机开发、浏览器调试与 macOS 验证。Windows 环境与实机兼容延期，不作为本阶段完成条件。

## 现状问题

前端已有 Vite、构建和回归命令，CI 覆盖 Go、前端和插件打包。现在新增统一 macOS 任务与隔离开发安装，详见 [本地开发指南](../development/local-workflow.md)。宿主调用与绑定仍紧耦合；无宿主浏览器模拟、固定数据场景和原生验收仍需实施。现有检查不等于完整 GUI 测试。

## 实施范围

- 统一应用开发、浏览器前端开发、绑定生成、检查、构建和插件打包入口；命令名称实施时确定。
- 固定版本和锁文件，增加环境诊断，明确平台依赖与启动失败原因。
- 模拟模式使用有类型的 Host API 与固定数据，覆盖空列表、异常、禁用、安装失败和多实例场景。
- 使用独立数据和缓存目录；初始化、重置只作用于这些目录，不隐式迁移或清空真实用户数据。
- 配置 sourcemap、WebView DevTools 与 Go 断点调试，明确前端热更新与 Go 重启的差异。
- 补充关键行为和 API 契约测试，维护少量稳定端到端场景，不为每个包装方法复制实现式测试。
- 优先让本地检查可复现，再在已授权的远端工作流中实际运行 macOS CI 并记录结果。Windows CI 的适配与运行归入延期任务，不为了获取 CI 结果自行推送或发布。

## 验收与验证

- 新检出工作区按说明安装依赖后，可用统一入口启动和检查。
- 界面能在浏览器调试，切换真实宿主无需修改业务组件。
- 固定数据的场景可重复复现，失败输出足以定位问题。
- 保留已有 CI，优先验证本地与 macOS 路径；远端结果单独记录，尚未触发不能写为通过。Windows 路径的实际运行与修复见 [延期任务](./windows-compatibility.md)。
- 原生场景包含呼出、搜索、打开两插件窗、隐藏启动器、关闭重开、禁用与重装。
- 浏览器测试不替代 WebKit、输入法、焦点、权限和真实更新重启验收。

## 实施记录

### 2026-10-04 确定低干扰开发与测试方案

用户反馈原生测试持续干扰主机使用。此前隔离了应用数据，却仍共用当前桌面的前台焦点、全局快捷键、剪贴板和对话框；改用 CLI/MCP 不会改变这些副作用。已在 AGENT.md 与本地指南登记执行约束，下面的适配和 runner 尚未实施，不能宣称已有完全无副作用的 browser 模式。

| 层级 | 执行环境 | 覆盖与边界 |
| --- | --- | --- |
| 日常开发与快速检查 | 前端浏览器/无界面浏览器；Go 无 GUI 测试 | 搜索、状态、插件 UI、存储及错误行为。窗口、剪贴板、快捷键、对话框及外部应用/系统命令使用内存夹具或动作记录 |
| 真实 Go 集成与 UI 回归 | Wails server + 无界面 Chromium/WebKit | 使用真实绑定、事件、资源路由、临时 SQLite 和插件 revision，只替换会影响主机的系统能力；不冒充原生 WKWebView、IME 或焦点验收 |
| 原生自动化 | 独立 macOS VM 或测试机 | AppKit 生命周期、原生 WebView、菜单、对话框、焦点、快捷键注册与权限状态；关闭宿主/访客剪贴板及共享用户目录等桥接，限制资源和并发 |
| 最后少量人工验收 | 用户指定的主机时间窗口 | 真实中文候选/提交、物理全局快捷键、多屏/热插拔及硬件相关体验；说明操作与时间，完成后关闭测试进程并记录剩余边界 |

实施顺序：

1. 先收紧 browser/test profile 的系统副作用。为剪贴板、窗口/快捷键、文件选择与外部命令执行设置可替换的明确边界；生产实现保持真实 macOS 行为，测试实现用内存内容/预设路径/动作记录，不让未实现能力落回真实系统。不要仅在 React 层 mock 或给所有业务接口复制一份假后端。
2. 增加无界面浏览器关键路径回归：搜索、插件打开、使用计数、存储、禁用/重装、清空与异步旧响应、失败提示。真实 Go 集成保留数据库/插件服务；纯 UI 模式通过类型化 Host API 使用固定场景。维护适配契约检查，限制日常测试并发，控制编译和 VM 的 CPU/内存占用。
3. 把有价值的原生探针整理为独立 VM runner，记录 OS/版本/测试环境与超时清理。VM 未准备好时，不自动回退到用户主机。VM 使用合成数据，不挂载用户目录、不连接用户账号，不开启共享剪贴板；访客 UI 操作仅在访客会话发生。
4. 按变更影响选择测试范围：纯业务/UI 改动运行前两层；Cocoa/Wails/快捷键改动追加 VM 原生回归；真实输入法/硬件相关变更安排主机人工窗口。失败优先保留日志、trace 与自己的页面快照，避免一边修改一边反复重启主机 GUI。

沿用当前 `browser/check/check-race/build` 入口；新增 profile/UI/VM 测试入口的名字在实施时确定，不把计划命令写成现有能力。主机上新建另一个 Space、另一个临时目录或仅把窗口隐藏，均不作为原生隔离方案；容器不能替代 macOS AppKit 桌面验收。

官方依据：[Wails server build](https://v3.wails.io/guides/server-build/)、[Playwright 浏览器与 headless 支持](https://playwright.dev/docs/browsers)、[Apple silicon 上运行 macOS VM](https://developer.apple.com/documentation/virtualization/running-macos-in-a-virtual-machine-on-apple-silicon)。Wails 滚动文档不覆盖本仓库固定 Beta.27 的实际 API；实施以固定源码及真实回归为准。Apple 支持 macOS VM，不代表 VM 可以覆盖用户主机的物理键盘、输入法、显示器拓扑或全部 TCC 情况。

### 既有工作流实施记录

2026-10-03 开始推进。基础阶段提交主题：`feat(dev): add isolated macOS development workflow`，对应提交哈希见 Git 历史。

已实现基础阶段：

- `node scripts/dev.mjs` 提供 doctor/setup/dev/browser/bindings/check/check-race/build/plugins 入口，固定 Go、pnpm 和 Wails CLI 版本来源，安装使用锁文件。
- Wails 浏览器开发模式调用真实宿主；开发、插件安装与包输出统一使用 `.cache/dev/`，检查、绑定和构建使用一次性应用数据与缓存。
- 修复显式 `WATOOLS_DATA_DIR` 仍自动导入旧缓存的问题，正式安装的默认迁移保留；增加隔离、参数转发、工具版本、失败/中断清理和绑定权限回归，并接入 macOS CI。
- 更新开发指南及入口说明，Windows CI 保留现状。

基础阶段（v2 历史）本机验证：macOS 26.6.2 / arm64，Node 24.11.0，固定 Go 1.26.8、pnpm 10.21.0、Wails 2.10.2。Go 工具缓存复用独立 `/private/tmp` 目录，应用数据仍由任务入口隔离。

| 命令或检查 | 实际结果 |
| --- | --- |
| `doctor` | 工具版本、SDK、系统版本与隔离路径符合预期 |
| `setup` | Go 模块下载、前端 frozen lockfile 安装成功，锁文件未修改 |
| `check` | 14 个工作流回归、10 个前端回归、前端构建、Go 全包测试/vet/生产标签编译通过 |
| `check-race` | 前端构建与 Go 全包 race 检查通过 |
| `bindings` | 固定 CLI 生成成功；绑定内容未变化；三份已有 Go 绑定模块的执行权限由生成入口规范为 0644 |
| `build` | 生成 `build/bin/watools.app`，二进制确认是 arm64 Mach-O；未启动 GUI |
| `plugins install` | 6 个官方插件打包安装成功，数据库与 revision 均位于 `.cache/dev/data/`，没有导入旧缓存的 migration 标记 |
| 文档与差异 | 本地 Markdown 引用与完整目录检查通过，`git diff --check` 通过 |

沙箱限制记录：Wails v2 打包依赖只读 `sysctl` 系统版本查询，初次执行被 agent 沙箱禁止；允许该系统查询后打包成功。入口已增加针对性诊断。普通 Go 编译通过不代表此原生打包步骤已执行。

未执行：开发宿主启动与浏览器页面的实际交互、原生 GUI/输入法/快捷键验收、远端 macOS CI。工作流回归中的替代工具只验证入口行为，不冒充原生交互验收。前端 `package.json.md5` 由 Wails 构建按新的环境声明自动更新。

提交前独立 subagent 审查发现信号只转发到 `go run` 会挂起或留下后代，以及捕获中断后返回 0 会继续后续任务。已改为独立进程组转发、等待退出和取消状态判断，并新增 SIGINT/SIGTERM、多层子进程、独立服务组延迟清理与取消后成功返回的回归。

独立复审通过：14 个工作流回归通过；真实 Go 1.26.8 的 `go run → 无 GUI 模拟宿主 → 独立服务进程组` 探针在 SIGINT/SIGTERM 下均等待延迟清理完成，宿主和服务没有残留。原信号与取消问题关闭，未发现新的提交阻塞点。此探针不属于原生 GUI 验收。

2026-10-03 v3 迁移阶段已把上述入口升级到固定 Beta.27，生成目录改为 `frontend/bindings/`，macOS 打包使用当前模板，开发 watcher 显式等待 Vite readiness。浏览器真实宿主启动、搜索、旧计算器加载/计算和存储已实际验证；16 个工作流回归、11 个前端回归及 Go 检查通过。当前命令与路径以 [本地开发指南](../development/local-workflow.md) 为准；上表与 Wails v2/sysctl/md5 描述属于此前提交的历史结果，不再是当前流程。原生测试证据和边界见 [迁移记录](./wails-v3-migration.md)。

剩余目标：类型化 Host API 与固定数据模拟；Go 源码断点流程；更完整的失败复现场景与契约测试；多窗口和完整原生 GUI 验收；在用户授权推送后实际运行 macOS CI。保持 in-progress，不以基础入口代替整项验收。
