# AGENT.md

WaTools 仓库速览。这个文件只负责帮助开发者和代理快速建立上下文，不承载所有细节说明。

## 项目摘要

WaTools 是一个基于 Wails 的桌面效率工具，当前代码形态可以概括为：

- Go 后端负责窗口生命周期、系统能力、命令扫描、插件安装与本地数据持久化
- React + TypeScript 前端负责命令面板、插件页、管理页与交互状态
- 主界面是透明无边框的 command palette 风格桌面浮层
- 插件同时支持 `executable` 与 `ui` 两类入口
- 当前真实能力以 macOS 侧最完整，仓库也保留了 Windows 分支实现

当前仓库内可确认的核心版本：

- Go: `1.26`
- Wails: `v2.10.2`
- React: `19`
- Vite: `7`
- Tailwind CSS: `4`

## 启动与调用链

主程序入口是 [`main.go`](./main.go)。

运行链路可以先按下面顺序理解：

1. `main.go` 嵌入 `frontend/dist`，初始化配置、日志与 Wails 应用
2. 仅绑定 [`internal/coordinator`](./internal/coordinator/) 的 `WaAppCoordinator`
3. 前端通过 `frontend/wailsjs` 生成绑定调用 coordinator
4. coordinator 再分发到 `internal/app`、`internal/command`、`internal/plugin`、`internal/api`、`internal/update`
5. 前端由 `frontend/src/components/watools/` 下的命令面板、插件页和管理页承接最终交互

需要记住的一点：

- 前端不会直接绑定 `internal/app` 或 `internal/plugin`
- 任何新的 Wails 可调用方法，都应先进入 `WaAppCoordinator`

## 仓库目录地图

### 应用入口与配置

- [`main.go`](./main.go)
  Wails 入口、资源嵌入、窗口选项、菜单与生命周期绑定
- [`config/`](./config/)
  项目元数据、缓存目录、运行环境相关配置
- [`wails.json`](./wails.json)
  Wails 构建与前端命令配置

### 后端核心

- [`internal/coordinator/`](./internal/coordinator/)
  前后端桥接层，也是当前唯一的 Wails 绑定出口
- [`internal/app/`](./internal/app/)
  窗口显示隐藏、热键、剪贴板、平台相关应用行为
- [`internal/command/`](./internal/command/)
  应用扫描、操作命令、文件监听与命令触发
- [`internal/plugin/`](./internal/plugin/)
  插件安装、卸载、启停、状态与存储
- [`internal/api/`](./internal/api/)
  对前端和插件暴露的通用能力，例如打开目录、保存图片、代理请求
- [`internal/handler/`](./internal/handler/)
  `/api/*` 自定义资源路由、图标与插件资源分发
- [`internal/update/`](./internal/update/)
  更新检查、下载和安装
- [`internal/app_menu/`](./internal/app_menu/)
  原生菜单定义

### 前端核心

- [`frontend/src/app.tsx`](./frontend/src/app.tsx)
  前端入口组件
- [`frontend/src/components/watools/`](./frontend/src/components/watools/)
  主界面、命令面板、插件页、日志页、更新页、插件管理页
- [`frontend/src/stores/`](./frontend/src/stores/)
  Zustand 状态层，负责输入状态、插件状态、应用命令状态、排序历史
- [`frontend/src/api/`](./frontend/src/api/)
  对 Wails 绑定的轻量封装
- [`frontend/src/lib/`](./frontend/src/lib/)
  搜索、排序、插件上下文、环境判断等前端机制代码
- [`frontend/src/schemas/`](./frontend/src/schemas/)
  前端使用的类型定义
- [`frontend/wailsjs/`](./frontend/wailsjs/)
  Wails 生成代码，通常不手改

### 数据、工具与插件

- [`pkg/db/`](./pkg/db/)
  SQLite 访问层、sqlc 生成代码、迁移与查询定义
- [`pkg/logger/`](./pkg/logger/)
  日志适配、查询与目录管理
- [`pkg/models/`](./pkg/models/)
  命令、插件等共享模型
- [`cmd/pluginctl/`](./cmd/pluginctl/)
  官方插件的打包、安装、列出工具
- [`plugins/`](./plugins/)
  官方插件源码与打包输出目录
- [`docs/`](./docs/)
  项目文档索引与专题文档

## 快速阅读路径

按任务选入口，不要默认整仓通读。

- 想看整体分层与运行链路：[`docs/architecture.md`](./docs/architecture.md)
- 想看界面组织与视觉基调：[`docs/ui-style.md`](./docs/ui-style.md)
- 想看命令、插件、更新等机制：[`docs/implementation-mechanism.md`](./docs/implementation-mechanism.md)
- 想看用户可感知功能模块：[`docs/feature-modules.md`](./docs/feature-modules.md)
- 想看插件开发规范：[`docs/PLUGIN_DEVELOPMENT_INDEX.md`](./docs/PLUGIN_DEVELOPMENT_INDEX.md)
- 想看文档总索引：[`docs/README.md`](./docs/README.md)

## 文档关系

- `AGENT.md` 负责仓库地图、快速摘要和阅读导航
- [`docs/README.md`](./docs/README.md) 负责完整文档索引
- `docs/plugin-development/` 保留为插件开发专题，不并入本文件
