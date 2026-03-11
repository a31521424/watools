# 插件开发 03: 运行时与上下文注入

## PluginContext

`match` 和 `execute` 函数接收同一份 `PluginContext`:

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

示例:

```javascript
match: (context) => {
    if (context.input.value.startsWith('calc')) return true;
    if (context.clipboard?.contentType === 'image') return true;
    return false;
}
```

## UI 插件上下文注入

WaTools 对 UI 插件和 Executable 插件使用同一份 `PluginContext` 数据模型。

- `match(context)` / `execute(context)` 直接接收 `PluginContext`
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
