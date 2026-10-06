# 2026-10-07 本机安装与管理中心验收

[证据索引](../README.md) · [本地流程](../../../development/local-workflow.md)

环境：当前 macOS arm64 开发机、Go 1.26.8、Node 24.11.0、pnpm 10.21.0、Wails v3 Beta.27。源代码为本轮未提交工作区。原生 QA 使用额外 `mcp` build tag，正式产物不含该服务；数据、缓存和目标应用全部位于一次性临时目录，没有使用真实用户数据做故障测试。

- `node scripts/dev.mjs check`：21 项工作流回归、57 项前端回归、前端类型检查与构建、Go 全包和 server 能力测试、vet、production 编译通过。
- `node scripts/dev.mjs check-race`：全包 race 检查通过。
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`：兼容编译通过，不代表 Windows 实机或安装验收。
- Go 故障测试覆盖：收据失败恢复旧应用及插件指针、拒绝覆盖其他应用、后续插件失败恢复此前升级、新插件注册回滚、重复包拒绝、跨进程安装锁；异步任务覆盖取消、并发拒绝、提交阶段延迟取消、参数与数据环境、输出上限。
- 独立目录实际运行 `node scripts/dev.mjs install --app-dir <临时目录>/Applications --no-launch`：成功安装签名校验的 `.app` 和 6 个官方插件，收据自动记录源码、Node、PATH 和目标位置。

[原生结果](./local-install-native.jsonl) 来自真实 AppKit/WKWebView 的 Wails MCP 探针。宿主以 `/usr/bin:/bin` 的精简 PATH 启动，模拟 Finder 不含开发工具路径；管理任务依靠收据仍能完成真实构建与安装。验证了管理中心自动配置、单插件 GUI 更新、离开页面后任务继续、整体更新与备份、存储／停用状态／第三方插件保留、运行中 JSON 小窗编辑内容保留，以及重启提示。探针只关闭自己的 QA 宿主。

取消和恢复覆盖程序正常处理的退出与错误；没有测试断电、强制 SIGKILL、磁盘损坏或多平台安装。旧 bundle 和插件 revision 保留，尚不自动清理。GitHub 商店、公开 Releases、签名公证和 Windows 安装不在本次实施范围。

## Go 工作流迁移验收

后续用户要求用 Go CLI 替代 Node 编排并清理残留。当前命令为 `go run ./cmd/watoolsctl`；上方 Node 命令和 21 项测试属于迁移前的历史结果，旧脚本及 Node 工作流测试已经移除。前端 JavaScript 回归仍保留。

- `go run ./cmd/watoolsctl check`：11 项 Go 工作流测试（包含参数表格场景）、57 项前端回归、全包 Go／server 测试、前端类型检查与构建、vet、production 编译通过。
- `go run ./cmd/watoolsctl check-race`：全包 race 检查通过。新增轻量 CLI、工作流和本地安装包的 Windows 交叉编译通过；不代表 Windows 原生验收。
- 工作流覆盖锁、参数、固定工具版本、临时数据、先准备后安装、无 Node 的插件更新、绑定权限、Wails watcher 命令、取消与提交、失败构建保留旧输出和成功后清除旧资源。管理任务覆盖先编译 CLI、收据迁移、输出限制和任务成功／失败后的临时可执行文件清理。
- [真实 go run 取消结果](./go-run-cancellation.jsonl)：只终止外层 Go 进程或向终端组发送 SIGINT 时，实际 CLI 都能清理子任务、报告取消并释放源码锁。[探针](./go-run-cancellation-probe.py.txt)使用临时最小源码与模拟工具，不碰正式数据。
- [原生开发结果](./go-dev-native.jsonl)与[探针](./go-dev-native-probe.py.txt)：带空格的独立检出使用共享依赖、独立 Vite 缓存，实际启动原生宿主与 Vite；终止外层 Go 进程后两者均退出，只关闭测试进程。
- 真实 Go 安装命令在独立目录安装本体和 6 个官方插件，收据记录 Go 并移除旧 Node 字段。
- [原生结果](./go-workflow-native.jsonl)与[探针](./go-workflow-native-probe.py.txt)：在精简 PATH 下从旧 Node 收据定位 Go；GUI 单插件和本体／全部插件更新成功，保留存储、停用状态、第三方插件及运行中的 JSON 编辑器，任务离开页面后继续，临时编译的 CLI 在结束后清除。
- 完成上述隔离验收后，实际执行 Go 安装命令覆盖 `/Applications/watools.app` 和正式目录的 6 个官方插件；签名校验通过，正式收据已记录 Go 且没有旧 Node 字段。该步骤为用户授权的正常安装，不用于故障测试。QA 目录、测试应用与旧 Node watcher 配置已清理，恢复备份与插件 revision 保留。

统一 CLI 为轻量 Go 工具；`cmd/pluginctl` 和 `cmd/localinstall` 分别是其打包与提交助手。旧更新页及孤立 API 包装也已删除；GitHub 更新后端保留，公开发布仍暂停。

## 提交前交叉审查

用户要求三路 subagent 审查全部工作区，分别覆盖工作流、安装事务及管理中心接口。安装事务未发现阻塞问题；其余审查复现并修复了以下问题：

- 非提交任务忽略 TERM 时，直属进程等待无上限；残余任务组也可能在源码锁释放后继续运行。现改为取消时在并发收尾流程中等待整组退出，10 秒宽限后 KILL，输出管道有界收尾；提交助手继续等待完成或恢复。新增行为测试分别覆盖直属进程和残余子进程，确认源码锁在收尾前保持、结束后释放且进程组不存在。
- 管理中心过早标记更新后的插件刷新完成，临时失败后不再重试；插件获取失败也被吞掉。现仅成功刷新后标记任务，获取失败向上报告，轮询重试成功后清理刷新错误。新增两项前端回归分别覆盖刷新重试和获取失败后的新 revision 载入。

修复后前端回归增至 59 项，Go 工作流测试增至 12 项（新增测试含两种取消场景）。全量 `check`、相关工作流／管理器 race、轻量工具的 Windows 交叉编译均通过；独立审查探针复跑确认两种取消任务组均在约 10 秒后消失且可安全清理。再次执行原生开发探针，宿主与 Vite 启动及外层 Go 退出后的收尾均通过。安装和原子提交关系复核通过，没有剩余阻塞问题。原生安装与开发验收的既有证据仍保留；上述新增边界使用隔离数据和测试进程验证。


## 全页面与官方插件 UI 统一

本次将启动器、管理中心、插件列表与详情、日志与详情，以及五个官方 UI 插件统一为浅色桌面风格。应用绘制的 `WaWorkspaceHeader` 展示返回、图标、名称、品牌、简介和页面操作；独立插件窗不重复绘制这条标题栏，继续采用原生标题栏和可缩放窗口。唯一设计源随 UI 插件打包，五个 UI 插件升级至 0.3.0。

- `go run ./cmd/watoolsctl check`：60 项前端回归、全包 Go／server 测试、共享样式打包检查、类型检查与前端构建、vet、production 编译通过。新增回归确认浏览器没有原生窗口尺寸时不发送无效缩放。
- 浏览器 QA 使用独立数据与本地 server 构建，通过 agent-browser 检查启动器、管理分类、插件列表和详情、日志及全部插件。JSON 格式化、二维码生成与文本统计经过实际操作检查；翻译检查设置和布局，未使用真实密钥或发起外部翻译请求。
- 窄窗复核 QR 与翻译的上下布局、滚动与无横向溢出。修正基础 CSS 优先级后，工具的响应式规则可正常覆盖共享默认值。
- [原生结果](./ui-redesign-native.jsonl)与[探针](./ui-redesign-native-probe.py.txt)：独立标识的 macOS QA 应用加载五个插件，验证统一背景与已加载的共享样式、计算器求值、二维码生成、文本统计和 JSON 格式化。Command+D 后编辑内容保留，主窗恢复 64px，并在 macOS 窗口动画结束后从屏幕窗口列表移除；同一 JSON 工具可分离两次。原生缩放至 480×320、640×500、1100×800 后内容仍保留；其余四个工具的独立窗在 480×320 无横向溢出。窗口尺寸由原生窗口 API 调整，未模拟物理拖动窗口边缘。探针仅关闭自己的 QA 进程。
- [JSON 工作区](./ui-json.png)、[管理中心](./ui-management.png)、[二维码窄窗布局](./ui-qr-narrow.png)为浏览器视图截图；不把它们当作原生标题栏的像素证据。

当前视觉与插件编写规范见 [UI 风格](../../../design/ui-style.md)和[插件 UI 指南](../../../plugin-development/05-ui-and-shortcuts.md)。未实现系统主题切换；常用工具插件为 executable 入口，无网页 UI。

正式安装已完成：`go run ./cmd/watoolsctl install --no-launch` 覆盖 `/Applications/watools.app` 并安装六个官方插件；五个 UI 插件为 0.3.0，安装目录中的样式与共享源逐字节一致。系统应用签名校验通过。正常安装保留旧应用备份、用户配置、插件数据及已运行窗口；需要保存编辑内容并重新打开应用才能使用新版宿主。QA 浏览器、宿主和独立测试目录在验证后清理。
