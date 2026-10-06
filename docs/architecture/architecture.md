# 架构说明

[返回文档索引](../README.md) · [返回仓库速览](../../AGENT.md)

## 概览

WaTools 是一个以 Wails 为壳的桌面应用，核心分为 5 层：

1. Wails 桌面容器
2. Go 后端能力层
3. React 前端交互层
4. SQLite 与本地数据持久化层
5. 插件运行与资源分发层

整体上不是传统 Web 应用，而是“本地桌面容器 + 本地系统能力 + 前端命令面板”的组合。

## 系统分层

### 1. 宿主层

- 入口在 [`main.go`](../../main.go)
- Wails 负责创建透明无边框窗口、生命周期钩子、菜单和前后端绑定
- `frontend/dist` 会在构建后嵌入到 Go 二进制中
- `/api/*` 请求由自定义 handler 接管，而不是完全交给前端静态资源服务器

### 2. 后端能力层

后端主要在 [`internal/`](../../internal) 下拆分职责：

- `app/`
  窗口显隐、热键、剪贴板、平台差异行为
- `command/`
  应用命令扫描、系统操作命令、目录监听与触发执行
- `plugin/`
  插件安装、状态管理、包内存储与运行时元数据
- `api/`
  提供给前端和插件的通用能力封装
- `update/`
  更新检查、下载与安装
- `handler/`
  图标和插件资源的 HTTP 分发

### 3. 前端交互层

前端代码集中在 [`frontend/src/`](../../frontend/src)：

- `components/watools/`
  主界面与主要业务页面
- `stores/`
  Zustand 状态管理
- `api/`
  Wails 绑定调用封装
- `lib/`
  搜索、排序、插件上下文、运行环境等工具机制
- `schemas/`
  业务类型定义

前端主入口是 [`frontend/src/app.tsx`](../../frontend/src/app.tsx)，页面壳由 [`frontend/src/components/watools/watools.tsx`](../../frontend/src/components/watools/watools.tsx) 承接。

### 4. 数据持久化层

- SQLite 访问层在 [`pkg/db/`](../../pkg/db)
- SQL 源文件在 `pkg/db/queries/`
- 迁移脚本在 `pkg/db/migrations/`
- 应用命令、插件状态、插件存储和使用统计都依赖本地数据库

### 5. 插件运行层

- 官方插件源码位于 [`plugins/official/`](../../plugins/official)
- `.wt` 插件包安装后写入持久数据目录中的独立 revision，缓存目录保留日志、图标与更新包
- UI 插件通过 iframe 承载
- executable 插件直接通过注入 API 的方式执行

## 核心调用链

典型运行链路如下：

1. `main.go` 初始化配置与日志，并启动 Wails
2. `internal/host/` 七个领域服务按职责启动模块；command、plugin、api、update 实例在入口显式构造
3. 前端通过 `api/host` 稳定契约调用真实 Wails 适配器或独立内存 mock
4. 对应服务验证 typed DTO，再调用业务模块；存储与插件资源路由共享同一状态实例
5. 前端 `stores` 和 `components` 根据返回结果渲染命令项、插件页与管理页

接口扩展从对应领域服务收口，同时更新自动生成绑定和两个 Host API 实现。只有 Wails 适配器依赖生成路径；插件继续使用按 packageId 创建的 SDK。具体服务、错误与保留的进程级资源见 [实现机制](./implementation-mechanism.md)。

## 关键数据流

### 命令数据流

- 启动时从 SQLite 读取应用命令
- 若数据库为空，则从磁盘扫描应用目录
- 文件监听器负责在目录变化后重新同步
- 前端读取应用命令和系统操作命令，统一进入命令面板排序与展示

### 插件数据流

- 插件由 `.wt` 包安装
- 插件元数据与启用状态从数据库加载
- UI 插件入口通过 `/api/plugin/...` 暴露资源
- executable 插件通过运行时注入 `window.watools` 获得宿主能力

### 使用统计流

- 前端按应用命令和插件分别缓冲使用记录
- 页面卸载、切换或命令触发后批量回写
- 排序历史与使用次数一起影响最终推荐结果

## 平台现状

- 当前仓库同时保留 macOS 与 Windows 的平台分支文件
- 从实现密度看，macOS 侧的应用扫描、热键和系统操作更完整
- 文档与功能判断应以当前实现而不是 README 宣传语为准

## 本机源码安装与管理中心（2026-10-07）

`/management` 合并本体更新和插件管理，旧管理路由保留为对应标签入口。新增本机能力归入既有 UpdateService，Host API 提供配置、任务状态、启动和取消；未向插件 SDK 增加安装权限。`internal/localinstall` 管理源码配置和异步子任务，界面离开后任务继续，宿主退出时等待取消或提交完成。任务输出限制为 120 行，每行 2000 字节。

CLI 和 GUI 共用 `go run ./cmd/watoolsctl install`：隔离构建全部产物，然后 `cmd/localinstall` 校验并暂存应用、保留备份、切换本体，复用插件安装器批量提交官方插件和安装收据。失败恢复旧 bundle 和插件指针，不重建已有用户记录；运行中小窗继续读取旧 revision。源码、数据及目标应用分别有安装锁。GUI 使用收据中的绝对 Go 路径和工具 PATH，先编译当前源码的轻量 CLI，再直接运行，避免依赖 Finder 的环境及 `go run` 外层退出行为。详细命令、取消和恢复边界见[本地指南](../development/local-workflow.md)。
