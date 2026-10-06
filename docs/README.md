# WaTools 文档系统

本目录按项目需求、当前实现、开发指南、未来计划和历史材料分类。Agent 首先读取 [根目录约束](../AGENT.md)，再按任务进入具体文档，不需要一次读取全部文件。当前应用使用固定 Wails v3 Beta.27，代码迁移与验收边界见 backlog；插件默认主窗口展开，Command+D 分离小窗的当前实现与剩余验收见插件会话 backlog。当前阶段以 macOS 开发者自用为优先，Windows 与对外分发任务延期。

## 分类与入口

| 分类 | 入口与用途 |
| --- | --- |
| 需求 | [开发者自用需求](./requirements/developer-self-use.md)：项目目标、自用场景、优先级和延期边界 |
| 架构 | [架构](./architecture/architecture.md)、[实现机制](./architecture/implementation-mechanism.md)、[功能模块](./architecture/feature-modules.md)：说明当前实现 |
| 设计 | [UI 风格](./design/ui-style.md)：当前界面定位、布局和交互规范 |
| 开发 | [本地开发](./development/local-workflow.md)：macOS 统一开发／安装更新命令、管理中心、隔离与调试；[发布流程](./development/release-cd.md)：发布与更新机制及迁移暂停记录 |
| 插件开发 | [专题索引](./plugin-development/README.md)：按需读取六个章节 |
| Backlog | [目标索引](./backlog/README.md)：优先级、依赖、实施范围和验收标准 |
| 调研 | [框架与语言调研](./research/technology-direction-2026-10-02.md)、[剪贴板与插件启动性能](./research/clipboard-launch-performance-2026-10-06.md)：保留讨论与实验依据，最新路线以 backlog 为准 |
| 审查 | [原始审查](./reviews/review-2026-10-02.md)、[修复记录](./reviews/remediation-2026-10-03.md)、[证据索引](./reviews/evidence/README.md)：有日期的发现和验证边界 |

## 按需阅读

- 了解项目：架构 → 功能模块；需要具体调用链时再读实现机制。
- 修改界面：UI 风格 → 对应组件；需要宿主能力时补充实现机制。
- 修改后端：实现机制 → 对应包；涉及历史缺陷时读取修复记录和指定证据。
- 开发插件：插件开发索引 → 相关章节 → 对应官方插件示例。
- 推进优化：backlog 索引 → 本次目标文档 → 依赖和源码；不要默认启动整个 backlog。
- 调整需求或验收：开发者自用需求 → backlog 索引 → 本次目标文档；延期平台项不混入当前阶段完成条件。
- 修改发布：发布流程 → CI 和更新源码。
- 启动、检查或调试：本地开发指南 → 对应任务；可选 Host API 固定场景见本地开发指南；原生多窗口仍按 backlog 推进。

## 文档维护规则

1. 项目需求与阶段边界写入 requirements，具体实施目标写入 backlog；当前实现写入 architecture、design、development 和 plugin-development，不把需求或计划混写为已具备的能力。
2. 一份文档聚焦一个主题，使用明确文件名。新增、移动、删除文件时同步更新本页完整目录、相关专题入口和引用；改变分类时同步更新 `AGENT.md`。
3. Backlog 项目包含状态、优先级、依赖、现状问题、范围、验收与验证。状态变化时同时更新目标文档和 backlog 索引。
4. 完成 backlog 后，登记实现提交、实际检查和未覆盖的验收，并更新现状文档。需要 GUI 或实机验收的项目不能仅凭编译成功标记完成。
5. Research 与 reviews 保留日期和适用版本。历史推荐不覆盖当前计划，旧探针和扫描结果不代表当前代码状态。
6. 验证材料放在 `reviews/evidence/<日期>/`，使用索引解释来源、命令和范围；不要写入真实用户数据或凭据。
7. 使用相对 Markdown 链接。移动文件时也检查代码块路径、脚本的仓库根目录定位和外层 README 引用。

## 仓库其他入口

- [AGENT.md](../AGENT.md)：统一约束和任务索引；[AGENTS.md](../AGENTS.md) 为标准 agent 入口。
- [README](../README.md) 与 [中文说明](../README.zh-CN.md)：对外介绍。
- [官方插件目录](../plugins/README.md)：插件源码、打包和安装；每个插件目录保留自身 README。

## 完整文件目录

下方包含本目录内所有文档、历史探针和验证输出；用途见分类入口与证据索引。维护时以实际文件系统为准更新。

```text
docs/
├── architecture/
│   ├── architecture.md
│   ├── feature-modules.md
│   └── implementation-mechanism.md
├── backlog/
│   ├── README.md
│   ├── development-and-testing.md
│   ├── diagnostics-and-agent-tools.md
│   ├── distribution-and-update-acceptance.md
│   ├── native-self-use-acceptance.md
│   ├── plugin-loading-and-sdk.md
│   ├── plugin-trust-and-permissions.md
│   ├── plugin-windows-and-sessions.md
│   ├── services-and-host-api.md
│   ├── wails-v3-migration.md
│   └── windows-compatibility.md
├── design/
│   └── ui-style.md
├── development/
│   ├── local-workflow.md
│   └── release-cd.md
├── plugin-development/
│   ├── 01-llm-must-read.md
│   ├── 02-templates-and-packaging.md
│   ├── 03-runtime-and-context.md
│   ├── 04-api-and-browser.md
│   ├── 05-ui-and-shortcuts.md
│   ├── 06-reference-and-checklist.md
│   └── README.md
├── requirements/
│   └── developer-self-use.md
├── research/
│   ├── clipboard-launch-performance-2026-10-06.md
│   └── technology-direction-2026-10-02.md
├── reviews/
│   ├── evidence/
│   │   ├── 2026-10-02/
│   │   │   ├── README.md
│   │   │   ├── api-probe_test.go.txt
│   │   │   ├── db-probe_test.go.txt
│   │   │   ├── govulncheck.txt
│   │   │   ├── pnpm-audit-summary.json
│   │   │   ├── runtime-probes-output.txt
│   │   │   ├── runtime-probes.cjs
│   │   │   ├── update-probe_test.go.txt
│   │   │   └── watcher-probe_test.go.txt
│   │   ├── 2026-10-03/
│   │   │   ├── README.md
│   │   │   ├── frontend-audit.json
│   │   │   ├── frontend-build.txt
│   │   │   ├── frontend-tests.txt
│   │   │   ├── macos-production-build.txt
│   │   │   ├── official-package.txt
│   │   │   ├── wails-v3-acceptance.txt
│   │   │   ├── watools-final-go-race.txt
│   │   │   ├── watools-final-go-test.txt
│   │   │   ├── watools-final-go-vet.txt
│   │   │   ├── watools-final-vuln-macos.txt
│   │   │   ├── watools-final-vuln-windows.txt
│   │   │   ├── watools-final-wails-build.txt
│   │   │   └── watools-final-windows-build.txt
│   │   ├── 2026-10-04/
│   │   │   ├── README.md
│   │   │   ├── checks.txt
│   │   │   ├── focus-helper.m.txt
│   │   │   ├── hotkey-probe.go.txt
│   │   │   ├── launcher.png
│   │   │   ├── native-acceptance.txt
│   │   │   ├── native-harness.py.txt
│   │   │   ├── native-probe.go.txt
│   │   │   ├── rapid-exit.py.txt
│   │   │   ├── services-acceptance.txt
│   │   │   └── services-native-harness.py.txt
│   │   ├── 2026-10-06/
│   │   │   ├── README.md
│   │   │   ├── clipboard-launch-probe.cjs
│   │   │   ├── clipboard-launch-results.json
│   │   │   ├── clipboard-checks.txt
│   │   │   ├── clipboard-native-harness.py.txt
│   │   │   ├── clipboard-helper.swift.txt
│   │   │   ├── clipboard-native-results.jsonl
│   │   │   ├── json-native-harness.py.txt
│   │   │   ├── json-native-results.jsonl
│   │   │   ├── match-context-native-harness.py.txt
│   │   │   ├── match-context-native-results.jsonl
│   │   │   ├── launcher-refactor-native-results.jsonl
│   │   │   ├── launcher-refactor-json-results.jsonl
│   │   │   ├── plugin-window-checks.txt
│   │   │   ├── plugin-window-native-harness.py.txt
│   │   │   ├── plugin-window-status.swift.txt
│   │   │   ├── plugin-window-native-results.jsonl
│   │   │   ├── plugin-upgrade-checks.txt
│   │   │   ├── plugin-upgrade-native-harness.py.txt
│   │   │   ├── plugin-upgrade-native-results.jsonl
│   │   │   ├── plugin-window-focus-checks.txt
│   │   │   ├── plugin-window-focus-native-harness.py.txt
│   │   │   ├── plugin-window-focus-native-results.jsonl
│   │   │   ├── plugin-window-focus-status.swift.txt
│   │   │   └── plugin-window-focus-helper.m.txt
│   │   ├── 2026-10-07/
│   │   │   ├── README.md
│   │   │   ├── local-install-native.jsonl
│   │   │   ├── go-workflow-native.jsonl
│   │   │   ├── go-workflow-native-probe.py.txt
│   │   │   ├── go-run-cancellation.jsonl
│   │   │   ├── go-run-cancellation-probe.py.txt
│   │   │   ├── go-dev-native.jsonl
│   │   │   └── go-dev-native-probe.py.txt
│   │   └── README.md
│   ├── remediation-2026-10-03.md
│   └── review-2026-10-02.md
└── README.md
```
