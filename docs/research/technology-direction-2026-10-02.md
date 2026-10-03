# WaTools 桌面框架与下一轮技术方向调研

[返回文档索引](../README.md) · [全仓审查](../reviews/review-2026-10-02.md)

**当前推进方向（2026-10-03）：在框架比较后，下一轮优先迁移 Wails v3、增加插件小窗并优化开发调试。目标已登记到 [backlog](../backlog/README.md)，尚未实施。下文保留前期候选分析与推荐讨论，不作为当前执行顺序。macOS Keychain 已按要求移除。**

调研日期：2026-10-02。基于当前仓库与框架官方资料，讨论插件独立窗口、Wails 生态、Go 是否继续承担主要开发工作，以及下一轮改造的推进顺序。本轮没有实现框架迁移，也没有实测候选框架的性能或窗口行为。

建议把 Electron 与 Wails v3 作为两个主要候选，先用同一组真实功能验证。按“长期扩展插件、主要依赖 agent 开发、没有紧迫工期”的条件，我倾向优先验证 Electron + TypeScript；如果轻量常驻和复用 Go 后端的权重更高，且 Wails v3 的窗口验证通过，继续使用 Go + Wails v3 完全合理。Tauri v2 是重视小体积、愿意维护 Rust 能力层时的候选。

当前按“插件可打开多个独立窗口，启动器隐藏后仍能保留”的需求评估；是否还需要附着启动器的临时浮窗，尚待确认。以下选型倾向属于项目判断，不是框架性能排名。

## 2026-10-03 补充：以简化长期开发为目标

当前用途是常驻启动器、应用搜索、本地工具插件与独立插件窗口，现有插件主要是 HTML/JavaScript。选型应先保留 React、TypeScript、插件内容与用户数据，只调整宿主和运行边界；更换 UI 框架、数据库与插件格式不是这次选型的必要条件。

如果主要目标是宿主、SDK 与插件共享语言和类型，建议优先验证 Electron + TypeScript。如果体积和常驻资源是硬指标，Tauri v2 是另一条路线；若重用 Go 更重要，Wails v3 仍是成本较低的选择。这是针对本项目的判断，尚未实测候选应用，不能据此承诺某个框架在此项目的内存或响应速度。

| 候选 | 与当前场景的关系 | 必须承担的成本 |
| --- | --- | --- |
| Electron + TypeScript + React | 宿主、SDK、插件可共享 TS 契约；BrowserWindow 提供插件独立窗，WebContentsView 提供独立 Web 内容视图 | 分发 Chromium/Node 与持续更新；应用扫描、图标和特殊系统行为仍需平台实现；慢任务不能阻塞主进程 |
| Tauri v2 + Rust + React | 系统 WebView，支持按窗口与 WebView 配置能力，适合重视轻量分发的路线 | Go 原生服务改写为 Rust，仍有两种业务语言及系统 WebView 差异；权限配置不能代替插件身份校验 |
| Wails v3 + Go + React | 大量现有 Go 能力可保留；原生多窗口已可作为独立插件窗的基础 | v3 迁移和平台验收；Go/TS 桥接仍需维护，当前全局插件运行模型仍需重构 |
| Electrobun + TypeScript + React（新候选） | TS 主进程、系统 WebView，可选固定 CEF；具备多窗口和隔离嵌入视图的构建基础 | 新运行时与工具链、npm 包兼容、两平台原生行为和自动化需要验证，不能直接按宣传体积判断真实应用资源 |

框架事实来自 [Electron 进程模型](https://www.electronjs.org/docs/latest/tutorial/process-model)、[WebContentsView](https://www.electronjs.org/docs/latest/api/web-contents-view)、[Tauri 架构](https://v2.tauri.app/concept/architecture/)、[Tauri capabilities](https://v2.tauri.app/security/capabilities/) 与 [Wails v3 多窗口](https://v3.wails.io/features/windows/multiple/)。表中的迁移成本与优先级是本仓库的工程判断。

Electrobun 值得新增到原型名单。此次官方文档已是 2.x 路线：默认 TS 运行时是 Cottontail，构建工具是 Hutch，Bun 是可选项，也支持 Go/Rust 等原生主进程。不能继续用早期“只有 Bun + TypeScript”的资料做选型。系统 WebView 与可选 CEF，以及内嵌隔离内容见 [官方概览](https://framework.blackboard.sh/electrobun/) 与 [Creating UI](https://framework.blackboard.sh/electrobun/guides/creating-ui/)。官方 [2.x 迁移指南](https://framework.blackboard.sh/electrobun/guides/migrating-to-v2/) 说明工具链也在改变；对希望降低维护复杂度的本项目，我建议先做独立原型，暂不直接替换主仓库。

Electron 的价值在于把平台编排、SDK 契约和插件逻辑集中到 TS，并使用 Chromium 渲染；Playwright 有实验性 [Electron 自动化入口](https://playwright.dev/docs/api/class-electron)。这是开发与验证路径上的优势，不能推导出 agent 必然更少出错。插件侧仍需禁用 Node integration，配置 sandbox、context isolation，并由宿主验证 IPC 来源与权限，参见 [Electron 安全指南](https://www.electronjs.org/docs/latest/tutorial/security)。

建议下一步只验证一个完整场景：快捷键呼出启动器，搜索后打开翻译插件小窗，隐藏启动器，小窗继续输入与翻译；再打开第二个实例、关闭并重新打开，检查存储和资源清理。对这个场景同时测冷启动、空闲 CPU、进程树内存、窗口焦点、中文输入法与安装体积。性能指标先从当前应用测基线，不预设框架宣传数字就是验收标准。

无论换不换栈，建议把职责压成三个边界：宿主拥有窗口、插件会话、存储和原生服务；插件使用有类型的显式 SDK；React 负责视图与临时输入状态。启动时读 manifest 和声明式匹配规则，真正执行或打开时才加载插件代码。CPU 任务进可终止的 worker，第三方插件只获得所需的宿主调用。这样才能减少当前全局 API 注入、状态多处缓存和跨语言接口扩散带来的维护成本。

macOS 已按用户要求移除 Keychain，凭据使用应用目录内的明文文件（目录 0700、文件 0600）；Windows DPAPI 保留。宿主 SDK 不再把 Secret API 等同于所有平台都加密。此改动不依赖将来选用的桌面框架，也未迁移或删除真实系统凭据。

## 当前仓库的约束

当前使用 Wails v2.10.2，Go 1.26.0，前端为 React 19、TypeScript、Vite 和 Zustand。主要依据是 [`go.mod`](../../go.mod)、[`frontend/package.json`](../../frontend/package.json) 和 [`main.go`](../../main.go)。

独立窗口受两层约束影响：框架采用 v2 的单窗口模型；应用内部也围绕一个启动器页面组织插件运行。后者需要在任何候选框架中重构。

| 当前实现 | 多窗口带来的具体问题 | 建议的边界 |
| --- | --- | --- |
| UI 插件在 `WaPlugin` 的 iframe 中，通过主页面注入 runtime 和 API | 新窗口不能依赖原主页面替它完成初始化 | 插件视图自行连接受限 SDK |
| 启动上下文当前存入主页面内存 Map（旧版曾使用 `sessionStorage`） | 不应假设另一个顶层窗口能读取启动器的会话存储 | 宿主管理会话，通过受控通道交付上下文 |
| 插件状态与使用次数在页面级 Zustand store 中缓存 | 多窗口各自计数、刷新、写回，可能发生覆盖和状态不一致 | 宿主作为状态权威，窗口订阅事件，计数按增量写入 |
| enabled 插件的入口模块在主页面动态 import | 慢 matcher 或异常插件影响启动器；多个实例的执行上下文难以管理 | 按需加载，独立执行环境，超时及终止机制 |
| macOS 通过 key/main/第一个窗口寻找“当前窗口”，隐藏操作使用应用级 `hide` | 容易操作错窗口，隐藏启动器时连带隐藏插件窗口 | 明确区分 launcher、plugin、settings，并按 ID 操作 |

代码依据：[`wa-plugin.tsx`](../../frontend/src/components/watools/wa-plugin.tsx)、[`plugin-context.ts`](../../frontend/src/lib/plugin-context.ts)、[`pluginStore.ts`](../../frontend/src/stores/pluginStore.ts)、[`api/plugin.ts`](../../frontend/src/api/plugin.ts)、[`app_darwin.go`](../../internal/app/app_darwin.go)。

尤其需要明确“展开”的状态语义：第一次实现可以序列化插件状态，再在独立窗口重建视图。把原 iframe 的实时 DOM 和 JavaScript 执行环境直接搬进原生窗口，不应作为所有框架都能满足的默认要求。

## Wails 当前能力与生态

### 版本与维护状态

查询时最新稳定版为 v2.14.0，仓库的 v2.10.2 已落后。v3 最新变更记录为 2026-10-01 发布的 v3.0.0-beta.27，仍处于 Beta。近期持续发布，不能因为本项目停更就判断 Wails 已停止维护。[v2 稳定版发布页](https://github.com/wailsapp/wails/releases/tag/v2.14.0)、[v3 状态](https://v3.wails.io/status/)、[v3 变更记录](https://v3.wails.io/changelog/)。

v3 可以进入验证阶段；正式采用时要固定版本，建立自己的平台回归和升级节奏。它目前仍在修复 WebView、权限和平台生命周期问题，因此“文档中有 API”与“WaTools 的实际体验通过验收”需要分别确认。

### 插件独立窗口与浮窗

v3 原生支持动态创建、管理和关闭多个窗口，可用于工具窗口、编辑器和设置页。macOS 窗口配置提供 NSPanel、NonActivating、窗口层级及 Spaces/全屏参与选项。这些能力与启动器和插件浮窗的需求直接相关。[多窗口](https://v3.wails.io/features/windows/multiple/)、[窗口选项](https://v3.wails.io/features/windows/options/)。

建议启动器使用符合呼出体验的面板，长期工作的插件默认使用可正常激活、输入和切换的独立窗口，并提供置顶选项。不要把所有插件都设为不激活浮窗；有文本输入、文件对话框和快捷键的插件需要独立验证焦点行为。

升级 v3 涉及应用生命周期、窗口对象、services、事件和生成绑定的迁移，并非仅修改依赖版本。当前仓库的 macOS 原生窗口代码也需要重新检查，优先用框架能力替代能替代的部分。[v2 到 v3 迁移指南](https://v3.wails.io/migration/v2-to-v3/)。

### 平台能力与生态的边界

v3 的 manager API 已覆盖窗口、屏幕、菜单、托盘、对话框和剪贴板等常用宿主功能，并有全局快捷键和应用更新模块。更新模块提供签名校验及替换流程，可评估是否替代当前自写的更新安装代码。[Manager API](https://v3.wails.io/concepts/manager-api/)、[全局快捷键](https://v3.wails.io/features/keyboard/global-shortcuts/)、[应用更新](https://v3.wails.io/guides/updater/)。

生态方向正在从基础 Go 绑定走向可复用 service，service 可以组合后端接口和前端资源。但官方明确说明通用插件系统不属于当前 Beta。它不能直接替代 WaTools 的 `.wt` 包安装、动态插件加载、权限、版本兼容和市场机制。[v3 Beta 说明](https://v3.wails.io/blog/wails-v3-beta/)。

对本项目的判断是：Wails v3 作为桌面壳的能力已明显改善；作为任意第三方插件的平台，仍需要自己设计运行时与授权。macOS 使用 WebKit、Windows 使用 WebView2，跨平台 Web 能力差异仍需测试。[Wails 状态与平台支持](https://v3.wails.io/status/)。

### 对 agent 开发的支持

v3 有实验性的应用 MCP，可枚举和控制窗口、查询 DOM、模拟输入、执行 JavaScript、调用 Go 方法；CLI MCP 另用于项目与构建操作。应用 MCP 的 DOM 快照不是像素截图，不能代替对透明度、窗口层级、真实焦点和系统权限对话框的视觉及原生验证。建议只在开发构建启用，发布构建排除该 build tag。[应用 MCP](https://v3.wails.io/guides/mcp-service/)、[CLI](https://v3.wails.io/reference/cli/)。

这使 Wails v3 值得纳入 agent 工作流验证，不能仅凭旧版经验认定它难以自动测试。

## Go 是否适合继续作为主要语言

适合。当前后端的文件扫描、监听、SQLite、安装管理、HTTP、任务调度与平台桥接，没有显示出必须换语言才能解决的需求。上一轮审查发现的数据丢失、生命周期、权限和共享上下文问题，主要来自状态所有权和接口设计；改写成 Rust 或 TypeScript 后仍可能重复出现。

但“Go 适合后端”与“所有新功能都应该优先用 Go 实现”是两个决定。WaTools 的 UI 和现有插件本来就在 Web/JavaScript 一侧。如果未来新增工作主要集中在插件 SDK、窗口交互、编辑器和 Web 工具，TypeScript 承担更多平台编排工作会减少跨语言接口维护。这是针对本仓库的开发结构判断。

| 语言职责 | 适合承接的工作 | 需要注意的限制 |
| --- | --- | --- |
| Go | 索引、文件任务、数据库、后台服务、已有平台能力 | 原生桌面能力依赖框架及平台桥接，语言本身不提供完善窗口模型 |
| TypeScript | React UI、插件 SDK、插件逻辑、Electron 宿主编排 | CPU 密集或不可信插件不能直接塞进主线程/主进程 |
| Rust | Tauri 宿主、需要 Rust 库的系统能力、经测量确定的性能热点 | 类型安全无法自动解决插件越权、业务事务和窗口状态错误 |

主要依靠 agent 开发时，建议比较一次完整改动能否被稳定验证：接口是否有类型、错误是否可观测、能否自动启动应用并操作多个窗口、失败是否能定位。不要假设某个语言天然具有更高的 agent 正确率。

## 三个候选方向

### Electron 与 TypeScript

适合作为优先验证的迁移方案。Electron 提供 BrowserWindow 和 WebContentsView，主进程负责桌面能力，渲染进程承载 Web UI；可以让独立插件窗口脱离启动器存活。内嵌插件视图应评估 WebContentsView，官方目前不推荐使用 `<webview>` 标签。[进程模型](https://www.electronjs.org/docs/latest/tutorial/process-model)、[Web 嵌入](https://www.electronjs.org/docs/latest/tutorial/web-embeds)。

它允许宿主编排、SDK、插件和 UI 共享 TypeScript 的工具链与接口定义。Playwright 提供实验性的 Electron 自动化接口，能连接应用及其窗口；能否覆盖 WaTools 的原生交互仍需实际验证。[Playwright Electron](https://playwright.dev/docs/api/class-electron)。

代价是随应用分发 Chromium 和 Node.js 带来的体积与运行资源基线，以及持续更新这些组件的责任。统一 Chromium 有利于减少插件浏览器兼容变量；不会消除 macOS 与 Windows 的系统行为差异。资源开销需要测量，本文没有给出未经实测的内存倍数。[Electron 进程模型](https://www.electronjs.org/docs/latest/tutorial/process-model)。

安全设计建议：插件渲染侧禁用 Node integration，开启 context isolation 和 sandbox，使用按插件受限的 preload 接口；宿主核验 IPC 来源、权限、导航及资源访问。独立进程不能代替这些校验，utilityProcess 也不能直接被当作执行恶意 Node 插件的安全沙箱。[Electron 安全指南](https://www.electronjs.org/docs/latest/tutorial/security)、[BrowserWindow](https://www.electronjs.org/docs/latest/api/browser-window)、[utilityProcess](https://www.electronjs.org/docs/latest/api/utility-process)。

### Wails v3 与 Go

适合作为升级方案，尤其在常驻资源、Go 后端复用和较少迁移变量更重要时。React 可以保留，后端业务与数据库层有较多复用空间。需要重做宿主适配、插件会话和受限桥接，并移除单窗口假设。

主要不确定性是 Beta 阶段的平台行为、插件对不同系统 WebView 的兼容性，以及自建插件隔离机制的成本。只有一个小窗需求时，直接抛弃 Go 的理由不充分；如果要让第三方插件成为长期产品核心，则需要把运行时建设成本纳入比较。

### Tauri v2 与 Rust

Tauri 使用系统 WebView，提供多窗口以及按窗口/WebView 配置的 capabilities；官方插件覆盖快捷键、剪贴板、文件系统、SQL、更新等常用能力。[架构](https://v2.tauri.app/concept/architecture/)、[多窗口与权限](https://v2.tauri.app/learn/security/capabilities-for-windows-and-platforms/)、[官方插件目录](https://v2.tauri.app/plugin/)。

它更适合把轻量分发与明确的原生能力授权作为较高优先级的路线。需要注意自定义 commands 默认允许所有窗口调用，必须主动纳入权限配置；capabilities 也不能替代业务层的插件身份和资源范围校验。[Capabilities](https://v2.tauri.app/security/capabilities/)。

自动化能力也不能按旧资料判断。当前官方推荐 WebdriverIO 的 `@wdio/tauri-service`，其嵌入式 WebDriver 路径支持 Windows、Linux 和 macOS；不应笼统说 Tauri 无法在 macOS 自动测试。[Tauri WebDriver](https://v2.tauri.app/develop/tests/webdriver/)。

对本项目，它同时引入框架和后端语言迁移，而系统 WebView 差异仍然存在。如果只是希望 agent 开发更顺畅，没有必要先承担 Rust 迁移。保留 Go sidecar 在技术上可行，但会增加进程生命周期、IPC、打包和更新协调；不建议仅为保留少量旧代码就长期维持三语言核心架构。

## 项目适配比较

下表中的成本与适配评价是基于当前仓库的判断，功能事实见上文官方来源。

| 判断维度 | Wails v3 | Electron | Tauri v2 |
| --- | --- | --- | --- |
| 原生多窗口 | 有 | 有 | 有 |
| 当前后端复用 | 较高 | 部分逻辑重写或阶段性保留 Go | 部分逻辑重写或阶段性保留 Go |
| Web 渲染环境 | 系统 WebView | 随应用分发 Chromium | 系统 WebView |
| 插件 SDK 与宿主统一为 TS | Go 仍参与宿主 | 更直接 | Rust 仍参与宿主 |
| 能直接获得 `.wt` 动态插件平台 | 不能 | 不能 | 不能 |
| 权限基础 | 需自建插件授权桥接 | sandbox/context isolation 配合自建 IPC 授权 | capabilities 配合自建插件身份授权 |
| agent 验证入口 | 实验性 MCP | Playwright 的实验性 Electron 接口 | WebdriverIO Tauri service |
| 当前主要代价 | Beta 与运行时建设 | 常驻资源与安全维护 | 后端重写及新工具链 |
| 建议定位 | 轻量与 Go 复用优先时采用 | 插件平台与 TS 开发闭环优先时采用 | 小体积与 Rust 能力层优先时采用 |

## 下一轮值得推进的架构

建议先确定插件和宿主的稳定边界，再选择承载它的框架。React 组件、插件资源、manifest 内容与用户数据库都可以逐步迁移，没有必要同时重写全部产品功能。

1. **插件会话独立于窗口。** 宿主管理 pluginId、version、entryId、instanceId、launchContext 和 lifecycle。启动器负责发起命令，插件会话可以由内嵌视图或独立窗口显示。事件、取消和清理按会话归属管理。
2. **运行代码与搜索声明分离。** manifest 保存触发条件和能力需求，启动时先读元数据；选中插件后再加载 UI 或执行逻辑。保留必须动态匹配的插件时，将 matcher 放进可终止的执行环境，并限制预算。
3. **SDK 采用明确授权。** 发布有版本的 TypeScript API，插件只获得所需能力。宿主从可信连接/窗口会话识别调用方，不信任请求里自报的 packageId。文件授权需范围控制，网络授权需域名或目标范围，密钥存储与普通插件 storage 分离。
4. **插件安装与用户数据分离。** 插件版本目录不可变，用户数据使用独立目录；升级先验证和准备新版本，再切换并保留回滚。启停、卸载、升级要与运行会话协调，修复上一轮审查发现的存储丢失问题。
5. **平台能力显式报告可用性。** 接口返回 supported、permissionRequired、temporarilyUnavailable 等状态。先把应用扫描、图标、文件定位、热键、焦点和剪贴板做成可靠服务，再按产品需求扩展 OCR、选中文本、辅助功能等能力。
6. **把可验证性作为平台功能。** 输出按插件及会话区分的日志，记录启动耗时、异常和权限拒绝；提供本地开发、重载、SDK 类型检查、打包校验与兼容测试，让 agent 有完整的修改与验证闭环。

官方自带的“框架插件”通常用于给应用集成原生能力，和用户运行时安装的 WaTools 插件属于不同层次。无论选谁，以上边界都需要 WaTools 自己拥有。

## 建议的验证与迁移顺序

### 第一步 保护现有数据并定义目标

优先处理审查中的插件升级数据丢失、更新路径校验和宿主越权问题。确定首批必须保留的现有插件和 SDK 接口，列出 macOS 与 Windows 的产品行为要求。工期宽松适合认真验证，但不代表同时更换 UI 库、数据库和所有业务逻辑能带来收益。

### 第二步 用相同功能验证 Electron 与 Wails v3

两个原型采用同一份 React 插件视图、相同的 SDK 契约和上下文数据。先实现启动器、一个有输入和存储的 UI 插件、一个后台执行命令，并允许同一插件同时开两个实例。暂不建设完整市场或云服务。

| 验证项 | 验收内容 |
| --- | --- |
| 独立窗口 | 主启动器隐藏后插件保留；一个插件关窗不影响另一个；重复开启/关闭能清理资源 |
| 展开与状态 | 输入、编辑结果、滚动及任务状态按产品要求保留，两个实例的上下文不串用 |
| 焦点 | 快捷键呼出、中文输入法、Escape、文件对话框、切回原应用均符合预期 |
| 桌面行为 | 全屏应用、Spaces、多显示器、缩放、置顶、窗口位置恢复分别验证 |
| 故障与隔离 | 慢 matcher、无限循环、异常插件不能拖死启动器；伪造插件身份不能访问其他 storage |
| 数据一致性 | 多窗口存储写入与使用次数不会互相覆盖；升级失败能保留旧版和用户数据 |
| agent 闭环 | agent 能启动应用、定位多个窗口、修改一个 SDK 调用并自动验证结果 |
| 分发 | 本机打包启动通过，并在 macOS/Windows 目标环境验证签名、权限与一次更新流程 |

性能比较使用发布构建、相同插件、相同硬件和输入。测量冷启动、热键呼出到可输入、插件首开/再次打开、输入匹配 p50/p95、空闲 CPU、进程树总内存、安装体积及多窗开关后的残留资源。系统 WebView 的子进程也要计入，不能只比较宿主 PID。

不要事先宣称某方案必然省几倍内存。先记录当前版本基线，再按常驻工具的产品目标设可接受预算。窗口焦点、权限隔离、数据保全是必要条件，不用性能分数抵消功能失败。

### 第三步 根据结果确定路线

- Electron 在焦点与资源预算内通过验证，且 TS 接口与自动化明显改善实际开发闭环：采用 Electron + TypeScript，逐步迁移宿主服务。
- Wails v3 达到同样的窗口和隔离要求，同时较好保留当前 Go 能力：采用 Wails v3，重构插件运行时，固定 Beta 版本并建立回归。
- 两者均未达到轻量目标，或 Rust 能力层成为明确需求：再把 Tauri 加入同一组验证，避免凭主观印象定输赢。

下一轮最值得先完成的里程碑是：同一插件的两个独立实例可以安全运行，隐藏启动器不影响它们，插件升级不丢数据，agent 可以自动验证完整链路。这个结果比把现有 Go 文件全部翻译成另一门语言更能证明新方向的价值。
