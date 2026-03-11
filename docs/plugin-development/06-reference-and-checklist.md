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
- `StorageGet/Set/Remove/Clear/Keys()`
- `OpenFolder(path)`
- `SaveBase64Image(base64): Promise<path>`
- `CopyBase64ImageToClipboard(base64): Promise<void>`

## 完整类型

```typescript
type PluginEntry = {
    type: "executable" | "ui"
    subTitle: string
    match: (context: PluginContext) => boolean
    execute?: (context: PluginContext) => Promise<void>
    icon: string | null
    file?: string
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

### 文件结构

- [ ] `manifest.json` 在根目录
- [ ] `app.js` 在根目录
- [ ] `index.html` 在根目录
- [ ] 构建模式输出的是 `dist/` 内容,不是工程源码

### 配置

- [ ] `manifest.json` 包含必需字段
- [ ] `packageId` 格式为 `watools.plugin.xxx`
- [ ] `app.js` 正确导出 `export default entry`
- [ ] `match` 同步返回 boolean

### 构建模式

- [ ] `vite.config.ts` 配置 `publicDir: 'public'`
- [ ] `package.json` 包含 `build` 和 `package`
- [ ] `manifest.json` 和 `app.js` 放在 `public/`

### API 使用

- [ ] 使用 API 包装
- [ ] HTTP 请求使用 `window.watools.HttpProxy`
- [ ] 存储使用 `window.watools.StorageXxx`
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
