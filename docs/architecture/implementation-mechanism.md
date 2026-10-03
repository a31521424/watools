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
