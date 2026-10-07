# 实现机制说明

[返回文档索引](../README.md) · [返回仓库速览](../../AGENT.md)

## 概览

这份文档描述 WaTools 当前实现中的关键机制，包括 Wails 绑定、命令系统、插件系统、统计排序、更新与资源分发。

## Wails 绑定机制

当前桥接由 [`internal/host/`](../../internal/host/) 中七个领域服务承担：

| 服务 | 职责 |
| --- | --- |
| AppService | 呼出/隐藏、剪贴板读取、快捷键状态、launcher 尺寸与环境初始化 |
| CommandService | 应用扫描、命令触发、使用统计与 watcher 生命周期 |
| PluginService | 插件列表、入口、安装/停用/卸载与统计；注入文件选择回调 |
| StorageService | packageId 范围内的 JSON 存储与凭据；与资源路由共享插件实例 |
| SystemService | 文件夹、图片与可取消 HTTP 请求；关闭时取消请求 |
| UpdateService | 检查/下载/安装更新；注入退出回调 |
| LogService | 日志目录、查询与前端日志 |

`main.go` 显式构造 command、plugin、api、update 实例，注入菜单、资源状态检查、事件、文件对话框和退出边界。Wails 按声明顺序启动服务，按逆序关闭；AppService 首先初始化环境。App 与 HotkeyManager 仍共享原有 native 单例，SQLite、目录配置和 secrets 仍是进程级资源，本轮没有声称消除所有全局状态。launcher 尺寸使用小型 `WindowGeometry` 接口，可在无事件循环的测试中替换。

请求使用具名 DTO：统计时间/次数、HTTP、日志查询、存储的 packageId/key 均由生成绑定提供类型。统计整批校验后才写入；缺失次数、非整数、无效时间、`enabled: null` 和凭据 `value: null` 返回 `INVALID_ARGUMENT`。存储值保留动态 JSON，允许显式 null，缺失 value 被拒绝；读取返回 RawMessage，避免 Wails 把 nil 响应与空字符串混淆。统一错误提供 code/message/operation，Go `Unwrap` 保留底层原因，展示文本与传输外层错误不包含原始凭据或请求内容。

本机安装使用局部 ValidationError/ErrBusy：可修正输入映射 `INVALID_ARGUMENT`，执行/提交或进程锁争用映射 `UNAVAILABLE`，I/O/启动/停止故障走安全 `OPERATION_FAILED`，取消/超时维持统一分类。转换保留 cause，快照过期映射也保留 sentinel。锁实现仅将实际锁争用标记为 Busy，其余系统故障保持原因。

前端 [`api/host/contracts.ts`](../../frontend/src/api/host/contracts.ts) 是稳定业务契约，只有 [`api/host/wails.ts`](../../frontend/src/api/host/wails.ts) 接触生成路径。该适配器转换窗口/命令/日志类型与 Wails 错误；mock 同样实现 HostAPI，使用独立内存固定场景。旧插件 SDK 名称、packageId 闭包、Storage 值与 HTTP `status_code` 保持兼容；旧 `internal/coordinator` 生成路径及方法 ID 已删除，直接依赖内部绑定的调用方必须迁移，内部路径不属于插件 SDK。

扩展能力时：在业务模块实现 → 在对应领域服务校验/暴露 → 运行 `go run ./cmd/watoolsctl bindings` → 更新 HostAPI 与两个适配器 → 迁移调用方并补关键契约回归。禁止手改生成文件。

当前固定 Wails v3 Beta.27 与同版 npm runtime。启动前配置 launcher 窗口，WindowRuntimeReady 后定位并注册快捷键；命令 watcher 通过注入回调发出 `watools.applicationChanged`。

主窗口是透明无边框 WebviewWindow，macOS 使用 non-activating NSPanel；位置桥接接收启动器的 NativeWindow 句柄，不查找任意 key/main window。原生菜单改为 v3 menu roles；插件安装文件对话框明确附着主窗口。

macOS 全局快捷键使用 Carbon 独占注册，生命周期入口先调度到 AppKit 主线程再取得锁，注册与移除在主线程完成，避免退出清理的锁反转。占用冲突会返回错误，现有配置更新流程恢复旧配置与注册；注册的事件处理器随快捷键释放。Windows 仍使用原有 hotkey 库。启动器呼出仅 Show／Focus 自己的 non-activating NSPanel，获得键盘焦点而不执行应用级激活，因此不会将其他独立插件窗一起置前。隐藏只 order out 启动器，由 AppKit 恢复键盘焦点，不强制激活先前应用，以免打断刚分离的小窗。

插件继续注入 `window.runtime` 兼容对象，由 `frontend/src/lib/legacy-runtime.ts` 将旧名称转换为 v3 API，包含旧剪贴板成功布尔值、窗口尺寸 `{w,h}` 与事件可变参数。常用日志走显式宿主方法；v3 framework 自动调用日志过滤 args/result。这个兼容对象不是完整 v2 运行时或不可信插件沙箱，具体支持边界见插件指南。

## 命令系统机制

命令系统的核心在 [`internal/command/`](../../internal/command)。

它分成两类：

- 应用命令
- 系统操作命令

### 应用命令

主要机制：

- 启动时优先从 SQLite 读取
- 查询成功且数据为空时从磁盘扫描应用目录；读取失败直接返回错误
- 目录变更通过 watcher 监听
- 有新增、更新或删除时发出 `watools.applicationChanged`

当前实现特征：

- 应用扫描与图标处理包含平台分支
- 命令触发依赖内存中的 runner 列表
- SQLite 既是缓存也是使用统计存储

DB 应用查询显式返回 error，单条记录不存在返回 `nil, nil`；读取失败不会触发扫描、写入、删除或通知。首次扫描或入库失败向 command/Host 返回，Host 转换为安全错误并拒绝 Wails Promise，成功响应仍是数组。命令读取、初始化与 runner 发布共享串行锁，失败保留旧 runner，避免较早查询晚发布覆盖较新缓存。后台刷新遇到 DB 故障停止后续阶段；通知仅反映已成功提交的批次，不把整轮部分提交解释为原子成功。

前端应用 store 保留失败前的列表和搜索索引，显示安全提示并提供重试；刷新期间旧命令仍可搜索。显式并发读取共享同一结果，查询期间到来的变更事件合并为后续读取，避免丢失更新。初始加载和事件刷新消费拒绝，查询发布时保留请求期间新增的本地使用统计及未保存 buffer；没有并发本地更新时采用新的 Host 统计。

watcher 错误处理与重试已收敛为 darwin/windows 共用实现；未知解析/I/O 故障不再按次数盲目重试，取消/超时立即结束。默认仅当整棵错误树都由 SQLite BUSY 构成时重试完整操作，包含未知回滚失败则不重试；等待可取消，最终失败报告实际次数，服务退出取消不增加失败指标。平台扫描器的部分成功、暂时不可读与删除规则、跨扫描/watch 的协调仍按 [实施计划](../backlog/architecture-simplification.md) 后续 E05/E09 推进。

### 系统操作命令

- 操作命令由 `internal/command/operator/` 提供
- 当前实现明显偏向 macOS 侧系统能力
- 这些命令与应用命令一起进入前端命令面板，但来源不同

## 插件系统机制

### 剪贴板原文与呼出性能

自动呼出走 AppService 的 `GetAutomaticClipboardApi`。macOS 在一次 AppKit 调度中检查剪贴板版本和主内容类型，只读取文本，不转换图片或读取文件列表；相同版本复用宿主内存快照。读取过程中版本变化会重试一次，避免返回混合版本的数据。Windows 保留对应接口，使用一次 STA PowerShell 调用与系统 sequence number，明确采用 UTF-8 输出；server 模式返回空素材。读取通过独立锁串行化，不持缓存锁等待原生主线程；退出可关闭缓存，晚到结果不再回写，避免退出与读取互相等待。

文本长度以 UTF-16 code unit 计。2000 以内的原文直接返回；更长的文本只返回 snapshotId、版本、长度、首尾摘要和独立的首尾各最多 128 单元匹配片段，完整 origin 留在 Go 内存。选择插件或“复制原文”时才通过 `GetClipboardSnapshotTextApi` 读取固定快照，不重新读取当前剪贴板。空白、换行、Unicode 和 NUL 均保留。手动文本粘贴直接采用事件中的原文；图片、文件等素材继续走完整读取接口。

文本写入统一经过 `SetClipboardTextApi`，旧插件的 `runtime.ClipboardSetText` 也使用此边界并保持成功返回 true 的兼容语义。macOS 以明确 UTF-8 字节长度同步写入 NSPasteboard，避免框架默认 C 字符串在 NUL 处截断；server 模式明确报不支持。浏览器 mock 只修改自己的合成剪贴板状态，不触碰系统剪贴板。旧 `runtime.ClipboardGetText` 仍是框架兼容接口，原文完整性保证使用上述快照读取边界。

前端分别处理 show、origin 和 query：show 是有界展示，origin 是原文或宿主快照引用，query 超过 128 单元时不进入通用 Fuse 搜索和全文排名规范化。超过 2000 单元的素材以单独摘要展示，输入新查询替换当前素材，“复制原文”使用完整 origin。摘要不会作为可编辑正文或搜索关键词。

启动器的唯一素材状态是 [`LauncherInput`](../../frontend/src/schemas/launcher.ts)：`inline` 持有完整 `PluginContext`，`clipboard-snapshot` 只持有 id、长度和首尾／展示元数据，没有空字符串正文占位。store 只保存素材、revision 和最近接受的剪贴板版本；展示、query、排名输入及轻量匹配在 [`launcher-input.ts`](../../frontend/src/lib/launcher-input.ts) 从同一个素材派生。首尾采样保持 Unicode surrogate pair 完整。`resolvePluginContext(input)` 是唯一原文解析边界，真实选择或复制才调用；内联内容复制可变外壳和文件数组，快照只读取捕获的 id，失败不改读当前剪贴板。

宿主最多保留四个快照，正文按 64 MiB 总量预算回收旧记录；单个超过预算的当前原文仍完整保留。未续租记录两分钟后过期，定时释放缓存。前端根组件随输入状态每分钟续租，清空／替换素材时释放，路由切换不提前销毁待打开插件的源快照；关闭宿主会停止定时器并清空缓存。过期或淘汰明确报错，不以新剪贴板内容替代旧 origin。未被接受的并发 focus 结果由有界缓存自动过期，不释放其他 focus 可能正在接受的共享结果。

匹配与执行使用独立类型：`matchPreview(PluginMatchContext)` 优先用于所有输入长度，只收到冻结的首尾各 128 单元、长度、来源与素材类型／文件数量，不含完整正文、图片数据、路径或宿主 API。`execute` 与 UI 收到完整 `PluginContext.input.value` 和文本 `clipboard.text`，只有真实选择时才取延迟原文。六个官方插件均接入新匹配接口；JSON 用首尾括号／命令前缀筛选候选，不解析全文；计算器只检查有界表达式特征，不在匹配时求值。无效计算候选在实际执行时抛出错误，宿主显示错误并保留输入，不按成功清空／隐藏。

插件模块导出 `PluginEntryDefinition`，加载边界将其归一成内部 UI／executable 判别联合 `PluginEntry`：只有一个 `matcher`，UI 必须有 file，executable 必须有 execute。[`filterPluginEntries(entries, input)`](../../frontend/src/lib/plugin-matching.ts) 每次匹配只创建一份冻结的轻量上下文，不再传递完整 context、长度、前缀和 preview 等平行参数。

旧入口兼容集中在 [`legacy-plugin-matcher.ts`](../../frontend/src/lib/legacy-plugin-matcher.ts)：短内联原文调用完整旧 matcher，每个调用都有独立可变外壳／文件数组；长文本和快照不调用旧函数，旧 UI 显式展示、旧 executable 用 `largeTextPrefixes`。新 matcher 的 false 不回退。六个官方入口现在只声明 `matchPreview`，重打包的新版插件需要当前宿主；外部旧插件仍可加载。片段边缘空白太长时可能漏推荐，不冒充全文校验。入口缓存与使用统计分离，统计更新只影响排序。可信 matcher 仍可能故意阻塞；完整声明式 SDK 和 Worker 改造仍属于独立 backlog。

触发时固定素材与 revision／focus generation，原文解析完成后和串行 executable 真正开始时均检查取消。执行、统计 flush、失焦或空输入 Escape 的延迟操作均不能清空／隐藏新输入；旧异步失败也不会在新输入上展示错误。iframe ready 仅按原 revision 清空；ready 前取消仍保留启动器源素材。缺少 launchId 的直接 UI 路由遇到延迟素材时明确提示从启动器选择，不交付假的空正文。

本轮验证与边界见 [2026-10-06 证据](../reviews/evidence/2026-10-06/README.md)。

插件机制集中在 [`internal/plugin/`](../../internal/plugin)。

### 默认展开与 Command+D 分离

UI 默认仍在主窗口展开。`wa-plugin.tsx` 在主页面和 iframe 捕获 Command+D，原页面保持可用直到新窗加载及恢复完成；失败展示错误，Esc 可取消。捕获期间暂时禁止编辑，防止状态导出后继续修改被丢弃。`plugin-window-state.ts` 使用插件 hooks 或普通表单兼容快照；官方 JSON 等待 Monaco 就绪后捕获编辑模型和视图，不传 DOM。

官方 UI 插件 0.2.0 均提供 hooks；UI SDK 在 context-ready 前注入只读 `session.id/windowMode`，独立窗恢复快照而不重复处理原文。JSON 恢复直接设置新模型，并在初始化前标记恢复所有权，阻止迟到上下文覆盖编辑；二维码保留现有图片和模式，翻译暂停未完成请求、保留译文且不重复自动请求。版本升级只切换安装指针，已打开的页面仍使用旧 revision，需保存内容后重新打开。

PluginService 的五个会话 API 负责创建、领取、查询 ready、确认恢复和关闭。`plugin_window.go` 在宿主保存会话，校验启用状态、包内路径和 revision；原生窗仅通过 `/?pluginSession=<id>` 加载独立 shell。隐藏窗恢复状态后显示，主窗口轮询轻量 ready，恢复成功后隐藏自己并返回空启动器；下次唤出仍为最小 64px。创建／显隐／关闭不持会话锁等待原生主线程，关闭清理，pending 30 秒超时，应用退出禁止回写。禁用／卸载先关闭所属窗口，升级保留已打开的旧 revision。

独立 shell 延迟加载插件视图，不初始化启动器命令与 usage store。每个 iframe 初始化自己的 SDK 和窗口 runtime；窗口方法绑定具体窗口，实例事件命名避免串用。HTTP 公共 requestId 在 UI 和宿主分别加实例／窗口前缀，单窗关闭只取消其请求。共享 Storage／Secret 仍以 packageId 为范围。主窗口 ResizeObserver 保持最小 64px，并丢弃旧异步尺寸读数；菜单刷新指向当前活跃窗口。更多契约及剩余范围见 [窗口 backlog](../backlog/plugin-windows-and-sessions.md) 和 [插件运行时](../plugin-development/03-runtime-and-context.md#分离窗口的状态接口)。

### 安装与状态

- 插件安装来源是 `.wt` 包
- 安装后写入持久数据目录中的独立 revision，再切换 current.json；升级保留配置和统计
- 启用状态、使用次数和存储内容会写入 SQLite
- 前端通过 `GetPluginsApi` 获得插件元数据和加载错误，列表不返回 storage 内容

### 两类插件入口

- `executable`
  作为命令面板中的动作直接执行
- `ui`
  作为 iframe 页面打开

### UI 插件上下文注入

UI 插件页由 [`frontend/src/components/watools/wa-plugin.tsx`](../../frontend/src/components/watools/wa-plugin.tsx) 承接。

宿主会向 iframe 注入：

- `runtime`
- `watools`
- `pluginContext`
- `watools:context-ready` 事件

这意味着插件正确读取上下文的方式应优先依赖 `window.pluginContext` 与 `watools:context-ready`，而不是依赖兼容字段。

路由只带 launchId，正文只在内存中交付。启动快照复制对象外壳和文件数组，共享字符串，避免 JSON 编解码；每个 iframe 文档收到独立外壳。快照固定在一次路由会话中，加载、重载和源输入清空均不改写原文。使用统计提交不阻塞打开；ready、失败、取消和路由切换释放相应记录，未领取记录五分钟后由定时器清理，并限制数量与保守估计的正文总量。

### executable 插件执行

可执行插件串行执行，执行期间将 `window.watools` 设为该包的 API，完成后恢复。新插件也可以从 `execute(context, api)` 的第二个参数接收 API。

Storage 使用 packageId 区分数据归属。当前采用可信插件模型，插件仍可访问宿主页面和完整桥接；这个约定不构成对恶意插件的安全隔离。

## 统计与排序机制

当前使用统计与排序不是单点逻辑，而是前后端协作完成。

### 使用统计

- 前端分别缓冲应用命令与插件的使用更新
- 正常触发后立即提交，多个并发更新由快照缓冲串行处理；失败保留并重试
- 后端负责持久化到 SQLite

### 排序

- 前端会把应用、操作命令、插件和内置功能项合并
- 使用 ranking history 和输入上下文进行排序
- 多来源结果会先去重，再统一排序

这也是主命令面板体验的核心机制之一。

## 更新机制

更新能力由 [`internal/update/`](../../internal/update) 与 UpdateService 暴露接口组成。

当前流程：

1. 前端请求检查更新
2. 后端返回版本信息
3. 前端触发下载
4. 后端执行安装
5. 若安装结果要求退出，则通过 v3 App.Quit 退出应用

## HTTP 与资源分发机制

WaTools 不是纯静态前端资源加载模式，`/api/*` 路由由自定义 handler 负责。

当前主要用途：

- 图标读取
- 插件静态资源分发
- 插件入口脚本地址生成

这也是插件 UI 页面能够通过宿主 URL 访问包内资源的基础。

桌面 shell 的已知管理入口（含 `/management`）与插件页面入口回退到 index；未知路径和缺失静态资源仍返回原始错误。日志页通过纯 reducer 与请求 generation 原子更新选择/记录/游标，旧响应、错误和 finally 不影响新查询；分页沿用已完成查询快照，筛选变化或卸载后失效。前端 warn 使用宿主 warning 方法。

## 本地持久化机制

当前本地持久化主要包括：

- 应用命令缓存
- 插件启用状态
- 插件存储 JSON
- 使用次数与最近使用时间
- 热键配置文件
- 日志目录与日志文件

相关代码主要位于：

- [`pkg/db/`](../../pkg/db)
- [`pkg/logger/`](../../pkg/logger)
- [`internal/app/`](../../internal/app)

未设置 `WATOOLS_DATA_DIR` 时，正式安装使用系统应用数据目录，并按既有流程迁移旧缓存；显式设置此变量时视为独立安装，不导入旧缓存。开发入口 `cmd/watoolsctl` 同时隔离数据和缓存，具体目录与临时检查行为见 [本地开发指南](../development/local-workflow.md)。

`pkg/db` 的批处理由 `withTx` 统一完成事务提交，回调只执行 SQL；回调故障的 Rollback 失败会与主错误一起返回，已结束事务的 `sql.ErrTxDone` 不作为额外失败。Commit 或真实 Rollback 失败额外保留 `ErrTransactionOutcomeUnknown`，防止按底层 BUSY 盲目重试结果不明的事务。插入成功日志在 Commit 成功之后产生。插件安装/卸载的主操作与补偿同时失败时使用 `errors.Join`，调用者可分别识别原因；错误链保留不代表补偿失败时数据已自动恢复。

## 本机源码安装与管理中心（2026-10-07）

`/management` 合并本体更新和插件管理，旧管理路由保留为对应标签入口。新增本机能力归入既有 UpdateService，Host API 提供配置、任务状态、启动和取消；未向插件 SDK 增加安装权限。`internal/localinstall` 管理源码配置和异步子任务，界面离开后任务继续，宿主退出时等待取消或提交完成。任务输出限制为 120 行，每行 2000 字节。

CLI 和 GUI 共用 `go run ./cmd/watoolsctl install`：隔离构建全部产物，然后 `cmd/localinstall` 校验并暂存应用、保留备份、切换本体，复用插件安装器批量提交官方插件和安装收据。失败恢复旧 bundle 和插件指针，不重建已有用户记录；运行中小窗继续读取旧 revision。源码、数据及目标应用分别有安装锁。GUI 使用收据中的绝对 Go 路径和工具 PATH，先编译当前源码的轻量 CLI，再直接运行；旧 Node 收据保留 PATH 并重新定位 Go。详细命令、取消和恢复边界见[本地指南](../development/local-workflow.md)。
