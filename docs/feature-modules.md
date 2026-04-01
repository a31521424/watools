# 功能模块详解

[返回文档索引](./README.md) · [返回仓库速览](../AGENT.md)

## 概览

这份文档按用户可感知能力梳理模块，而不是按源码目录罗列。

## 应用启动器

- 作用：扫描本机应用并作为命令项展示，支持快速搜索和打开
- 用户入口：主命令面板 `/`
- 主要实现：`internal/command/application`、`internal/command/watcher`、`pkg/db`
- 核心行为：启动时加载缓存，必要时全量扫描，目录变化后增量更新
- 注意事项：当前平台能力明显以 macOS 为主

## 系统操作命令

- 作用：执行锁屏、睡眠、显示桌面、截图等系统级动作
- 用户入口：主命令面板 `/`
- 主要实现：`internal/command/operator`
- 核心行为：把系统操作以命令项形式并入统一搜索结果
- 注意事项：不同平台实现差异较大，不能默认跨平台完全一致

## 主命令面板

- 作用：统一承接输入、匹配、排序与触发
- 用户入口：应用唤起后的默认页面
- 主要实现：`frontend/src/components/watools/wa-command.tsx`
- 核心行为：合并应用、操作命令、插件项和内置项，按上下文和历史排序
- 注意事项：这是整个产品体验的主中枢，改动需要优先保护键盘交互

## 插件管理

- 作用：安装、卸载、启停插件，并展示插件状态
- 用户入口：`/plugin-management`
- 主要实现：`internal/plugin`、`frontend/src/components/watools/wa-plugin-management.tsx`
- 核心行为：从 `.wt` 安装包导入插件，对数据库中的启用状态进行维护
- 注意事项：插件被视为用户主动安装的可信代码，不是强隔离沙箱

## UI 插件容器

- 作用：为带界面的插件提供嵌入式工作区
- 用户入口：`/plugin`
- 主要实现：`frontend/src/components/watools/wa-plugin.tsx`、`internal/handler/plugin.go`
- 核心行为：加载 iframe、注入 `watools` API 与 `pluginContext`
- 注意事项：插件上下文应优先通过 `window.pluginContext` 和 `watools:context-ready` 获取

## 可执行插件

- 作用：让插件像命令一样直接参与搜索和执行
- 用户入口：主命令面板 `/`
- 主要实现：`frontend/src/components/watools/wa-command.tsx`、`frontend/src/api/api.ts`
- 核心行为：临时注入 package 作用域 API，执行插件逻辑并回收上下文
- 注意事项：执行期间的宿主 API 注入是临时态，不应假设全局常驻

## 剪贴板与输入上下文

- 作用：把文本、图片、文件等输入统一整理为插件和命令可消费的上下文
- 用户入口：命令面板输入、粘贴动作、插件触发
- 主要实现：`internal/app`、`frontend/src/stores/appStore.ts`、`frontend/src/lib/plugin-context.ts`
- 核心行为：读取剪贴板、识别内容类型、构造 `PluginContext`
- 注意事项：兼容字段存在，但主路径应使用统一上下文对象

## 使用统计与排序

- 作用：让结果排序更贴合近期行为与输入语义
- 用户入口：主命令面板的结果排序
- 主要实现：`frontend/src/stores/commandRankingStore.ts`、`frontend/src/lib/command-ranking.ts`
- 核心行为：记录选择历史、合并多来源结果、执行排序与去重
- 注意事项：排序结果受输入上下文影响，不只是简单的字符串匹配

## 日志管理

- 作用：查看日志目录、列出日志文件、查询日志内容
- 用户入口：`/log-management`
- 主要实现：`pkg/logger`、coordinator 的日志接口、前端日志管理页
- 核心行为：后端提供日志查询能力，前端负责展示与筛选
- 注意事项：日志是运维和排查入口，接口变更应优先保持向后兼容

## 自动更新

- 作用：检查版本、下载更新包并完成安装
- 用户入口：`/update-management`
- 主要实现：`internal/update`、前端更新管理页
- 核心行为：检查、下载、安装三段式流程
- 注意事项：安装完成后可能触发应用退出，前端应按这个生命周期理解流程

## 官方插件体系

- 作用：维护仓库内置的插件源码、打包与安装流程
- 用户入口：`plugins/official/`、`cmd/pluginctl`
- 主要实现：`cmd/pluginctl/main.go`、`plugins/README.md`
- 核心行为：列出插件、打包 `.wt`、安装到本地 WaTools 缓存
- 注意事项：`plugins/dist/` 是产物目录，不是源码目录
