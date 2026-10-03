# WaTools 优化 Backlog

[返回文档系统](../README.md) · [Agent 约束](../../AGENT.md)

登记日期：2026-10-03。当前 Wails v2.10.2 继续作为已修复的运行基线。下一轮优先保留 Go、React、TypeScript 和用户数据，推进 Wails v3、插件独立小窗与开发调试能力。本文是计划，尚未执行框架迁移；不同时更换 UI 框架、数据库和插件包格式。

## 目标目录

| 目标 | 优先级 | 状态 | 依赖 |
| --- | --- | --- | --- |
| [Wails v3 迁移](./wails-v3-migration.md) | P0 | planned | 无；先确认固定 Beta 版本与平台基线 |
| [服务与宿主接口整理](./services-and-host-api.md) | P0 | planned | v3 基础壳可运行；作为迁移中的独立步骤推进 |
| [插件会话与独立小窗](./plugin-windows-and-sessions.md) | P0 | planned | v3 窗口、绑定和宿主接口 |
| [开发工作流与测试](./development-and-testing.md) | P0 | planned | 基础命令可先整理；原生检查依赖 v3 与小窗 |
| [插件加载与 SDK](./plugin-loading-and-sdk.md) | P1 | planned | 宿主接口、会话与调用边界 |
| [日志诊断与 Agent 工具](./diagnostics-and-agent-tools.md) | P1 | planned | 日志约定可先做；MCP 与窗口诊断依赖 v3 |
| [插件信任与权限边界](./plugin-trust-and-permissions.md) | P1 | planned | 威胁模型可先明确；执行依赖 SDK 与会话边界 |

P0 为下一轮核心目标，P1 为后续增强。`planned` 是待办；`in-progress` 是实施中；`blocked` 必须记录具体阻碍；`done` 必须有实际验收证据。当前没有项目标记完成。

## 分阶段推进

1. **恢复现有能力**：固定 v3 版本，迁移启动、资源、绑定、事件与构建；整理必要服务，建立统一开发和检查入口。
2. **完整运行插件小窗**：实现宿主会话、按 ID 操作窗口和多窗口一致性；用一至两个真实插件验收，不仅验证窗口创建成功。
3. **提高日常开发效率**：补充浏览器模拟、插件开发目录加载、按需执行、调用链诊断和 agent 自动化；扩展权限设计与插件兼容测试。

每阶段独立提交、保留可运行状态。平台验证从 macOS 和 Windows 开始，Windows 编译与 Windows 实机验证分别记录；Linux 不是当前新增承诺。

## 已确定的边界

- macOS 不恢复系统 Keychain；凭据当前使用本地明文文件，不以本轮优化名义引入新的系统凭据依赖。
- 保留 `.wt` 兼容和 SQLite 数据；SDK 可渐进升级，不要求全部插件一次改写。
- 不假设 iframe 的 DOM 与执行环境可直接搬到新窗口；先采用明确的状态与上下文交付方案。
- 不预设性能数字。先测当前基线，再比较启动、常驻资源、插件加载和发布体积。
- 现有可信插件模型不等于任意第三方代码可安全执行；详见权限目标。

## 完成记录与关联材料

各目标文件的“实施记录”用于登记开始时间、提交、检查结果与剩余验收。结束实施时同步本页状态及相关现状文档。

- [当前架构](../architecture/architecture.md) 与 [实现机制](../architecture/implementation-mechanism.md)
- [已知问题修复](../reviews/remediation-2026-10-03.md) 与 [全仓审查](../reviews/review-2026-10-02.md)
- [历史技术调研](../research/technology-direction-2026-10-02.md)：候选比较与依据，当前执行顺序由本索引维护。
