# WaTools 优化 Backlog

[返回文档系统](../README.md) · [Agent 约束](../../AGENT.md)

登记与需求调整日期：2026-10-03。Wails v2 基线保留在提交 `809f635`；当前宿主代码使用固定 Wails v3 Beta.27。下一轮优先保留 Go、React、TypeScript 和用户数据，推进 Wails v3、插件独立小窗与开发调试能力。按用户最新要求，先覆盖当前 macOS 开发机上的自用需求，Windows 兼容和对外分发延期；完整范围见 [项目需求](../requirements/developer-self-use.md)。开发工作流的基础入口已落地，v3 代码迁移已实施，原生验收边界见目标记录；不同时更换 UI 框架、数据库和插件包格式。

## 目标目录

| 目标 | 优先级 | 状态 | 依赖 |
| --- | --- | --- | --- |
| [Wails v3 迁移](./wails-v3-migration.md) | P0 | in-progress | 固定 Beta.27；代码已迁移，待完成原生验收 |
| [服务与宿主接口整理](./services-and-host-api.md) | P0 | done | 七个领域服务/HostAPI、无 Go mock、原生绑定与独立复审通过 |
| [插件会话与独立小窗](./plugin-windows-and-sessions.md) | P0 | in-progress | 主窗展开／Command+D 分离与多实例已实施；共享状态与实机体验待验 |
| [开发工作流与测试](./development-and-testing.md) | P0 | in-progress | 基础入口与数据隔离已实施；模拟适配与原生检查仍依赖后续目标 |
| [macOS 自用原生验收](./native-self-use-acceptance.md) | P0 | in-progress | v3 原生回归已推进；物理输入法、快捷键及完整场景待验 |
| [插件加载与 SDK](./plugin-loading-and-sdk.md) | P1 | planned | 宿主接口、会话与调用边界 |
| [日志诊断与 Agent 工具](./diagnostics-and-agent-tools.md) | P1 | planned | 日志约定可先做；MCP 与窗口诊断依赖 v3 |
| [插件信任与权限边界](./plugin-trust-and-permissions.md) | P1 | planned | 威胁模型可先明确；执行依赖 SDK 与会话边界 |
| [Windows 兼容与实机验收](./windows-compatibility.md) | P2 | deferred | 有明确 Windows 使用需求后恢复 |
| [对外分发与自更新验收](./distribution-and-update-acceptance.md) | P2 | deferred | 有对外发布或自动升级需求后恢复 |

P0 为当前自用阶段核心目标，P1 为自用开发体验增强，P2 为延期的平台与分发任务。`planned` 是待办；`in-progress` 是实施中；`deferred` 是主动延期，记录恢复条件；`blocked` 必须记录具体阻碍；`done` 必须有实际验收证据。服务与宿主接口整理已完成；其他项目按表中状态推进。权限目标本阶段按可信自用插件验收，面向不可信第三方的完整隔离另列延期范围。

## 分阶段推进

2026-10-07 用户授权本机一键安装和内置管理中心，实施记录见[本地指南](../development/local-workflow.md)与[原生验收证据](../reviews/evidence/2026-10-07/README.md)；GitHub 商店和公开分发未纳入本次。

2026-10-04 用户撤回低干扰 browser/test 系统能力适配与 VM runner 的新增安排，沿用现有 dev flow。P0 **[服务与宿主接口整理](./services-and-host-api.md)** 已完成。2026-10-06 用户明确要求继续实施 **[插件会话与独立小窗](./plugin-windows-and-sessions.md)**，并确定默认主窗展开、Command+D 分离和同插件多实例；该交互已实施，剩余验收见目标记录。其他 backlog 不因本次工作自动授权。

1. **恢复现有能力**：固定 v3 版本，迁移启动、资源、绑定、事件与本机构建；整理必要服务，建立统一开发和检查入口，用 macOS 原生场景验证。
2. **完整运行插件小窗**：实现宿主会话、按 ID 操作窗口和多窗口一致性；用一至两个真实插件在本机验收，不仅验证窗口创建成功。
3. **提高日常开发效率**：补充浏览器模拟、插件开发目录加载、按需执行、调用链诊断和 agent 自动化；扩展权限设计与插件兼容测试。

每阶段独立提交、保留本机可运行状态。当前验收以 macOS 开发机为准，记录系统与架构；Windows 编译、实机验证与安装器发布均延期，macOS universal/Intel 分发覆盖同属后续分发范围。保留已有代码与检查，遇到平台问题单独登记，不将其作为自用阶段的阻碍。Linux 与移动端不纳入本阶段。

## 已确定的边界

- macOS 不恢复系统 Keychain；凭据当前使用本地明文文件，不以本轮优化名义引入新的系统凭据依赖。
- 保留 `.wt` 兼容和 SQLite 数据；SDK 可渐进升级，不要求全部插件一次改写。
- 不假设 iframe 的 DOM 与执行环境可直接搬到新窗口；先采用明确的状态与上下文交付方案。
- 不预设性能数字。先测当前基线，再比较启动、常驻资源、插件加载和发布体积。
- 现有可信插件模型不等于任意第三方代码可安全执行；详见权限目标。
- 自用可使用本机构建和手动替换的安装方式，保留旧包与数据；签名公证、安装器分发与真实自动升级故障演练不阻塞本阶段。

## 补充验收事项对应关系

| 原总结中的补充事项 | 明确归属 | 当前安排 |
| --- | --- | --- |
| 原生 GUI、输入法、快捷键、剪贴板、对话框和插件重载 | [macOS 自用验收](./native-self-use-acceptance.md) | P0 in-progress |
| Windows 快捷方式参数、工作目录、快捷键和 DPAPI | [Windows 验收](./windows-compatibility.md) | P2 deferred |
| 真实更新替换、重启、失败恢复、签名与公证 | [分发与更新验收](./distribution-and-update-acceptance.md) | P2 deferred |
| 实际运行新增 CI | [开发工作流](./development-and-testing.md) | 本地与 macOS 优先；Windows 部分延期，不为触发 CI 自动推送 |
| 重新打包安装旧官方插件 | [macOS 自用验收](./native-self-use-acceptance.md) | P0 in-progress，继续在独立测试目录执行 |

## 完成记录与关联材料

各目标文件的“实施记录”用于登记开始时间、提交、检查结果与剩余验收。结束实施时同步本页状态及相关现状文档。

- [当前架构](../architecture/architecture.md) 与 [实现机制](../architecture/implementation-mechanism.md)
- [已知问题修复](../reviews/remediation-2026-10-03.md) 与 [全仓审查](../reviews/review-2026-10-02.md)
- [历史技术调研](../research/technology-direction-2026-10-02.md)：候选比较与依据，当前执行顺序由本索引维护。
