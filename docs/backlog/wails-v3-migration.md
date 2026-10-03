# Wails v3 迁移

[返回 Backlog](./README.md)

状态：in-progress。优先级：P0。目标是在保留用户数据和插件兼容的前提下，把桌面宿主从 Wails v2.10.2 迁移到固定的 v3 Beta 版本，为原生多窗口提供基础。按 [自用需求](../requirements/developer-self-use.md) 优先当前 macOS 开发机，Windows 与多架构分发不作为本阶段完成条件。

## 现状与依赖

迁移前的 v2.10.2 基线保留在提交 `809f635`。当前 `main.go` 使用 v3 application/service/window，绑定位于 `frontend/bindings/`，旧 `frontend/wailsjs/` 已移除。macOS 桥接操作显式启动器的 NativeWindow，不再查询任意 key/main window。代码已实施，物理输入法、系统权限及完整原生体验仍按下述边界验收。

## 实施范围

- 实施时核对官方版本、平台要求和破坏性变更，固定 Go 模块、CLI 和前端运行时的兼容版本，记录依据。
- 迁移初始化、服务注册、启动关闭、资源路由、事件和生成绑定；保持插件资源 revision 路径与用户数据。
- 迁移本机构建任务与开发启动，先生成当前 macOS 架构可运行的应用；保持发布配置有明确迁移记录，跨平台与 universal 发布完整验收见延期任务。
- 检查 macOS 原生桥接，优先使用已验证的框架能力；不能替代的部分保留明确的平台边界。
- 保留升级前可运行提交，记录数据读取、失败恢复和回退限制；不同时改写数据库或插件包格式。

## 验收与验证

- 主窗口的呼出、隐藏、搜索、命令触发和原有插件页可正常运行。
- 插件安装、重装、启停、存储和旧数据读取通过验证。
- 日志、更新检查、资源服务、快捷键、菜单和剪贴板没有因迁移丢失。
- 本机 Go 与前端检查通过，当前 macOS 架构的原生应用可构建并启动；Windows、universal/Intel、签名和发布包列入 [Windows](./windows-compatibility.md) 与 [分发](./distribution-and-update-acceptance.md) 延期项。
- 原生焦点、中文输入法、透明窗口和系统权限在测试安装验收，不能用编译结果替代。

迁移完成不代表插件小窗已完成；见 [插件会话与独立小窗](./plugin-windows-and-sessions.md)。服务整理见 [宿主接口目标](./services-and-host-api.md)。

## 官方依据

- [v2 到 v3 迁移](https://v3.wails.io/migration/v2-to-v3/)
- [v3 状态与平台要求](https://v3.wails.io/status/)
- [多窗口](https://v3.wails.io/features/windows/multiple/)

这些链接是实施入口，具体能力以实施时固定版本为准。

## 实施记录

2026-10-03 实施。固定 Go 模块/CLI `v3.0.0-beta.27` 与 npm runtime `3.0.0-beta.27`，Go 工具链 `1.26.8`。实施时通过官方迁移文档、状态页，以及 Go proxy/npm registry 核对发布版本；具体调用核对该固定模块源码，避免滚动文档与 Beta API 不一致。当前实机 macOS 26.6.2 / arm64，Node 24.11.0、pnpm 10.21.0。

已实施：

- 初始化、服务生命周期、显式启动器窗口、事件、菜单、剪贴板、绑定与资源服务迁移到 v3。保留 `/api/plugin/<package>/_v/<revision>/...`，客户端页面重载显式回到 shell，缺失插件或静态资源仍返回错误。
- 重新生成 v3 TypeScript 绑定，插件 iframe 注入 `window.runtime` 的常用 v2 调用兼容层，保留剪贴板布尔返回值、事件可变参数及 `{w,h}` 尺寸形状。插件包和 SQLite schema 未修改；v3 依赖使 SQLite 驱动升级到 modernc.org/sqlite 1.44.3。
- 更新隔离开发/server 浏览器模式、watcher、当前架构 macOS 打包、图标与 ad-hoc 签名，固定 macOS 最低版本 12，保留原 bundle ID。浏览器/server 强制 loopback；默认应用没有 MCP 服务。公开 release 工作流明确暂停 v2 打包，待延期分发任务迁移后再恢复。
- v3 框架日志接入原旋转日志，过滤自动绑定/runtime 日志的 `args/result`，避免记录完整剪贴板与凭据。macOS 凭据仍为原本地文件方案。
- 原生启动回归发现并修复后台并发访问 NSPasteboard 的 SIGSEGV：Cocoa 读取与图片写入统一调度到主线程。原生页面重载还发现 v3 无 v2 自动 SPA fallback，已补显式 shell 路由并添加回归。

实际验证（合成 UI 操作与物理输入分开）；脱敏材料见 [v3 验收摘要](../reviews/evidence/2026-10-03/wails-v3-acceptance.txt)：

| 检查 | 结果与边界 |
| --- | --- |
| 本地检查 | 16 个工作流回归、11 个前端回归，TypeScript/Vite、Go 全包测试/vet/production 编译以及 server 两包回归及真实 HTTP 空剪贴板/图片写入拒绝/窗口空操作 smoke 通过；Go 全包 race 检查通过 |
| 依赖扫描 | 当前生产 Go govulncheck 与 pnpm audit 均未发现已知漏洞；不是未来安全保证 |
| 绑定 | 固定 CLI 再生成 10 个 TypeScript 文件，前后 SHA-256 一致；生成输出未手改 |
| 构建 | 当前 arm64 `.app` 构建与 ad-hoc 签名通过，实际 Mach-O deployment target 为 macOS 12；不代表 notarization/分发验收 |
| 浏览器真实服务 | 通过 9246 宿主读取 6 个旧插件与应用命令；搜索、计算器 iframe 加载及 `(2+3)*4 = 20` 历史保存通过，没有浏览器错误；9245 单独页面不是宿主 |
| 原生 WebKit | 隔离 `.app` 配合临时 `production,mcp` 构建，搜索并打开旧计算器、注入上下文、插件存储写入/删除、剪贴板读写和事件参数传递通过；剪贴板测试恢复原值，不将其写入验收记录 |
| 旧数据与重装 | 使用 v2 开发数据库的 SQLite 只读备份和插件 revision 副本，排除 secrets；读取 6 个插件通过，计算器重装保留禁用状态及存储，随后恢复启用并移除测试键 |
| 原生线程回归 | 修复后 5 次独立重启，每次 40 个并行 GetClipboardContentApi（合计 200）；另有 25 次图片写入与读取并发，PNG 可读且恢复原剪贴板；全部成功，不输出剪贴板内容 |
| 重载与焦点 | 原生隐藏后 visible/focused 均为 false，呼出后均为 true；插件路由 reload 后 iframe SDK 恢复、存储仍可读（未验证物理系统快捷键或其他应用的焦点回交） |
| 日志与更新 | 原生日志标记确实落入旋转文件，日志列表绑定可读；更新检查的错误经 v3 绑定正确传回，当前线上 manifest 返回 HTTP 404，未执行更新安装/重启 |

原生启动与 watcher 在执行沙箱中分别被 AppKit/FSEvents 限制；在已授权的本机执行环境测试通过，不能将沙箱中编译成功代替原生启动。测试使用独立 `/private/tmp` 数据和缓存目录；浏览器使用 `.cache/dev/` 开发安装。未读取或改写正式安装的数据库、凭据；测试构建不安装到 Applications。

恢复与回退：实施前备份正式安装的数据目录和插件，不删除旧可运行应用。回退代码可使用 `809f635` 在单独检出中重新构建 v2，并使用升级前的数据备份。虽然 schema/包格式未变、已验证 v2 备份可被 v3 读取，但没有验证 v3 写入后的所有文件对 v2 的反向兼容，不能把降级直接覆盖同一数据库作为已验证恢复方案。

未执行：物理全局快捷键、真实中文输入法组合/候选、透明窗口视觉检查、多屏实机切换、原生文件选择对话框、系统权限首次授权、真实更新安装重启、Windows/Intel/universal 与远端 CI。当前输入与键盘测试由浏览器或 Wails MCP 合成，不冒充物理输入法和全局快捷键验收。保持 in-progress，待这些必需的 macOS 原生体验验收后再标记 done；插件独立小窗不在本次迁移范围。

提交前按用户要求进行两位独立 subagent 交叉 review，分别覆盖原生/后端及前端/构建工作流。审查发现并修复：图片剪贴板写入口遗漏主线程调度；server build 的 InvokeSync 不提供原生主线程，自动剪贴板读取和宿主窗口操作仍可能进入 Cocoa。现在使用官方 `application.System.IsServer()` 在能力边界阻断这些访问：返回空剪贴板、窗口操作为空操作、图片写入明确报不支持；新增 server build 回归，并纳入本地 `check` 与 macOS CI。

两位独立复审均未发现剩余提交阻塞问题；独立执行工作流/前端/TypeScript、核心 Go 包 race、server build 回归与 diff 检查通过。独立审查同样不代表物理 IME/全局快捷键/系统权限验收。
