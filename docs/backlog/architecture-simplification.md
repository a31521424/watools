# 架构精简与编码实践实施计划

[Backlog 索引](./README.md) · [审查依据](../reviews/architecture-and-simplification-2026-10-07.md)

日期：2026-10-07；阶段结案更新：2026-10-09。优先级：P1。状态：in-progress（前四轮实施批次已阶段结案，原生验收及后续批次待推进）。报告基线提交为 `4665624`（原代码基线 `fb22aa1`）。用户已要求提交工作区、制定计划并充分使用 subagent 开始实施；本计划登记执行范围，历史审查的“待 review”描述保留作为当时状态。

## 目标与约束

保留 Go/Wails/React 模块化单体和七领域 Host 服务，改善失败语义、任务所有权和规则复用。每个模块按报告 C01–C11 验收，不以行数下降代替行为正确。保留用户数据、`.wt` 兼容、可信插件模型和 macOS 明文凭据策略；Windows 现有代码保留，实机和公开分发继续延期。

## 首轮并行实施

| 工作项 | 负责人/文件范围 | 报告对应 | 完成条件 |
| --- | --- | --- | --- |
| 安装补偿与 DB 事务 | installation_transactions：plugin installer/测试、wa_db/测试 | E02、E09 的事务部分、M06/M09 | 两个失败的 cause 都保留；helper 统一 Commit；Commit 后才记录成功；结果不明标记禁止重试；实际 SQLite 故障用例 |
| 本机安装与 Host 错误语义 | host_error_semantics：localinstall、host app/update/error、filelock/测试 | E03、E10 的锁分类部分 | 参数、忙碌、I/O、取消分开；安全 wire 消息且保留 cause；不新增错误码和绑定签名 |
| 日志异步与管理路由 | ui_route_reliability：日志页面及必要纯状态模块/测试、logger/兼容 bridge、handler/测试 | M10/M15/M16、C08/C09 | 旧响应/旧失败/旧 finally 不污染新查询；分页归属正确；updater 纯；warn 正确；管理路由有效且未知路由404 |
| watcher 重试与集成 | 主 agent：watcher 通用错误/重试/测试、文档与集成 | E06、M04/C01/C02/C10 | 平台重复收敛；取消/永久错误不重试；只重试明确可恢复故障；实际次数正确；等待可取消 |

并行采用共享工作区与明确文件所有权；subagent 不提交，主 agent 审查差异后统一集成。跨范围修改先协调，生成操作统一执行，避免覆盖其他修改。各分支使用独立临时数据/缓存，复用仓库工具缓存。

首轮不自动覆盖整份报告：扫描的部分成功与删除规则、热键恢复、全局初始化、SDK 并发、旧 revision 回收仍需按后续批次推进。

## 后续批次和依赖

| 顺序 | 范围 | 依赖及验收重点 | 状态 |
| --- | --- | --- | --- |
| 1 | DB 错误全链透传 E01：DB → command → Host → UI/mock | 首轮事务规则；失败不触发假空初始化，失败保留旧数据，统一重新生成绑定 | implemented，自动验收通过；原生交互待验证 |
| 2 | 扫描解析/协调 E05/E09、watcher 配置生效与共享规则 | E01；缺失与暂时不可读分开，已提交结果决定通知，并发 reconcile 回归 | implemented；自动验收记录见第三轮，原生/Windows 待验 |
| 3 | 热键补偿 E04、必需清理 E07、停止 E10 剩余部分 | 独立故障用例与状态契约；涉及原生行为增加原生验收，提交阶段不强杀；E08 显式初始化另轮推进 | implemented，第四轮自动验收通过；原生/Windows 边界见记录 |
| 3a | 显式初始化 E08 | 启动失败返回契约、目录/迁移/DB 故障与既有数据保持；本轮明确排除 | planned |
| 4 | 版本/catalog/解压/usage 调度收缩，store 生命周期 | 前述错误与提交协议；复用机制保留策略差异，不引入万能框架 | planned |
| 5 | launcher/controller、管理 hooks、iframe 生命周期、官方插件 JS 模块化 | 状态所有权；保留既有回归，直接导入行为模块，加生命周期/多窗测试 | planned |
| 6 | 加载、图标、日志、Worker、revision 回收等性能工作 | 对应独立 backlog、真实基线、兼容与取消验收；不预设性能数字 | planned |

可并行的范围由文件所有权和依赖决定；有数据一致性或契约依赖的工作必须按顺序集成。需要进一步选择的产品语义沿用报告 review 决策，不擅自改变统计含义、卸载凭据和共享草稿范围。

## 检查与交付

第二轮仍使用 3 个 subagent：后端负责查询/扫描调用链与 Host error 返回，前端负责加载状态/安全反馈/重试/usage 保持，独立验证负责真实 SQLite 与业务/Host/watcher 故障测试。主 agent 负责接口集成、统一重新生成 Wails 绑定、文档和最终检查。文件所有权分别限定在生产 Go、手写前端与新 Go 测试，避免共享文件互相覆盖。

E01 的完成条件：合法空库仍能初始化；查询失败不扫描、不写入、不通知、不转换零值记录；扫描/首次入库失败向上返回；失败不清空已有 runner 和前端搜索数据；Host 拒绝有稳定安全错误码并保留 cause；初始失败和事件刷新失败被消费，用户可重试恢复；请求期间的 usage 和待写 buffer 不丢失。

先跑对应故障/交错测试，再统一运行 `pnpm test`、前端 build/type 检查、Go 测试、Go vet、production 编译及相关 race 检查。采用本地统一工作流的临时数据隔离和 macOS 12 deployment flags。涉及 Host 返回签名的后续批次重新生成，不手改绑定。

每轮记录实际命令、结果、提交和剩余验收，更新对应现状文档；历史报告保持当时的证据。原生 GUI、物理热键、Windows 实机、真实安装与发布未执行时明确记录，目标整体仍保持 in-progress。

## 实施记录

- 2026-10-07：审查报告与索引已提交为 `4665624`；3 个 subagent 与主 agent 完成首轮并行实现及交叉复审。首轮代码已提交，目标整体仍为 in-progress，下一批优先 E01。

| 实施提交 | 完成范围 | 验收边界 |
| --- | --- | --- |
| `5aecf01` | E02 双错误保链、withTx 统一提交、成功日志时机、ErrTransactionOutcomeUnknown | 真实文件 rename 双故障、SQLite 中途回滚/批处理/延迟外键 Commit 故障、driver 生命周期故障通过；未实现底层故障连接恢复 |
| `ac70267` | E03 语义映射与快照 cause；E10 争锁分类与取消失败状态 | 安全消息、参数/文件/I/O/启动/取消故障、context、真实锁争用通过；Shutdown/强制停止机制仍待后续 |
| `dd70fa1` | watcher 共用错误/重试，保留平台默认配置；只重试明确 BUSY | 取消、次数、真实 BUSY、永久失败、BUSY+回滚故障/结果不明通过；解析类型与安装就绪策略待 E05 |
| `361be1d` | `/management` shell fallback | 真实 Wails HTTP handler 检查 query 参数、请求保持及未知路由404；未进行原生导航验收 |
| `d516f66` | 日志纯 reducer/generation/分页归属、选择ID派生、warn bridge | 前端回归新增8项，共68项通过；协调器 cleanup/restart 已测，真实 React StrictMode 挂载和原生 GUI 未执行 |

复审修正了三个边界：BUSY 主错误叠加回滚故障不可重试；Commit/真实 Rollback 失败必须保留结果不明标记；普通文件作为源码路径属于验证错误，真实权限/I/O 错误仍属操作失败。未新增 Host wire 错误码或公开方法签名，因此本轮无需重新生成绑定。

实际检查：

| 命令/范围 | 结果 |
| --- | --- |
| `go run ./cmd/watoolsctl check` | 通过：68项前端回归、前端 build、Go全量测试、server-tag app/api测试、Go vet、production编译 |
| `pnpm exec tsc --noEmit`（frontend） | 通过 |
| 最后事务标记补充后 `go test ./...` | 全量通过 |
| `go test -race ./internal/command/watcher ./internal/host ./internal/localinstall ./internal/filelock ./pkg/db ./internal/plugin` | 最终6包通过 |
| 最后补充后 `go vet ./...`、`go build -tags production,private_mac_apis ./...` | 通过 |
| Windows watcher/filelock/localinstall 测试交叉编译；filelock 额外 amd64 编译 | 通过；没有运行 Windows 实机测试 |
| 交叉复审、文档链接与 `git diff --check` | 通过 |

Go 检查使用独立临时数据/缓存与仓库 `.cache/tools` 工具缓存，设置 macOS 12 CGO/deployment flags。仍存在既有 objc/deployment target 链接警告，退出码为0；本轮未修改工具链链接策略。没有执行真实安装、原生 GUI、物理热键、真实 React 组件挂载、公开发布或性能基准。

### 第二轮：E01 查询错误全链透传

2026-10-07，用户要求继续。3 个 subagent 分别实施后端、前端及独立故障验证，并交叉复审；主 agent 集成并提交。本轮自动验收通过，目标整体继续为 in-progress，下一批推进 E05/E09。

| 实施提交 | 完成范围 | 验收边界 |
| --- | --- | --- |
| `f465bad` | 三个 DB 查询显式返回 error；合法缺失为 `nil, nil`；command/watcher 查询失败停止副作用；Host 安全拒绝并保留 cause；读取与 runner 发布串行 | 真实 SQLite 空库、关闭连接、缺表、取消/超时；扫描/入库/后续批次故障；旧 runner 保留；确定性并发发布测试通过，恢复旧锁位置的临时 overlay 探针明确失败 |
| `952d788` | 加载失败保留列表/搜索并提供重试；刷新期间旧命令可见；请求期间的 usage 保持；事件合并并补充后续查询；mock 查询、触发与 usage 使用同一快照 | 新增 12 项前端行为回归，共 80 项通过；初始/事件失败与恢复、合法空结果、并发调用、交错统计、事件突发、mock 克隆与失败、同步诊断故障均覆盖；未进行真实 React 挂载或原生点击验收 |

交叉复审补充修复了旧查询晚发布覆盖新 runner、查询期间事件被合并后丢失、mock 查询成功后触发/统计仍查旧快照，以及同步 Logger 抛错导致 usage 重试未安排的问题。后台 reconcile 后续批次失败时仅通知此前确已提交的变化，不宣称整轮原子提交；暂时不可读/解析失败的删除规则和平台扫描器部分成功仍属于 E05。

| 命令/范围 | 结果 |
| --- | --- |
| `go run ./cmd/watoolsctl bindings` | 重新生成并构建通过；生成文件无差异，成功返回仍为 `Promise<ApplicationCommand[]>`，Go error 由 Wails 拒绝 Promise |
| `go run ./cmd/watoolsctl check` | 通过：80 项前端回归、前端类型检查/build、Go 全量、server-tag app/api、Go vet、production 编译 |
| `pnpm exec tsc --noEmit`（frontend） | 通过 |
| `go test -race ./pkg/db ./internal/command ./internal/command/watcher ./internal/host` | 最终四包通过 |
| Windows arm64、`CGO_ENABLED=0` 的上述四包测试交叉编译 | 通过；未执行 Windows 运行时测试 |
| 交叉复审、gofmt、文档链接与 `git diff --check` | 通过 |

沿用临时数据/缓存、仓库工具缓存和 macOS 12 flags；既有链接警告仍在。本轮没有执行真实安装、原生 GUI、物理热键、真实 React 组件挂载、Windows 实机或公开发布；原生重试交互仍待验证。

### 第三轮：E05/E09 扫描解析、路径协调与 watcher 配置

2026-10-07，从 `a4932ad` 开始。按用户授权创建 3 个新 subagent（均 `fork_turns="none"`），分别独占 application 解析、watcher 配置/事件、新增独立故障与并发测试；主 agent 负责 command 协调、默认 handler、DB 元数据提交语义、文档和集成。保留 E01 与已有测试，未扩展至热键、初始化、SDK 或性能批次。

本轮实施：

- 解析返回四类 sentinel 与原始 cause；应用根缺失、非应用、读取故障、无效元数据分开。缺 plist 不删除旧记录，可选名称/图标/cache 失败明确降级。扫描保留有效部分并聚合故障；首次空库初始化仍遇故障停止，后台扫描可提交有效部分。
- 周期与 watcher 路径入口共享应用串行所有权，usage、查询/初始化与 runner 发布参与协调；事件携带路径提示，所有权内重新解析，保留 ID/TriggerID/usage。删除提交前再次检查根路径，防止替换窗口误删后重建身份。
- debounce/buffer 配置实际生效；共用配置验证/getter，保留平台默认 delay 和原生路径规则；构造和读取配置隔离可变目录列表。重复及交错事件按路径合并，以最终磁盘状态处理。
- SQL 元数据完整替换，允许清空不可用的可选图标/描述，保留 usage；修改 sqlc 源查询并用 v1.30.0 重新生成。通知和日志依据已提交批次，失败上抛；服务/任务取消同步检查，迟到解析不开始写入或发布通知。

交叉审查与独立真实 SQLite 验证发现并修复三处边界：Time 的内部表示不能用于判定元数据变化；旧 COALESCE 查询无法清空可选值，会反复更新/通知；AfterFunc 的异步取消不能代替写入前同步检查 owner context。另补安装期间根路径重新出现的删除竞态，避免 ID/usage 丢失。

实际验证：

| 命令/范围 | 结果 |
| --- | --- |
| `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate` | 从源查询生成应用 metadata 更新；排除无关 plugin 输出的函数排序变化，无手改生成内容 |
| `go run ./cmd/watoolsctl check` | 最终通过：80 项前端回归、类型检查/build、Go 全量测试、server-tag app/api、Go vet、production 编译 |
| `go test -race ./pkg/db ./internal/command/application ./internal/command ./internal/command/watcher ./internal/host` | 最终五包通过；独立故障验证新增 20 个 command 行为用例，保留原有测试 |
| command 的取消/交错/路径恢复选择用例 `-race -count=20` | 通过；watcher 跨路径组合事件额外 `-race -count=5` 通过 |
| Windows arm64、`CGO_ENABLED=0` 的上述五包 `go test -c` | 交叉编译通过；Windows 解析测试包括读取/损坏/可选降级与部分成功，未执行 Windows 运行时 |
| 交叉复审、gofmt、文档链接与 `git diff --check` | 通过；最终无待处理逻辑阻断 |

测试使用一次性临时应用数据/缓存，复用仓库 `.cache/tools` 工具缓存，并设置 macOS 12 flags。统一检查的前几次失败定位到 SQLite 可选值未清空及生成后残留 import；修复后完整重跑通过。既有 objc/deployment target 链接警告仍在，退出码为 0。初次直接启动检查遇系统默认 Go cache 的沙箱权限错误，改用仓库工具缓存后正常执行，没有访问真实数据或凭据。

| 实施提交 | 完成范围 | 验收边界 |
| --- | --- | --- |
| `0242cb8` | E05/E09 四类解析错误、部分扫描与保留规则；周期/watcher/usage 共用协调；删除根复核；配置实际生效；metadata 完整替换与同步取消 | 真实 SQLite、读取/损坏/可选降级、批次故障、重复事件、交错、usage、取消与恢复根用例；统一检查、五包 race、Windows 编译通过 |

未执行原生 GUI、真实安装/替换、Windows 实机、公开发布或推送；平台元数据子进程仍没有新增加的终止/超时机制，只保证取消后的迟到结果失效。整体目标仍为 in-progress。

### 第四轮：E04 热键恢复、E07 必需恢复与 E10 停止

从 `f6af1fe` 开始，工作区干净。用户授权新建 3 个 subagent（均 `fork_turns="none"`）与本地提交：hotkey_recovery 独占 app 热键更新、manager/native 与专属测试；workflow_shutdown 独占 workflow 权限恢复/清理、localinstall manager/process 与专属测试；fault_validation 仅新增独立故障/生命周期测试并交叉审查。主 agent 负责 Host/lifecycle、安全消息、手写契约/mock、统一生成、文档与集成；跨范围先协调，subagent 不提交。

本轮实施：

- E04：热键更新和查询继续串行，先进入 macOS 原生调度再取生命周期锁。分别恢复旧配置与更新前实际注册集合，两步都尝试；`HotkeyUpdateError` 经 `errors.Join` 保留主失败与全部恢复失败。查询分别返回配置和捕获的实际注册值；失败后的配置与注册可不一致，但不伪造恢复成功。Carbon 注册/注销的部分清理失败保留 listener、key/handler 与 callback handle 所有权。原子保存合并 Write/Close，失败不替换旧磁盘配置。
- Host 暴露状态查询/现有配置能力，安全消息区分完整恢复与恢复失败，无新增设置 UI。同步业务契约、mock 并从 Go 源重新生成 Wails 绑定；Shutdown 返回真实停止故障，生命周期保留 cause 并记录诊断。
- E07：生成绑定的 WalkDir 与每个 Chmod 失败均影响结果，并与生成主失败合并。必需命令输出、关闭与准备失败停止后续副作用；短写与子进程非零退出可同时保链。无害 bundle/临时目录清理失败记录残留路径，不反转已提交安装；影响后续步骤的准备失败仍返回。
- E10：bootstrap 有界停止任务组；可信安装 CLI 自己停止普通子任务，提交阶段始终等完成或恢复。采用独立输出管道、并发读取/等待、停止通知文件与标记；读取或停止失败使 Shutdown 可返回诊断，实际所有者保留 process/done、锁与临时资源直到真实退出及输出排空。清理后才统一发布终态，防止旧任务覆盖新任务。纯取消退出 130，不打印错误；helper 已成功提交而输出诊断失败退出 131，管理器仍报告实际安装成功和重启需求并保留故障。bootstrap 的迟到零退出不开始安装，非零提交失败不产生成功标记。

独立新增三个故障文件共 26 个顶层用例，另保留全部既有测试并补专属热键/提交/CLI/Host 用例。覆盖配置与注册恢复组合、残余原生资源、Write/Close/短写、遍历/权限、尽力清理、输出读取/排空/关闭/停止/退出组合、取消与迟到结果、源码锁与未退出资源保持、清理期间拒绝新任务及真实提交事实。交叉审查发现并修复 Wait 提前关 stdout、停止结果尚未完成就清理、旧 reader 未排空、终态提前发布覆盖新任务、并发写输出、非零退出吞写出 cause，以及已提交结果误分类问题。

| 命令/范围 | 结果 |
| --- | --- |
| `go run ./cmd/watoolsctl bindings` | 从 Go 源重新生成并构建通过；新增状态/更新 DTO，无手改输出 |
| `go run ./cmd/watoolsctl check` | 最终通过：81 项前端回归、类型检查/build、Go 全量测试、server-tag app/api、Go vet、production 编译 |
| `go test -race ./cmd/watoolsctl ./internal/app ./internal/host ./internal/localinstall ./internal/workflow` | 最后短写与停止通知消费补丁后最终五包通过 |
| 26 个独立故障用例 `-race`；关键取消/提交事实/输出/锁/清理交错 `-race -count=10` | 通过；专属 atomic 边界/短写系列另重复 10 次，正常取消握手重复 5 次通过 |
| Windows arm64、`CGO_ENABLED=0` 的上述五包 `go test -c` | 最终交叉编译通过，仅编译，未运行 Windows 实机 |
| 三方交叉复审、gofmt、文档链接与 `git diff --check` | 通过；无剩余本轮阻断项 |

检查采用一次性临时应用数据/缓存，复用仓库 `.cache/tools`，设置 macOS 12 flags。首次新增 Host 方法的前端契约检查在生成前失败，重新生成后通过；独立 race 曾捕获停止通知与子进程输出写入竞争，修复后复验通过。最终统一检查和 race 仍有既有重复 objc/deployment target 链接警告，退出码均为 0。

本轮不实施 E08、SDK、性能、新热键设置 UI 或应用解析子进程的通用超时；不重做前三轮事务补偿、Host/锁分类与扫描协调。不推送、不发布、不安装到正式环境。原生 GUI、物理热键、真实安装替换与 Windows 实机仍未验收，整体目标仍为 in-progress。Windows 沿用的热键库吞掉原生注销结果，状态准确确认有限；可信安装 CLI 暂缺 Windows 优雅取消通道，取消明确报停止失败并保留运行状态，避免强杀可能正在提交的安装。这两项没有因交叉编译通过而标为已验收。

实现与文档由主 agent 交叉审查后统一本地提交为 `7faa2fa`。

## 2026-10-09 阶段结案

结案范围为前四轮已授权的可靠性修复与必要规则收敛，代码基线为 `7faa2fa`。核对时工作区干净，全部实现已本地提交；本次仅整理结案与范围记录，不新增生产功能、不推进后续批次。该阶段的代码实施与自动化验证已完成；整体架构目标和 macOS 自用验收继续保持 in-progress。

| 已交付范围 | 主要实现提交 | 结案依据 |
| --- | --- | --- |
| 安装/DB 补偿、Host/锁错误分类、watcher 安全重试、日志异步与管理路由 | `5aecf01`、`ac70267`、`dd70fa1`、`361be1d`、`d516f66` | 主失败与恢复失败保链、事务结果不明禁止重试、安全反馈、旧响应失效、路由回归 |
| E01 查询错误全链透传与前端旧数据保持 | `f465bad`、`952d788` | DB → command → Host → UI/mock 故障传播、查询/usage/刷新交错 |
| E05/E09 解析、提交统计、路径协调与 watcher 配置 | `0242cb8` | 暂时故障保留旧记录、通知依据实际提交、统一协调与配置生效 |
| E04 热键恢复、E07 必需恢复/清理、E10 停止与提交事实 | `7faa2fa` | 配置/注册可核对、恢复全 cause、停止/排空/取消/迟到结果与提交保护 |

最近一次完整自动化验证在 2026-10-07、最终代码补丁之后完成：81 项前端回归、类型检查/build、Go 全量测试、server-tag app/api、Go vet、production 编译及第四轮相关五包 race 全部通过；前三轮对应包 race 与故障验证见各轮记录。测试使用临时应用数据/缓存及仓库工具缓存。Windows arm64 交叉编译仅作为历史编译结果保留。本次结案核对提交与既有检查记录，并检查文档链接和 diff；没有重新执行运行期测试，不能把历史通过记录表述为 2026-10-09 新验收。

| 剩余事项 | 状态与后续边界 |
| --- | --- |
| 本轮修复涉及的 macOS 原生 GUI、物理热键、真实安装替换/恢复、真实 React 挂载 | 待验收；四轮自动化测试不代替这些场景，既有其他功能的原生证据不移作本轮验收 |
| E08 显式初始化 | planned，未实施；另批明确范围与故障验收 |
| 版本/catalog/解压/usage、store/页面/iframe 生命周期、SDK 与性能工作 | 保持对应 planned/backlog 状态，本次结案不自动授权 |
| Windows、公开分发与发布 | deferred，等待明确需求与授权；没有 Windows 实机验证，也没有公开发布 |

Windows 范围已明确收窄：第三轮同步改过 Windows 扫描行为与测试，第四轮调整过共享停止接口并禁止强杀安装；这些是已提交的历史事实，不能概括成“只做交叉编译”，也不代表启动了完整 Windows 功能补全。后续实施以 macOS 为范围，Windows 仅做共享接口变化必需的编译适配；Windows 行为修复、新功能、专属测试或实机验收须另行明确授权。现有 Windows 优雅取消与热键注销确认限制继续登记为延期事项，不回滚已提交成果，也不将它们作为本阶段结案的完成条件。

本阶段交付保留 Go/Wails/React、七领域 Host、SQLite、`.wt` 与现有数据策略。下一阶段的候选顺序是先验收本轮 macOS 原生行为，再单独选择 E08 或其他 backlog；本次停止在阶段结案，不继续实施、不推送、不发布、不安装到正式环境。
