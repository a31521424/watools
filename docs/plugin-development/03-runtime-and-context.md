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

`execute(context, api)` 和 UI 的 `pluginContext` 始终收到完整原文；宿主自动呼出时内部使用摘要与 origin 快照引用，选择插件后才取原文，内部引用不会交给插件。启动 URL 只带 launchId，不放正文。宿主清空启动器、剪贴板被再次复制或 iframe 重载，不会改变该次启动的内容。

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
- ESC 由主应用自动处理
- 插件应通过 `window.watools` 使用宿主能力
- 调试时可能没有完整宿主 API,需要提供浏览器降级
