# 剪贴板呼出、插件启动与独立小窗证据

[返回证据索引](../README.md) · [调研结论](../../../research/clipboard-launch-performance-2026-10-06.md)

源码基线为 `2c6a229`，采样日期为 2026-10-06。基线调研和实施后验收分开记录；当前机制见 [实现说明](../../../architecture/implementation-mechanism.md#剪贴板原文与呼出性能)。

## 主窗口展开与 Command+D 分离

### 分离默认隐藏与单窗口唤出补充

本轮修正分离成功后仍显示空主窗，以及应用级激活将独立小窗一起置前的问题。启动器改用 Wails Beta.27 的 non-activating NSPanel，Show／Focus 仅作用于启动器；隐藏由 AppKit 恢复键盘焦点，不再保存和强制激活前一个应用。小窗恢复后，源页面先隐藏主窗再复位空启动器；失败保留源页面。

- [`plugin-window-focus-checks.txt`](./plugin-window-focus-checks.txt)：完整检查和签名构建摘要，57 项前端、16 项流程回归通过。
- [`plugin-window-focus-native-harness.py.txt`](./plugin-window-focus-native-harness.py.txt)、[`plugin-window-focus-native-results.jsonl`](./plugin-window-focus-native-results.jsonl)：两个 JSON 实例（一个含百万字符），分离后主窗不在原生屏幕窗口列表中，小窗编辑值与原文逐字符一致，WebView 保有焦点；下次唤出为 64px 空启动器。
- [`plugin-window-focus-status.swift.txt`](./plugin-window-focus-status.swift.txt)、[`plugin-window-focus-helper.m.txt`](./plugin-window-focus-helper.m.txt)：仅记录 QA 和外部参考进程的 CoreGraphics 窗口 ID／顺序与活跃进程。连续三轮外部参考窗覆盖小窗后，调用真实呼出 API，只出现启动器在其上方；小窗保持在参考窗之后且相对顺序不变，外部应用仍为活跃应用，启动器 WebView 获得键盘焦点。隐藏启动器后小窗仍在屏幕窗口列表中。

脚本先确认 iframe 宿主上下文和分离按钮就绪，再发送合成 Command+D。JSON 推荐由原生 Focus 自动读取合成剪贴板产生。已恢复原剪贴板并退出全部 QA 进程。这里验证 AppKit／WebKit 焦点与原生层级；物理键盘、真实中文输入法、多屏仍未验收。下文较早阶段的“返回空启动器”记录保留其历史行为，以本轮“自动隐藏”为当前实现。

### 官方插件 0.2.0 与旧包升级补充

用户反馈分离报错后，只读检查确认正式与开发安装的 JSON 0.1.0 均无状态 hooks；源码接口已改，但宿主构建不会更新插件安装。这一轮补齐五个官方 UI 插件 hooks 并升为 0.2.0，UI SDK 在 context-ready 前提供只读 session 元数据，区分首次导入和分离恢复。JSON 恢复直接设置新模型并拒绝迟到的原文覆盖；二维码恢复现有图片，不重生成；翻译保留译文、暂停未完成请求，不序列化密钥，不重复自动翻译；计算器恢复不写共享历史／草稿，文本统计保存选区和滚动位置。

- [`plugin-upgrade-checks.txt`](./plugin-upgrade-checks.txt)：57 项前端回归、完整检查、应用构建、打包和两个安装目录的升级摘要。
- [`plugin-upgrade-native-harness.py.txt`](./plugin-upgrade-native-harness.py.txt)、[`plugin-upgrade-native-results.jsonl`](./plugin-upgrade-native-results.jsonl)：先构造无 hooks 的旧 JSON 包验证报错与原页面保留，再原位升级并验证启用状态、使用次数和合成存储不变；最后验收百万字 JSON 多开、面板状态及其余四个 UI 插件的迁移。

本轮原生翻译场景没有真实密钥或外部服务调用；带合成密钥的重复请求、初始化去重与无密钥序列化由执行页面源码的回归验证。所有 native key events 为合成事件；物理输入法与多屏体验仍在待验范围。正式和开发目录已通过标准安装器升级，不读取用户插件存储／凭据来做测试；验证升级的数据保留使用独立 QA 数据。启动新应用前应先保存旧页面，因为更新后旧 iframe 仍保留其原 revision。

复现使用下方构建的 QA bundle 和两个 Swift 辅助程序；重新打包插件后串行运行：

```sh
node scripts/dev.mjs plugins package
python3 docs/reviews/evidence/2026-10-06/plugin-upgrade-native-harness.py.txt > /private/tmp/watools-plugin-upgrade-native-results.jsonl
```

### 首轮窗口实现验收

这一阶段基于剪贴板优化提交 `d5b07b2` 继续实施。默认在主窗口展开 UI 插件，只有 Command+D 或分离按钮创建独立 AppKit 窗口；新窗领取宿主保存的原始上下文和编辑状态，恢复完成才显示，随后主窗口返回空启动器（最小高度 64）。同插件每次分离创建独立实例，URL 只含会话 ID。待加载超时与 ready 在同一锁内裁决，迟到的超时不会关闭已恢复的窗口；原生操作在锁外执行。

- [`plugin-window-checks.txt`](./plugin-window-checks.txt)：最终完整检查、Go race、Windows 交叉构建和插件打包摘要。
- [`plugin-window-native-harness.py.txt`](./plugin-window-native-harness.py.txt)：通过真实呼出 API 激活 QA 启动器，再用 Wails MCP 合成键盘事件验收；使用隔离目录和合成剪贴板。
- [`plugin-window-status.swift.txt`](./plugin-window-status.swift.txt)：只读取 QA 进程的 CoreGraphics 窗口顺序和应用隐藏状态；Wails 的 macOS `visible` 为未遮挡状态，不能用来判断重叠窗口是否隐藏。
- [`plugin-window-native-results.jsonl`](./plugin-window-native-results.jsonl)：最终 AppKit／WebKit 实际断言，仅保存长度和布尔结果。

最终 51 项前端、16 项工作流回归、完整 check、Go race 和 Windows amd64 production 交叉编译通过。原生运行验证了两个 JSON 实例、百万字符编辑文本及原文／路径／输出逐字一致、事件隔离、独立尺寸、主窗复位与隐藏、单实例关闭及禁用清理；文本统计插件的普通表单编辑迁移和 Esc 单窗关闭也通过。剪贴板恢复，宿主和辅助进程退出。原生窗口列表在两个实例创建后分别有 2／3 个屏幕窗口，隐藏主窗后剩 2 个；窗口服务器异步更新需等待稳定，不能用即时读数或遮挡状态判失败。

在本页下方完成 QA bundle 构建与剪贴板辅助程序编译后，串行执行：

```sh
cp docs/reviews/evidence/2026-10-06/plugin-window-status.swift.txt /private/tmp/watools-qa-window-status.swift
swiftc -module-cache-path /private/tmp/watools-swift-module-cache /private/tmp/watools-qa-window-status.swift -o /private/tmp/watools-qa-window-status
python3 docs/reviews/evidence/2026-10-06/plugin-window-native-harness.py.txt > /private/tmp/watools-plugin-window-native-results.jsonl
```

验证边界：快捷键由脚本合成，物理 Command+D、中文输入法、多屏／缩放／置顶、Windows GUI 尚未验收。JSON 提供专用 hooks；普通表单恢复输入并派发 input／change，其他自定义模型和运行中任务须由插件导出。重载恢复分离时快照，不承诺保留分离后的编辑或 undo 栈。共享历史的并发合并、旧 revision 回收及不可信插件权限不属于本次已完成交互，目标仍为 in-progress，见 [小窗 backlog](../../../backlog/plugin-windows-and-sessions.md)。

## 实施后验收

本次工作区实现已通过源码检查和 macOS 原生 GUI 验证。测试文本均为合成素材，包含首尾空白、换行、Emoji 和 NUL。应用使用独立临时数据／缓存目录和 QA bundle ID，安装本次重打包的六个官方插件。原生测试临时替换系统剪贴板，原内容仅在辅助进程内存中备份；最后已恢复，宿主与辅助进程均已退出。凭据和原始用户剪贴板未写入证据。

- [`clipboard-checks.txt`](./clipboard-checks.txt)：完整检查、race 与 Windows 交叉构建的验收摘要。
- [`clipboard-native-harness.py.txt`](./clipboard-native-harness.py.txt)：通过 Wails MCP 操作真实 AppKit／WebKit、隔离数据与记录断言的脚本。
- [`clipboard-helper.swift.txt`](./clipboard-helper.swift.txt)：NSPasteboard 合成素材、内存备份和有条件恢复的辅助程序。
- [`clipboard-native-results.jsonl`](./clipboard-native-results.jsonl)：初始优化阶段的原生实际输出；只保存长度、耗时和断言结果。
- [`json-native-harness.py.txt`](./json-native-harness.py.txt)、[`json-native-results.jsonl`](./json-native-results.jsonl)：补充的长 JSON 入口、Monaco 原文导入与格式化原生验收。
- [`match-context-native-harness.py.txt`](./match-context-native-harness.py.txt)、[`match-context-native-results.jsonl`](./match-context-native-results.jsonl)：轻量匹配契约阶段的原生验收。

### 当前类型与边界重构验收

按 KISS 进一步收敛：启动器只有 `LauncherInput` 的 inline／clipboard-snapshot 两种状态；展示、搜索和轻量匹配均派生自此状态。加载时把声明归一成一个 matcher，旧 matcher 只在独立适配器中处理；`resolvePluginContext` 是唯一原文解析边界，不再构造假的空正文上下文。六个官方插件只声明 `matchPreview`，新版包需要当前宿主。异步原文读取、排队执行、统计 flush、Escape 和 iframe ready 均检查捕获的素材版本，防止取消后执行或清空新输入。三个子 agent 分别审查设计、匹配边界和回归测试，最终只读复核未发现阻断项。

最终 45 项前端回归、16 项开发流程回归、完整 `check`、Go race、Windows production 交叉构建通过；绑定由 Go 重新生成，七个服务／42 个方法。六个官方插件已重打包。收尾发现并修复 Wails 默认 macOS 文本写入在 NUL 处截断的问题，写入统一走长度感知的 `SetClipboardTextApi`，mock 与旧 SDK 包装同步更新。

- [`launcher-refactor-native-results.jsonl`](./launcher-refactor-native-results.jsonl)：当前最终全流程原生结果。2,008／20,008／100,008／1,000,008／10,000,008 单元呼出首帧分别为 29／17／19／15／50 ms；缓存读取 0–1 ms，输入控件正文为 0。百万单元原文交付及实际文本统计编辑器逐字符一致，加载约 807 ms；手动百万字符粘贴首帧 8 ms。20,008 单元含 NUL、Emoji 和首尾空白的复制前将系统剪贴板换为其他内容，复制原文逐字符一致。计算器无效表达式实际触发后错误可见、输入保留。
- [`launcher-refactor-json-results.jsonl`](./launcher-refactor-json-results.jsonl)：110 万字符普通文本不命中 JSON；1,100,033 单元 JSON 自动推荐，首帧 22 ms。桥接 head／tail 各 128 单元，inline 原文为 0；实际 Monaco 导入约 287 ms，两个上下文字段及编辑器原文一致。格式化约 145 ms，正文与 Unicode 字段值保持一致。

提交前全工作区复审发现并修复：原生读取持缓存锁等待主线程、退出时反向等待缓存锁的潜在死锁（改为独立读锁，关闭后禁止回写）；计算器轻量候选 `2+`／`1/0` 执行错误静默返回导致清空输入（改为抛错并保留输入）。同时补齐失焦 flush 期间异步新素材的 revision 保护、Windows 自动文本读取的明确 UTF-8 输出。新增回归覆盖阻塞读取不阻止关闭、并发过期后的版本复用、计算器失败无复制／历史副作用及宿主保留素材、失焦新素材不被旧任务隐藏。三份分工复审的发现均已处理；45 项前端、16 项开发流程回归及完整构建、Go race、Windows 交叉构建重新通过。

两轮均恢复剪贴板并退出宿主。以上是合成事件的一次采样，非统计性能上限；物理全局快捷键、Windows GUI 和像素级视觉验收未覆盖。当前脚本包含 NUL 复制断言，只返回布尔结果，避免将正文写入证据。下文各阶段结果作为历史记录保留。

### 匹配与触发契约阶段

补齐 `PluginMatchContext` 后，六个官方插件均优先走 `matchPreview`。匹配对象被冻结，只包含首尾各最多 128 单元、完整长度、来源／素材类型及文件数量；真实触发交付完整 `PluginContext.input.value` 与文本 `clipboard.text`。匹配不含正文、图片 base64、文件路径、快照 ID 或宿主 API。Go 新增独立 suffix，生成绑定重新生成，手动输入也从原文的有界首尾生成相同契约。

最终源码回归 31 项通过，覆盖轻量对象没有原文字段、不会回退旧 matcher、没有 origin 读取、实际触发才取固定原文；官方 JSON 不解析全文、计算器不求值；仅新 callback 的入口通过加载校验。完整 `check`、`check-race`、官方插件重打包及 Windows 交叉编译通过。

最终原生运行：110 万字符普通文本没有 JSON 入口；1,100,033 单元 JSON 命中入口，呼出首帧 29 ms。桥接 head/tail 均 128 单元、inline text 为 0。选中后实际 Monaco 编辑器及两个上下文字段逐字符一致，导入约 309 ms，格式化约 136 ms且字段值保持一致。原剪贴板已恢复，进程已退出。数据为单次采样、合成事件，边界同下文。

首尾括号只判断候选，允许无效 JSON 命中；JSON 标量或首尾空白超过采样窗口可能无法自动推荐。输入新查询仍替换素材，独立 query 尚未实现。可信 matcher 可以自行访问全局环境或阻塞执行；当前数据契约限制原文传递，未实现 Worker 强制终止。

### 初始优化与 JSON 导入补充

| 合成原文长度（UTF-16 单元） | focus 到 DOM 更新后的首帧 |
| --- | --- |
| 2,008 | 20 ms |
| 20,008 | 13 ms |
| 100,008 | 17 ms |
| 1,000,008 | 9 ms |
| 10,000,008 | 44 ms |

以上均为同一轮单次采样，无预热或统计分位数。长文本 input 的 value 长度均为 0，摘要为 71 单元；相同版本缓存查询为 0–1 ms。100 万单元文本启动文本统计插件后，`pluginContext.input.value`、`clipboard.text` 和插件实际编辑器逐字符一致，完整业务加载约 802 ms，路由 URL 为 121 字符。系统剪贴板改变后，原快照仍逐字符一致。100 万字符的手动粘贴显示约 6 ms，“复制原文”也逐字符一致。

用户追问 JSON 场景后，补充检查发现 JSON 页面原来忽略无命令前缀的原始 JSON。已修复为优先导入剪贴板正文，也支持对象、数组、标量及类似命令的字面文本。追加回归后前端共 28 项通过。重新打包 JSON 插件，使用原生 QA 宿主与独立安装验证 1,100,033 单元合成 JSON：“打开 JSON 编辑器”入口可见，呼出 21 ms，Monaco 导入 284 ms，两个上下文字段和编辑器内容逐字符一致；点击格式化约 121 ms，结果成功、正文与 Unicode 字段保持一致。剪贴板恢复成功，进程退出。这些耗时同样只是一次采样。

上述两轮结果记录初始阶段：当时长文本统一展示 UI，尚未自动识别 JSON；最终实现改为上文的有界候选匹配。历史输出保留，不作为最终匹配结果。旧全流程脚本已把至少五个入口的断言改为至少一个，以适配候选过滤，其余原文验收逻辑保留。排序仍使用历史权重，不保证 JSON 置顶；输入新查询仍替换素材。

这里实际经过 NSPasteboard、Wails 传输、React 渲染和插件 iframe；focus、粘贴、点击及 Escape 由脚本事件触发。未测试物理全局快捷键、真实输入法、Windows GUI，以及其他应用延迟提供剪贴板数据的最坏情况。窗口截图未成功，未做像素级视觉验收。性能数据不能解释为任意剪贴板提供者的恒定上限。

原生脚本的复现步骤（macOS，本地需要 GUI 和 Xcode 命令行工具）：

```sh
node scripts/dev.mjs check
node scripts/dev.mjs plugins package
mkdir -p .cache/clipboard-native
python3 - <<'PY'
import pathlib, plistlib, shutil
source = pathlib.Path('build/bin/watools.app')
target = pathlib.Path('.cache/clipboard-native/watools-qa.app')
shutil.copytree(source, target, dirs_exist_ok=True)
path = target / 'Contents/Info.plist'
info = plistlib.loads(path.read_bytes(), fmt=plistlib.FMT_XML)
info.update(CFBundleIdentifier='com.wails.watools.clipboard-qa', CFBundleName='WaTools Clipboard QA', CFBundleDisplayName='WaTools Clipboard QA')
path.write_bytes(plistlib.dumps(info))
PY
export GOTOOLCHAIN=go1.26.8
export GOCACHE="$PWD/.cache/tools/go-build"
export GOMODCACHE="$PWD/.cache/tools/go-mod"
export GOPATH="$PWD/.cache/tools/gopath"
export MACOSX_DEPLOYMENT_TARGET=12.0
export CGO_CFLAGS=-mmacosx-version-min=12.0
export CGO_LDFLAGS=-mmacosx-version-min=12.0
go build -tags production,private_mac_apis,mcp -o .cache/clipboard-native/watools-qa.app/Contents/MacOS/watools .
codesign --force --deep --sign - .cache/clipboard-native/watools-qa.app
cp docs/reviews/evidence/2026-10-06/clipboard-helper.swift.txt /private/tmp/watools-clipboard-helper.swift
swiftc -module-cache-path /private/tmp/watools-swift-module-cache /private/tmp/watools-clipboard-helper.swift -o /private/tmp/watools-clipboard-helper
python3 docs/reviews/evidence/2026-10-06/clipboard-native-harness.py.txt > /private/tmp/watools-clipboard-native-results.jsonl
# 如需单独复现长 JSON：与上一条串行执行，共用相同宿主与辅助程序。
python3 docs/reviews/evidence/2026-10-06/json-native-harness.py.txt > /private/tmp/watools-json-native-results.jsonl
# 最终轻量匹配与完整触发验收（串行执行）：
python3 docs/reviews/evidence/2026-10-06/match-context-native-harness.py.txt > /private/tmp/watools-match-context-native-results.jsonl
```

MCP 端口使用 9267，脚本会启动并停止 QA 宿主；不要与其他占用该端口的测试同时运行。脚本不自动删除临时目录，便于失败排查。复现会临时替换剪贴板，请保持测试期间不复制其他内容；若检测到外部改变，辅助程序会拒绝覆盖新的内容。

## 历史基线调研

以下输出只适用于 `2c6a229` 源码。仅合成文本和应用列表，不读取或修改真实剪贴板、用户数据或凭据；不是优化后性能结果。

- [`clipboard-launch-probe.cjs`](./clipboard-launch-probe.cjs)：读取指定基线源码，测量 Fuse 查询解析／搜索、store、排名、启动快照和六个官方插件 matcher。每项在独立 Node 子进程执行，5 秒总超时；计时不包含模块加载、转译和构造数据。
- [`clipboard-launch-results.json`](./clipboard-launch-results.json)：实际结果、环境和状态语义探针。关闭扩展查询的补充实验在基础实验完成后单独运行，纳入同一结果文件。

在本次工作区根目录运行，以 `WATOOLS_PROBE_ROOT` 指向独立的 `2c6a229` 基线检出（需已安装 frontend 依赖）；该探针不适配优化后的模块依赖：

```sh
WATOOLS_PROBE_ROOT=/absolute/path/to/baseline node docs/reviews/evidence/2026-10-06/clipboard-launch-probe.cjs > /private/tmp/watools-clipboard-results.json
WATOOLS_PROBE_ROOT=/absolute/path/to/baseline node docs/reviews/evidence/2026-10-06/clipboard-launch-probe.cjs --case application-no-extended words 100000
```

也可用 `WATOOLS_PROBE_ROOT` 指定基线检出的路径，探针文件保留在本次工作区。完整探针的标准错误输出每个已完成条目，标准输出为 JSON 汇总。基线回归结果为 18 项通过、0 失败。探针为一次采样，无预热或统计分位数；JSON 形状使用包含引号、空格和换行的片段，不保证是完整合法 JSON。128 字符查询与浅层快照在基线时仅为实验对照，本次已实施。

基线调研未进行原生验收；本次实施后原生结果及其边界见本页前半部分。
