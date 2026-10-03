# 日志诊断与 Agent 工具

[返回 Backlog](./README.md)

状态：planned。优先级：P1。日志约定可先做；窗口诊断和 MCP 依赖 [Wails v3](./wails-v3-migration.md) 与 [插件会话](./plugin-windows-and-sessions.md)。目标是操作调用可追踪，agent 可在开发环境检查真实应用。

## 现状问题

`pkg/logger/` 已有轮转和查询，前端有日志入口，但请求、插件、会话和窗口仍需统一关联，原生窗口与插件运行缺少集中诊断视图。

## 实施范围

- 统一 `requestId`、`pluginId`、`sessionId`、`windowId`、耗时和错误码，串联命令、窗口、资源及宿主调用。
- 诊断页展示窗口、会话、revision、加载状态、事件和错误，支持按关联 ID 定位。
- 默认不记录凭据、完整剪贴板和敏感请求正文；导出材料脱敏。
- 性能问题按需采集 Go trace 或 profile，限定时间与目录，先采用现有工具。
- 评估固定版本的 v3 应用与 CLI MCP，配置开发访问和稳定 UI 定位标识，供 agent 复现。
- MCP 仅用于开发构建，发布流程排除相关 tag，检查环境变量不会误开启发布包能力。

## 验收与验证

- 一次插件操作可从 UI 追到 Go 调用及结果，错误有位置与关联信息。
- 可定位实例窗口、revision、挂起请求与清理状态。
- 导出不含真实凭据或默认输入正文。
- MCP 能执行选定窗口场景，记录可覆盖与不能覆盖范围。
- 发布包无 MCP 服务；DOM 快照不是像素截图，不据此宣称焦点、输入法和权限已验收。

## 官方依据

- [Wails MCP](https://v3.wails.io/guides/mcp-service/)：实验能力，实施时重新核对 API。
- [调试与 Go trace](https://v3.wails.io/guides/dev/debugging/)

## 实施记录

尚未开始。未接入 MCP，现有日志不代表完整调用链。
