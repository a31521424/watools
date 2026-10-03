# WaTools Agent 工作约束与阅读入口

本文件规定仓库内的工作方式，并提供按任务阅读的文档入口。先读本文件，再打开相关专题；完整目录见 [docs/README.md](./docs/README.md)，不要默认通读全部文档。

## 当前实现与后续方向

- 当前是 Go + Wails v3.0.0-beta.27 + React 19 + TypeScript + Vite 7，Go 工具链以 `go.mod` 和 CI 为准。
- Go 负责原生能力、命令扫描、插件安装和 SQLite 持久化；React 负责启动器、插件页与管理界面。
- 插件提供 `executable` 和 `ui` 入口；当前 UI 插件在主窗口 iframe 中运行。
- Wails v3 代码迁移已落地，原生验收按 backlog 记录。下一步推进服务与宿主接口整理，为插件独立小窗提供基础，详见 [backlog](./docs/backlog/README.md)。服务整理与插件小窗尚未实现，不能作为现有 API 使用。
- 阶段需求以 [开发者自用需求](./docs/requirements/developer-self-use.md) 为准：优先当前 macOS 开发机上的稳定使用与快速调试，Windows 兼容和对外分发任务延期。保留现有平台实现，未完成的延期项不作为自用阶段完成条件。
- macOS 按用户要求不使用系统 Keychain；凭据当前保存为应用数据目录内的本地明文文件。Windows 仍使用 DPAPI，不要把 Secret SDK 名称解释为所有平台都加密。

## Agent 工作约束

1. 先确认当前任务范围、工作区改动和相关调用链。保留已有工作，不擅自丢弃、重置或覆盖用户改动。
2. 将当前实现、历史审查、未来方案分开。Backlog 是计划，不是自动实施授权；只处理本次明确要求的范围。
3. 按任务索引读取最少必要文档，再核对源码。代码与文档不一致时，明确指出并同步修正文档，不把历史描述当作当前事实。
4. 当前 Wails v3 注册 `WaAppCoordinator` 服务。新增宿主能力仍从此出口暴露；拆分领域服务时同时更新调用方、生成绑定、生命周期和文档，不混用 v2/v3 API。
5. Wails 绑定与 sqlc 输出属于生成代码。优先修改生成源并重新生成，不通过手改输出制造与生成源不一致的实现。
6. 改动围绕明确职责；只在需要替换、模拟或平台分离的边界增加接口，不为了分层增加重复转发。业务接口使用明确类型。
7. 不使用用户真实凭据或唯一数据目录做破坏性验证。开发和测试使用独立临时数据、缓存目录；日志不写入凭据或完整剪贴板内容。
8. 现有插件为可信插件模型。iframe、Worker、独立窗口和 SDK 包装都不等于安全沙箱；权限改造必须验证真实调用来源与资源范围。
9. 对改动运行相关检查。区分编译成功、模拟测试、原生 GUI 验收和实机验证，不宣称未执行的检查已通过。
10. 未经用户要求不主动分派子 agent。外部消息、发布和推送遵循用户明确授权；不要把本地提交等同于远端发布。
11. 修改行为、接口或开发流程时，同步更新对应现状文档与入口索引。移动文档须修正引用和探针路径，保留有价值的历史证据。
12. 完成任务后说明结果、验证及未解决事项；只有验收实际满足后才将 backlog 标记完成。

## 按任务读取文档

| 任务 | 首先读取 | 按需补充 |
| --- | --- | --- |
| 确定需求、优先级或验收范围 | [开发者自用需求](./docs/requirements/developer-self-use.md) | [Backlog](./docs/backlog/README.md) 与本次目标文档 |
| 初次定位仓库 | [架构](./docs/architecture/architecture.md) | [功能模块](./docs/architecture/feature-modules.md) |
| 修改绑定、资源、命令或数据机制 | [实现机制](./docs/architecture/implementation-mechanism.md) | 对应源码与 [修复记录](./docs/reviews/remediation-2026-10-03.md) |
| 修改前端界面 | [UI 风格](./docs/design/ui-style.md) | [功能模块](./docs/architecture/feature-modules.md) |
| 开发或维护插件 | [插件开发索引](./docs/plugin-development/README.md) | 索引中的相关章节、[官方插件工具](./plugins/README.md) |
| 修改发布和更新流程 | [发布说明](./docs/development/release-cd.md) | `.github/workflows/`、`internal/update/` |
| 本机启动、检查、构建或插件调试 | [本地开发指南](./docs/development/local-workflow.md) | `scripts/dev.mjs`；无宿主模拟与 v3 多窗口仍在 backlog |
| 验证已知缺陷或安全边界 | [修复记录](./docs/reviews/remediation-2026-10-03.md) | [原始审查](./docs/reviews/review-2026-10-02.md)、[证据索引](./docs/reviews/evidence/README.md) |
| 推进下一轮优化 | [Backlog 入口](./docs/backlog/README.md) | 本次任务对应的目标文档及其依赖 |
| 重新评估框架和语言 | [历史技术调研](./docs/research/technology-direction-2026-10-02.md) | 官方最新资料；当前推进顺序以 backlog 为准 |

## 文档分类

| 路径 | 用途 |
| --- | --- |
| [docs/README.md](./docs/README.md) | 文档维护规则、阅读路径和完整文件目录 |
| [docs/requirements/](./docs/requirements/developer-self-use.md) | 项目需求、阶段范围和验收边界，不代表已经实现 |
| [docs/architecture/](./docs/architecture/) | 当前架构、机制和功能 |
| [docs/design/](./docs/design/) | 当前 UI 与交互规范 |
| [docs/development/](./docs/development/) | 当前本机统一任务、数据隔离、调试与发布流程 |
| [docs/plugin-development/](./docs/plugin-development/README.md) | 插件作者与 agent 的专题指南 |
| [docs/backlog/](./docs/backlog/README.md) | 尚未完成的目标、依赖和验收标准 |
| [docs/research/](./docs/research/) | 有日期的调研与候选方案，不作为最终执行计划 |
| [docs/reviews/](./docs/reviews/) | 审查、修复记录和有日期的验证证据 |

## 源码定位

- `main.go`、`config/`、`wails.json`：应用入口、目录配置与当前 Wails v3 初始化与构建。
- `internal/coordinator/`：当前唯一的 Wails 绑定出口。
- `internal/app/`、`internal/app_menu/`：窗口、快捷键、剪贴板、原生菜单与平台行为。
- `internal/command/`：应用扫描、命令触发和文件监听。
- `internal/plugin/`、`internal/handler/`：插件安装、状态、存储及资源路由。
- `internal/api/`、`internal/secrets/`、`internal/update/`：宿主能力、凭据与更新。
- `frontend/src/components/watools/`、`frontend/src/stores/`、`frontend/src/api/`、`frontend/src/lib/`：界面、状态、宿主调用与插件运行机制。
- `frontend/bindings/`、`pkg/db/`：生成绑定、SQLite 查询和迁移；`pkg/logger/`：日志。
- `cmd/pluginctl/`、`plugins/`：插件工具与官方插件源码；`scripts/`、`.github/workflows/`：检查和发布自动化。
