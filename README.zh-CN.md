# WaTools

[English README](./README.md)

WaTools 是一个受 uTools 和 Alfred 启发的开源桌面效率工具。它把 Wails 桌面容器、Go 后端和 React 命令面板 UI 组合在一起，用一个全局热键统一承接本地工具、系统操作和插件能力。

## 项目概览

WaTools 当前主要围绕 4 类能力组织：

- 透明、无边框、command palette 风格的桌面主界面
- 本地应用搜索与启动
- 内置系统操作命令
- 支持 `executable` 和 `ui` 两类入口的本地可信插件系统

仓库结构是跨平台的，但当前真实实现仍以 macOS 侧最完整。

## 项目演进

WaTools 早期主要以人工设计和人工编码的方式推进。核心架构、运行链路和初始功能模块都来自传统的软件工程开发过程。

随着项目推进，开发流程逐步引入 AI 协作。后续迭代中开始使用 Codex、Claude Code 以及相关的 vibe coding 工作流来辅助仓库理解、文档重构、实现提速和插件式功能迭代。

因此，这个项目的开发方式可以概括为：

- 前期以人工开发为主
- 后期引入 Codex、Claude Code 等 AI 协作编程方式
- 仍然坚持明确模块边界、可审阅代码路径和可落地文档

## 当前架构

当前项目的核心组成如下：

- `main.go`
  Wails 入口、资源嵌入、生命周期与窗口配置
- `internal/`
  后端模块，负责应用行为、命令扫描、插件、API、更新和资源处理
- `frontend/`
  React 19 + TypeScript + Tailwind 4 前端，负责命令面板、插件页和管理页
- `pkg/`
  共享模型、SQLite 访问层和日志能力
- `plugins/`
  官方插件源码与打包产物目录
- `docs/`
  架构、UI、实现机制、功能模块、发布与插件开发文档

当前唯一的 Wails 绑定出口是 `internal/coordinator/`，它负责把前端请求路由到具体后端模块。

## 核心能力

- 通过全局热键唤起主面板
- 本地应用发现与启动
- 系统操作命令触发
- 从可信 `.wt` 包安装插件
- 通过 iframe 承载 UI 插件，并传递统一 `PluginContext`
- 直接从命令面板执行 executable 插件
- 日志管理与更新管理页面
- 一组官方插件，覆盖常用工具、计算器、JSON、二维码、翻译和文本统计等场景

## 技术栈

- 后端：Go `1.26`
- 桌面框架：Wails `v2`
- 前端：React `19`、TypeScript、Vite `7`
- 样式：Tailwind CSS `4`
- 状态管理：Zustand
- 搜索与命令交互：Fuse.js、cmdk
- 包管理：`pnpm`
- 本地持久化：SQLite

## 文档入口

不要默认整包通读，按任务进入对应文档。

- [`AGENT.md`](./AGENT.md)
  仓库地图、运行摘要和快速上下文入口
- [`docs/README.md`](./docs/README.md)
  文档总索引
- [`docs/architecture.md`](./docs/architecture.md)
  系统分层、边界与数据流
- [`docs/ui-style.md`](./docs/ui-style.md)
  主界面结构、路由与 UI 风格
- [`docs/implementation-mechanism.md`](./docs/implementation-mechanism.md)
  Wails 绑定、命令、插件、更新与资源分发机制
- [`docs/feature-modules.md`](./docs/feature-modules.md)
  面向用户的功能模块拆解
- [`docs/PLUGIN_DEVELOPMENT_INDEX.md`](./docs/PLUGIN_DEVELOPMENT_INDEX.md)
  插件开发阅读入口

## 仓库结构

```text
.
|-- main.go
|-- config/
|-- internal/
|-- frontend/
|-- pkg/
|-- plugins/
|-- docs/
|-- cmd/pluginctl/
`-- build/
```

## 开发启动

### 环境要求

- Go `1.26.1+`
- Node.js `18+`
- `pnpm`
- Wails CLI

### 安装与运行

1. 克隆仓库并进入目录。

```sh
git clone https://github.com/a31521424/watools.git
cd watools
```

2. 安装前端依赖。

```sh
cd frontend
pnpm install
cd ..
```

3. 启动开发模式。

```sh
wails dev
```

4. 构建生产版本。

```sh
wails build
```

构建产物默认输出到 `build/bin/`。

## 官方插件

官方插件源码位于 [`plugins/official`](./plugins/official)。

当前官方插件包括：

- `watools.plugin.common`
- `watools.plugin.calculator`
- `watools.plugin.json`
- `watools.plugin.qr`
- `watools.plugin.translate`
- `watools.plugin.textstats`

常用插件命令：

```sh
go run ./cmd/pluginctl list
go run ./cmd/pluginctl package
go run ./cmd/pluginctl install
```

更多插件说明见：

- [`plugins/README.md`](./plugins/README.md)
- [`docs/PLUGIN_DEVELOPMENT_INDEX.md`](./docs/PLUGIN_DEVELOPMENT_INDEX.md)

## 发布与更新

仓库包含 GitHub Actions 发布流程和应用更新能力。

- 工作流：[`/.github/workflows/release.yml`](./.github/workflows/release.yml)
- 说明：[`docs/release-cd.md`](./docs/release-cd.md)

## 信任模型

当前已安装插件被视为用户主动选择的本地可信代码。WaTools 目前不把自己定位成面向不可信公共市场插件的强隔离沙箱。

## 贡献

欢迎提交 issue 和 pull request。如果改动涉及架构、插件行为或开发者工作流，请同步更新 [`docs/`](./docs/) 下对应文档。

## 许可证

本项目采用 MIT License。
