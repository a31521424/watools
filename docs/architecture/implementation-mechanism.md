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

前端 [`api/host/contracts.ts`](../../frontend/src/api/host/contracts.ts) 是稳定业务契约，只有 [`api/host/wails.ts`](../../frontend/src/api/host/wails.ts) 接触生成路径。该适配器转换窗口/命令/日志类型与 Wails 错误；mock 同样实现 HostAPI，使用独立内存固定场景。旧插件 SDK 名称、packageId 闭包、Storage 值与 HTTP `status_code` 保持兼容；旧 `internal/coordinator` 生成路径及方法 ID 已删除，直接依赖内部绑定的调用方必须迁移，内部路径不属于插件 SDK。

扩展能力时：在业务模块实现 → 在对应领域服务校验/暴露 → 运行 `node scripts/dev.mjs bindings` → 更新 HostAPI 与两个适配器 → 迁移调用方并补关键契约回归。禁止手改生成文件。

当前固定 Wails v3 Beta.27 与同版 npm runtime。启动前配置 launcher 窗口，WindowRuntimeReady 后定位并注册快捷键；命令 watcher 通过注入回调发出 `watools.applicationChanged`。

主窗口是透明无边框 WebviewWindow，macOS 位置与焦点桥接接收该窗口的 NativeWindow 句柄，不再查找任意 key/main window。原生菜单改为 v3 menu roles；插件安装文件对话框明确附着主窗口。

macOS 全局快捷键使用 Carbon 独占注册，生命周期入口先调度到 AppKit 主线程再取得锁，注册与移除在主线程完成，避免退出清理的锁反转。占用冲突会返回错误，现有配置更新流程恢复旧配置与注册；注册的事件处理器随快捷键释放。Windows 仍使用原有 hotkey 库。启动器显式呼出使用 `activateIgnoringOtherApps:YES`，避免 macOS 14+ 的 best-effort `activate` 留下可见但没有键盘焦点的窗口；隐藏时仍按原有逻辑交还先前应用。

插件继续注入 `window.runtime` 兼容对象，由 `frontend/src/lib/legacy-runtime.ts` 将旧名称转换为 v3 API，包含旧剪贴板成功布尔值、窗口尺寸 `{w,h}` 与事件可变参数。常用日志走显式宿主方法；v3 framework 自动调用日志过滤 args/result。这个兼容对象不是完整 v2 运行时或不可信插件沙箱，具体支持边界见插件指南。

## 命令系统机制

命令系统的核心在 [`internal/command/`](../../internal/command)。

它分成两类：

- 应用命令
- 系统操作命令

### 应用命令

主要机制：

- 启动时优先从 SQLite 读取
- 数据为空时从磁盘扫描应用目录
- 目录变更通过 watcher 监听
- 有新增、更新或删除时发出 `watools.applicationChanged`

当前实现特征：

- 应用扫描与图标处理包含平台分支
- 命令触发依赖内存中的 runner 列表
- SQLite 既是缓存也是使用统计存储

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

未设置 `WATOOLS_DATA_DIR` 时，正式安装使用系统应用数据目录，并按既有流程迁移旧缓存；显式设置此变量时视为独立安装，不导入旧缓存。开发入口 `scripts/dev.mjs` 同时隔离数据和缓存，具体目录与临时检查行为见 [本地开发指南](../development/local-workflow.md)。
