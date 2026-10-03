# 服务与宿主接口整理

[返回 Backlog](./README.md)

状态：done。优先级：P0。依赖：[Wails v3 基础迁移](./wails-v3-migration.md)。目标是减少重复转发与手工类型解析，让宿主能力可单独测试、前端可替换调用实现。

## 现状问题

实施前 `internal/coordinator/coordinator.go` 是单一绑定出口，包含转发和 `map[string]interface{}` 参数转换；部分模块依赖全局单例与 Wails context。`frontend/src/api/` 直接引用生成绑定。实施前 browser 模式支持真实 Go 服务，固定场景模拟与可替换 Host API 尚未实施。

## 实施范围

- 按应用、插件、存储、更新等实际职责注册 v3 服务；删除无实际作用的转发，不为每个方法新增一层服务。
- 请求和响应使用明确 Go 类型，让绑定生成 TS 类型；统一错误码和可展示信息，保留必要底层上下文。
- 显式构造依赖，只为窗口、文件、网络等需要替换的边界设计接口；逐步减少全局单例。
- 建立有类型的 Host API，提供 Wails 与浏览器模拟实现。插件使用自己的 SDK，不直接接触全部宿主生成绑定。
- 保持 SDK 兼容层，按职责清理平台重复代码；同步说明绑定出口变化。

## 验收与验证

- 核心业务在无 Wails 事件循环的测试中运行。
- 前端不依赖具体生成路径，真实实现与模拟实现通过同一类型约束。
- 参数错误有明确结果，不依靠无声类型转换和默认零值。
- 绑定可重复生成，无需手工修改；关键调用和错误行为通过测试。
- 没有增加无职责的包装层，兼容范围和破坏性变更有明确记录。

## 实施记录

2026-10-04 已拆分七个领域服务，移除 coordinator 与 API 中未实现的重复存储出口；显式创建业务实例，并让插件服务、存储与资源路由共享实例。命令和插件列表返回具名响应；typed DTO 拒绝缺失次数、分数、无效日期和有破坏性的 null 参数。JSON storage 与凭据 SDK 保持兼容，Go 底层错误可 Unwrap，传输与展示错误统一 code/message/operation。

前端 HostAPI 与 Wails/mock 两个实现完成，只有真实适配器引用生成路径，插件 SDK、legacy runtime 尺寸与所有管理页调用方已迁移。mock 使用固定场景独立内存状态；其余 legacy runtime 原生能力不属于本项模拟范围。App/hotkey、DB/config/secrets 仍保留必要进程级共享资源，未进行无职责的全仓接口化。

实现提交：包含本记录的 `feat: split domain services and add typed host API`。

最终验收：16 个工作流回归、18 个前端回归、Go 全包与 race、server 变体、vet、生产构建通过；23 个生成 TS 文件重复生成完全一致。无 Go 的 headless mock 搜索与插件开关通过。真实 macOS WebView 的 37 个绑定、两种旧官方插件安装、JSON/凭据、无效参数拒绝、共享资产状态、日志、窗口尺寸/显隐、SPA 导航、Reload 后恢复和正常退出通过。双 agent 独立 review → 修复 → 回归 → 独立复审无阻塞。发现并修复 nil 响应与空字符串混淆，RawMessage 保持 JSON null；凭据覆盖/插件停用拒绝 null 并保持原值。

证据见 [验收摘要](../reviews/evidence/2026-10-04/services-acceptance.txt) 和 [材料入口](../reviews/evidence/2026-10-04/README.md)。插件多窗口、完整原生输入法/快捷键及 Windows/分发验收继续归属各自目标。
