# 实现机制说明

[返回文档索引](./README.md) · [返回仓库速览](../AGENT.md)

## 概览

这份文档描述 WaTools 当前实现中的关键机制，包括 Wails 绑定、命令系统、插件系统、统计排序、更新与资源分发。

## Wails 绑定机制

当前前后端桥接以 [`internal/coordinator/coordinator.go`](../internal/coordinator/coordinator.go) 为单一出口。

实际含义：

- `main.go` 只绑定 `WaAppCoordinator`
- 前端不会直接绑定 `internal/app`、`internal/plugin` 或其他业务包
- 新的前端可调用能力，必须先在业务模块实现，再由 coordinator 暴露

典型扩展步骤：

1. 在对应后端模块实现能力
2. 在 coordinator 增加对外方法
3. 必要时补前端 `api/` 封装
4. 更新前端 store、schema 或组件

## 命令系统机制

命令系统的核心在 [`internal/command/`](../internal/command/)。

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

插件机制集中在 [`internal/plugin/`](../internal/plugin/)。

### 安装与状态

- 插件安装来源是 `.wt` 包
- 安装后复制到本地缓存目录
- 启用状态、使用次数和存储内容会写入 SQLite
- 前端通过 `GetPluginsApi` 获得完整插件元数据

### 两类插件入口

- `executable`
  作为命令面板中的动作直接执行
- `ui`
  作为 iframe 页面打开

### UI 插件上下文注入

UI 插件页由 [`frontend/src/components/watools/wa-plugin.tsx`](../frontend/src/components/watools/wa-plugin.tsx) 承接。

宿主会向 iframe 注入：

- `runtime`
- `watools`
- `pluginContext`
- `watools:context-ready` 事件

这意味着插件正确读取上下文的方式应优先依赖 `window.pluginContext` 与 `watools:context-ready`，而不是依赖兼容字段。

### executable 插件执行

可执行插件在触发时会临时把 `window.watools` 替换为带 package scope 的 API 实例，执行后再恢复。

这保证：

- 插件可以访问宿主能力
- 插件存储可以按 `packageId` 做隔离

## 统计与排序机制

当前使用统计与排序不是单点逻辑，而是前后端协作完成。

### 使用统计

- 前端分别缓冲应用命令与插件的使用更新
- 页面隐藏、卸载或触发动作后批量提交
- 后端负责持久化到 SQLite

### 排序

- 前端会把应用、操作命令、插件和内置功能项合并
- 使用 ranking history 和输入上下文进行排序
- 多来源结果会先去重，再统一排序

这也是主命令面板体验的核心机制之一。

## 更新机制

更新能力由 [`internal/update/`](../internal/update/) 与 coordinator 暴露接口组成。

当前流程：

1. 前端请求检查更新
2. 后端返回版本信息
3. 前端触发下载
4. 后端执行安装
5. 若安装结果要求退出，则通过 Wails runtime 退出应用

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

- [`pkg/db/`](../pkg/db/)
- [`pkg/logger/`](../pkg/logger/)
- [`internal/app/`](../internal/app/)
