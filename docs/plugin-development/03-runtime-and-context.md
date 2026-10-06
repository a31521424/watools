# 插件开发 03: 运行时与上下文注入

## PluginContext

`execute` 和 UI 接收完整 `PluginContext`；旧 `match` 仅在有界兼容路径使用它：

```typescript
type PluginContext = {
    input: {
        valueType: "text" | "clipboard"
        value: string
        clipboardContentType?: "text" | "image" | "files"
    }
    clipboard: {
        contentType: "text" | "image" | "files"
        text: string | null
        imageBase64: string | null
        files: string[] | null
    } | null
}
```

完整执行上下文中的 `input.value` 与文本 `clipboard.text` 都是原始正文，不是摘要。`originValue` 是概念名，当前 API 沿用这两个字段，不添加重复的正文参数。

## 轻量匹配上下文

新入口使用 `matchPreview(context: PluginMatchContext)`。它只接收以下有界数据，没有 `input.value`、剪贴板正文、文件路径、图片 base64、snapshotId 或宿主 API：

```typescript
type PluginMatchContext = {
    readonly input: {
        readonly valueType: "text" | "clipboard"
        readonly clipboardContentType?: "text" | "image" | "files"
        readonly length: number
        readonly head: string
        readonly tail: string
        readonly truncated: boolean
    }
    readonly clipboard: {
        readonly contentType: "text" | "image" | "files"
        readonly fileCount: number
    } | null
}
```

`length` 是完整正文的 UTF-16 单元数；`head`、`tail` 各最多 128 单元，是原始首尾片段，保持空白与大小写。`truncated` 表示正文超过 128 单元，不可把片段当作完整输入。只对片段做 trim、大小写转换、startsWith／endsWith 或简单正则。非常长的边缘空白可能让候选无法识别；这里是启发式推荐，不是全文校验。对象与嵌套元数据均冻结，防止一个 matcher 改写其他入口收到的内容。

JSON 候选示例：

```javascript
matchPreview: (context) => {
    const head = context.input.head.trimStart();
    const tail = context.input.tail.trimEnd();
    return (head.startsWith('{') && tail.endsWith('}'))
        || (head.startsWith('[') && tail.endsWith(']'));
}
```

## 长文本与启动快照

`execute(context, api)` 和 UI 的 `pluginContext` 始终收到完整原文；宿主自动呼出时内部使用摘要与 origin 快照引用，选择插件后才取原文，内部引用不会交给插件。主窗口启动 URL 只带 launchId，独立小窗只带 pluginSession，会话正文保存在宿主内存，不放 URL。宿主清空启动器、剪贴板被再次复制或 iframe 重载，不会改变该次启动的内容。

宿主内部使用 `LauncherInput` 的内联／快照判别联合，快照没有 `PluginContext` 占位。加载时把插件声明 `PluginEntryDefinition` 归一为一个现代或旧 matcher，匹配列表只接受同一个输入素材；真正触发通过唯一 `resolvePluginContext(input)` 取得完整上下文。新插件只需实现轻量匹配和完整执行／UI，无须处理快照 id、续租或旧接口分派。

宿主优先调用 `matchPreview`，所有长度的文本都使用同一轻量契约；返回 false 后不会再调用旧 matcher 或无条件展示 UI。没有新接口的旧入口：2000 单元以内仍调用完整 `match(context)`，更长的旧 UI 入口成为显式选择、旧 executable 可用 `largeTextPrefixes`。这是兼容路径，新插件无需使用它。长文本列表不只截前三个。

```javascript
{
    type: "executable",
    subTitle: "打开 URL 或路径",
    largeTextPrefixes: ["http://", "https://", "www.", "~/", "/"],
    match: (context) => /* 原有短输入检测 */ false,
    execute: async (context, api) => { /* 使用完整 context */ },
    icon: null,
}
```

旧前缀规则最多 32 条，每条 1 至 128 单元；宿主将规则与原始前 128 单元去掉开头空白后按小写比对。新轻量接口与旧规则都是候选判断，昂贵解析应在用户实际选择后进行。完整 manifest 规则和可终止 Worker 尚未实现。可信插件仍能访问全局环境，轻量数据契约不是安全沙箱，也不能终止故意死循环的函数。

处理 JSON、字数统计、翻译或二维码中的剪贴板文本时，优先原样使用 `context.clipboard.text`。官方页面已区分剪贴板原文与用户输入的显式命令，剪贴板不再经过命令前缀剥离或 trim。只有显式键入的指令才解析其前缀。

例如复制超长 JSON 后，呼出只显示摘要；JSON 入口按 `{…}`／`[…]` 轻量特征匹配，选择后完整 JSON 进入编辑器，再通过“格式化”或 `Cmd/Ctrl + Enter` 格式化。匹配不执行 JSON.parse，格式是否有效由页面解析。六个官方插件现在仅声明 `matchPreview`，需要当前宿主；旧第三方插件由宿主适配。JSON 标量可通过 `json` 命令打开编辑器后粘贴，首尾括号规则不覆盖所有 JSON 类型。排序仍遵循使用历史，尚不保证 JSON 置顶；输入新查询仍会替换素材，尚不支持保留素材并另行搜索插件。

## UI 插件上下文注入

WaTools 对 UI 插件和 Executable 插件使用同一份 `PluginContext` 数据模型。

- `matchPreview(context)` 接收轻量 `PluginMatchContext`，`execute(context)` 接收完整 `PluginContext`
- iframe UI 插件通过 `window.pluginContext` 读取同一份 `PluginContext`
- 宿主会在 iframe 就绪后派发 `watools:context-ready`
- 新插件应把 `window.pluginContext` + `watools:context-ready` 视为唯一推荐入口
- `?seed=...` 和 `window.inputValue` 仅用于兼容旧插件或浏览器单独调试

### 推荐模式

```javascript
const readHostContext = () => window.pluginContext || {
    input: {
        value: typeof window.inputValue === "string" ? window.inputValue : "",
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
};

const applyHostContext = (context) => {
    const inputValue = context?.input?.value || "";
    // 根据 input / clipboard 更新 UI
};

applyHostContext(readHostContext());

window.addEventListener("watools:context-ready", (event) => {
    applyHostContext(event.detail || readHostContext());
});
```

### 约束

- 不要只绑定 `window.location.search` 的 `seed`
- 不要假设 `window.inputValue` 一定存在
- 文本、图片、文件三类带入数据都应从 `PluginContext` 读取
- 需要命令面板预填充时,优先读取 `context.input.value`

## 运行时环境要点

- UI 插件运行在 iframe
- 主窗口 Esc 返回启动器，独立小窗 Esc 关闭自身；输入法组合中不截获
- 主窗口 Command+D 在捕获阶段处理，优先于编辑器自带快捷键；独立窗不再分离
- runtime 的窗口方法绑定所在原生窗，独立窗 Quit 关闭自身；Hide 不隐藏其他窗口
- UI 会话的 runtime 事件订阅／发射按实例命名，关闭或 iframe 重载只移除自己的订阅
- HTTP 请求按实例与原生窗划分取消范围，窗口关闭取消自身请求；Storage／Secret 仍按 packageId 共享，不是实例私有存储
- 插件应通过 `window.watools` 使用宿主能力
- UI 的 `window.watools.session` 包含只读 `id` 与 `windowMode: "inline" | "detached"`，在 context-ready 事件之前注入；独立窗使用 restore 恢复，初始化不应重新导入原文或自动启动业务请求
- 调试时可能没有完整宿主 API,需要提供浏览器降级

## 分离窗口的状态接口

主窗口默认承载插件，不会因选择入口直接创建原生窗。Command+D 捕获当前状态，宿主创建隐藏窗，通过该窗绑定上下文领取会话，恢复完成后显示；主窗口收到 ready 后返回启动器。每次分离有独立窗口名和会话 ID，可多次打开 JSON 再分离。待加载会话 30 秒超时回收，关闭释放上下文和状态；禁用／卸载关闭该插件全部小窗，升级保留已打开的旧 revision。最多同时 32 个小窗、同一时间一个待分离会话，状态 JSON 限 64 MiB；超过限制明确报错，原页面保留。

自定义编辑器提供以下接口，返回可序列化 JSON，允许异步等待初始化；导出等待最多 30 秒，失败时保留原页面：

```javascript
window.watoolsSession = {
    capture: async () => ({text: editor.getValue(), view: editor.saveViewState()}),
    restore: async (saved) => {
        editor.setValue(saved.text);
        editor.restoreViewState(saved.view);
    },
};
```

五个官方 UI 插件从 0.2.0 起均提供专用 hooks：JSON 保存编辑文本、Monaco 视图／光标、路径输入、提取输出和面板开合；二维码保存当前图像和模式，不重新生成或解码；翻译保留原文和现有译文、暂停未完成的请求，不把密钥放入快照，也不因分离再次收费调用；计算器保存当前表达式、显示结果和历史视图，恢复不写共享草稿／历史；文本统计保存文本、选区和滚动位置。初始化和恢复明确区分，避免异步初始化覆盖已恢复内容。

未接入的普通表单插件兼容保存 input／textarea／select，并派发 input／change 重建视图；不捕获 password 和 file 输入。检测到 Monaco 或 contenteditable 却未提供 hooks 时拒绝分离，避免无声丢失编辑内容。其他自定义模型、绘图结果及运行中任务需要插件自行导出，不能搬运实时 DOM 或 Promise。独立窗重载恢复分离时快照，不承诺跨重载保留之后的编辑或 undo 栈。

宿主构建不会替换已安装插件。修改接口后必须重新打包并安装；旧版本正在运行的页面仍使用其不可变 revision，应先保存编辑内容，再返回启动器、刷新插件列表并重新打开。开发安装使用 `node scripts/dev.mjs plugins install`；正式安装使用仓库根目录的 `go run ./cmd/pluginctl install`，二者数据目录不同。见 [官方插件工具](../../plugins/README.md)。

会话接口检查 Wails 绑定上下文中的窗口名以避免跨实例误读；现有插件仍为可信模型，不能把该检查、iframe 或独立窗当作不可信代码权限沙箱。共享历史／配置的并发合并及 revision 垃圾回收仍见 backlog。
