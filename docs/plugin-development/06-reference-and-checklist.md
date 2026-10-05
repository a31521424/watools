# 插件开发 06: 参考与检查清单

## API 附录

### `window.runtime`

- `ClipboardGetText(): Promise<string>`
- `ClipboardSetText(text: string): Promise<boolean>`
- `Hide() / Show() / Quit()`
- `WindowCenter() / WindowMaximise() / WindowMinimise()`
- `WindowSetSize(w, h) / WindowGetSize()`
- `LogInfo(msg) / LogError(msg) / LogDebug(msg)`
- `BrowserOpenURL(url: string)`
- `Environment(): Promise<{platform, arch, buildType}>`

### `window.watools`

- `HttpProxy(request): Promise<response>`
- `CancelHttpProxy(requestId): Promise<void>`
- `SecretGet/Set/Remove()`：插件凭据存储（macOS 本地明文文件，Windows DPAPI）
- `StorageGet/Set/Remove/Clear/Keys()`
- `OpenFolder(path)`
- `SaveBase64Image(base64): Promise<path>`
- `CopyBase64ImageToClipboard(base64): Promise<void>`

## 完整类型

```typescript
type PluginEntryDefinition = {
    subTitle: string
    matchPreview?: (context: PluginMatchContext) => boolean
    match?: (context: PluginContext) => boolean // 旧入口兼容，仅用于 <=2000 单元文本
    icon: string | null
    largeTextPrefixes?: string[] // 超长文本 executable 的有界前缀规则
} & (
    {type: "ui"; file: string}
    | {type: "executable"; execute: (context: PluginContext, api?: WaToolsApi) => Promise<void>}
)

type PluginMatchContext = {
    readonly input: {
        readonly valueType: "text" | "clipboard"
        readonly clipboardContentType?: "text" | "image" | "files"
        readonly length: number
        readonly head: string // 最多 128 UTF-16 单元，原始前缀
        readonly tail: string // 最多 128 UTF-16 单元，原始后缀
        readonly truncated: boolean // 原文长度 >128
    }
    readonly clipboard: {readonly contentType: "text" | "image" | "files"; readonly fileCount: number} | null
}

type PluginContext = {
    input: AppInput
    clipboard: AppClipboardContent | null
}

type AppInput = {
    valueType: "text" | "clipboard"
    value: string
    clipboardContentType?: "text" | "image" | "files"
}

type AppClipboardContent = {
    contentType: "text" | "image" | "files"
    text: string | null
    imageBase64: string | null
    files: string[] | null
}

declare global {
    interface Window {
        runtime?: any
        watools?: any
        inputValue?: string
        pluginContext?: PluginContext
    }
}
```

## 最终检查清单

插件导出的是上述声明类型；内部 `PluginEntry` 含规范化 matcher，不属于插件参数协议。旧兼容仅存在于宿主边界，新入口无需同时写两份 matcher。

### 文件结构

- [ ] `manifest.json` 在根目录
- [ ] `app.js` 在根目录
- [ ] `index.html` 在根目录
- [ ] 构建模式输出的是 `dist/` 内容,不是工程源码

### 配置

- [ ] `manifest.json` 包含必需字段
- [ ] `packageId` 格式为 `watools.plugin.xxx`
- [ ] `app.js` 正确导出 `export default entry`
- [ ] 至少提供 `matchPreview` 或兼容 `match`，同步返回 boolean；新入口优先用有界的 `matchPreview`

### 构建模式

- [ ] `vite.config.ts` 配置 `publicDir: 'public'`
- [ ] `package.json` 包含 `build` 和 `package`
- [ ] `manifest.json` 和 `app.js` 放在 `public/`

### API 使用

- [ ] 使用 API 包装
- [ ] HTTP 请求使用 `window.watools.HttpProxy`
- [ ] 普通偏好使用 `window.watools.StorageXxx`，密钥使用 `SecretXxx`，遵循宿主的平台存储策略，不额外复制到 localStorage
- [ ] UI 插件通过 `window.pluginContext` 读取上下文
- [ ] UI 插件监听 `watools:context-ready`
- [ ] 不使用 `alert` / `confirm` / `prompt`
- [ ] 不使用 `window.open()`

### UI 设计

- [ ] Modal/Dialog 不依赖 fixed 覆盖整个窗口
- [ ] Dropdown/Popover 有 iframe 边界定位逻辑
- [ ] Toast 考虑 `max-width`
- [ ] 所有覆盖层在 iframe 内可正常显示

### 快捷键

- [ ] 同时监听 `e.ctrlKey || e.metaKey`
- [ ] 快捷键提示根据平台显示不同符号
- [ ] 避免与主应用冲突
- [ ] 提供统一快捷键处理函数或 Hook

### 打包验证

- [ ] `.wt` 文件内容在根级别
- [ ] 解压后打开 `index.html` 能在浏览器运行
- [ ] 文件总大小合理
