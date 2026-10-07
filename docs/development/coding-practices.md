# 编码与故障处理约定

[文档索引](../README.md) · [本地检查](./local-workflow.md) · [实施计划](../backlog/architecture-simplification.md)

本约定适用于新增和修改的手写代码。现有缺项按 [2026-10-07 审查](../reviews/architecture-and-simplification-2026-10-07.md) 分批修复；规范不代表历史代码已经全部符合。以业务行为、可处理的错误和明确所有权为依据，避免通用框架和无语义的全仓格式改动。

## Go 错误与提交

- 查询失败返回 error；空结果、未找到、部分成功有明确契约。失败不触发假空初始化，不用零值或 nil 数据隐瞒故障。
- 返回错误时增加必要的操作上下文；调用者需要识别的 cause 用 `%w`/Unwrap 保留，用 `errors.Is/As` 判断，不按错误文字分支。是否暴露底层类型是 API 决策，只给实际需要处理的业务类别定义 sentinel 或小型类型。
- 主操作与补偿都失败时分别包装并 `errors.Join`。调用方必须能识别两个 cause；回滚失败后不能按主错误的“可重试”分类直接重复副作用。
- Host 负责错误码和安全用户消息，业务模块提供错误事实；参数错误、忙碌、取消、超时与 I/O 分开。日志不得包含真实凭据和完整剪贴板内容；wire 消息不直接透传未知底层文本。
- 事务 helper 拥有 Begin/Commit/Rollback，回调只执行事务内工作。成功日志、统计与通知以提交成功为依据。`sql.ErrTxDone` 是事务已结束的清理结果，Commit 失败后不能假定再次 Rollback 会恢复底层驱动。DB helper 在 Commit 或真正的 Rollback 失败时保留 `ErrTransactionOutcomeUnknown`，调用方不得自动重复结果不明的副作用。
- 最终处理者记录失败；消费错误继续降级的层说明降级结果。重试过程与最终失败分级，正常取消不制造错误噪声。

包装与 API 契约参见 [Go：Working with Errors](https://go.dev/blog/go1.13-errors)，事务生命周期参见 [Go：Executing transactions](https://go.dev/doc/database/execute-transactions)。项目具体错误码沿用 HostAPI；修改码或返回签名同时更新契约、mock、生成源与调用方。

## 生命周期、资源与并发

- 长操作传递 context，明确单次调用、后台任务和服务生命周期。派生取消函数、goroutine、定时器、事件监听和进程有负责退出的所有者。
- 取消需要终止或使迟到结果失效；普通子任务收尾有界，安装提交阶段等待完成或恢复。取消信号失败不能先宣称已经取消。
- 必需写入、Close/Sync、提交及恢复失败影响成功结果，必须返回或合并；只读资源关闭和尽力清理按实际影响处理并说明必要的例外，不机械改写所有 defer。
- 状态有唯一所有者，锁保护状态而不是无关慢操作；锁外工作返回后重新核对 generation/当前状态。必须串行的安装切换继续串行，减少锁范围时验证交错与退出。
- 重试只用于可恢复且可重复执行的操作。取消和永久失败立即结束，报告实际次数。watcher 当前仅对错误树所有叶子均为 SQLite BUSY 的整操作有界重试；包含事务结果不明标记或其他补偿失败则停止。解析错误区分根路径不存在、非应用、暂时不可读与无效元数据；不可读/无效元数据保留旧记录，以后续事件或周期扫描恢复，不按次数盲目重试。

SQLite BUSY 与 LOCKED 的区别见 [SQLite result codes](https://www.sqlite.org/rescode.html)；二者不能统一视为等待即可恢复。context 与 goroutine 所有权参见 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)。

## 前端状态与边界

- reducer/updater 保持纯函数；关联状态一次更新，不用 updater 修改外部变量供另一个状态更新读取。选择保存 ID，从当前集合派生对象。
- 请求使用 generation 或对应的取消机制；旧结果、旧错误和旧 finally 均不得影响新查询。分页 cursor 属于生成它的查询快照，刷新/切筛选/卸载后失效。
- Promise、store 启动、定时 flush 和事件监听有显式完成/失败/清理路径。错误反馈与诊断有负责方，持久化失败不能静默覆盖旧数据。
- 未知输入在边界校验，内部使用明确 DTO；仅 Wails 适配器引用生成绑定，业务 API/mocks 保持一致。不手改 Wails/sqlc 输出，不为类型断言绕过已知的契约问题。
- 复用有独立规则的纯函数或生命周期模块，保留平台、信任策略和所有权差异。依赖仅在需要替换、测试或平台分离的边界使用窄接口。

请求 cleanup 参见 [React：Synchronizing with Effects](https://react.dev/learn/synchronizing-with-effects#fetching-data)。

## 验证与交付

使用独立临时数据/缓存目录。对适用改动运行类型/格式/行为测试、Go vet 和相关 race；对错误/事务补故障注入，对状态补反序响应/取消/退出测试，对原生行为补原生验收。生成、vendor 排除在手写源码整理之外。

测试验证行为与故障状态，不镜像每条分支或仅断言源码形状。变更说明记录适用规范、用户可见行为、验证范围和剩余风险；编译、模拟、真实 React 挂载、原生 GUI、实机与公开发布分别报告。完整命令沿用本地检查入口。
