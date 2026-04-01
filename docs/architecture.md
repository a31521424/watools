# 架构说明

[返回文档索引](./README.md) · [返回仓库速览](../AGENT.md)

## 概览

WaTools 是一个以 Wails 为壳的桌面应用，核心分为 5 层：

1. Wails 桌面容器
2. Go 后端能力层
3. React 前端交互层
4. SQLite 与本地缓存持久化层
5. 插件运行与资源分发层

整体上不是传统 Web 应用，而是“本地桌面容器 + 本地系统能力 + 前端命令面板”的组合。

## 系统分层

### 1. 宿主层

- 入口在 [`main.go`](../main.go)
- Wails 负责创建透明无边框窗口、生命周期钩子、菜单和前后端绑定
- `frontend/dist` 会在构建后嵌入到 Go 二进制中
- `/api/*` 请求由自定义 handler 接管，而不是完全交给前端静态资源服务器

### 2. 后端能力层

后端主要在 [`internal/`](../internal/) 下拆分职责：

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

前端代码集中在 [`frontend/src/`](../frontend/src/)：

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

前端主入口是 [`frontend/src/app.tsx`](../frontend/src/app.tsx)，页面壳由 [`frontend/src/components/watools/watools.tsx`](../frontend/src/components/watools/watools.tsx) 承接。

### 4. 数据持久化层

- SQLite 访问层在 [`pkg/db/`](../pkg/db/)
- SQL 源文件在 `pkg/db/queries/`
- 迁移脚本在 `pkg/db/migrations/`
- 应用命令、插件状态、插件存储和使用统计都依赖本地数据库

### 5. 插件运行层

- 官方插件源码位于 [`plugins/official/`](../plugins/official/)
- `.wt` 插件包安装后会被复制到本地缓存目录
- UI 插件通过 iframe 承载
- executable 插件直接通过注入 API 的方式执行

## 核心调用链

典型运行链路如下：

1. `main.go` 初始化配置与日志，并启动 Wails
2. `WaAppCoordinator` 在 `Startup` 时初始化 app、command、plugin 等模块
3. 前端通过 `frontend/wailsjs` 调用 coordinator 暴露的方法
4. coordinator 将请求分发到具体子模块
5. 前端 `stores` 和 `components` 根据返回结果渲染命令项、插件页与管理页

这意味着 coordinator 是稳定边界：

- 前端与后端之间的接口变更应优先从这里收口
- 新能力应先落到业务模块，再由 coordinator 暴露

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
