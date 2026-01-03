# WaTools 插件开发技术规范

本文档定义 WaTools 插件系统的技术规范,供 LLM 生成符合要求的插件。

---

## 产出目标

### 插件文件结构

**基础结构**:
```
watools.plugin.{name}/
├── manifest.json    # 元数据(必需)
├── app.js          # 入口配置(必需)
└── index.html      # UI 页面(UI 插件需要)
```

**扩展结构**(可选):
```
watools.plugin.{name}/
├── manifest.json
├── app.js
├── index.html
├── module.js        # 额外模块
├── styles.css       # 样式文件
└── assets/          # 资源目录
```

### manifest.json

```json
{
  "packageId": "watools.plugin.xxx",
  "name": "插件名称",
  "description": "功能描述",
  "version": "0.0.1",
  "author": "作者",
  "uiEnabled": true,
  "entry": "app.js"
}
```

**字段约束**:
- `packageId`: 格式 `watools.plugin.xxx`,全局唯一
- `uiEnabled`: `true` 表示包含 UI 插件,`false` 表示纯 Executable

### app.js

```javascript
const entry = [
    {
        type: "ui" | "executable",
        subTitle: "操作描述",
        icon: "icon-name" | "🔢" | null,
        match: (context) => boolean,

        // UI 插件必需
        file?: "index.html",

        // Executable 插件必需
        execute?: async (context) => { }
    }
];

export default entry;
```

**约束**:
- 可导出多个 entry
- `match` 必须同步返回 boolean,执行时间 < 10ms
- `execute` 必须返回 Promise
- `icon`: Lucide Icons 名称 / Emoji / null

### index.html (UI 插件)

标准 HTML5 文档,可使用任意 CSS/JS 框架。

**关键点**:
- ESC 键由主应用自动处理,无需实现
- 使用 `<script type="module">` 导入模块

### 打包格式

插件打包为 `.wt` 文件(ZIP 格式):

```bash
# 压缩(文件必须在 ZIP 根级别)
zip -r plugin.wt manifest.json app.js index.html

# ✅ 正确结构
plugin.wt/
├── manifest.json
├── app.js
└── index.html

# ❌ 错误结构
plugin.wt/
└── watools.plugin.xxx/  ← 不要包含父文件夹
    ├── manifest.json
    └── ...
```

### 代码运行环境约束

**关键要求**: 产出的代码必须是**浏览器原生可运行**的,不能是框架脚手架项目。

**✅ 允许**:
- 原生 HTML/CSS/JavaScript
- ES Module (`<script type="module">`)
- 浏览器直接支持的语法(ES6+)
- CDN 引入的库(React CDN、Vue CDN 等)
- 内联的 TypeScript(如果使用支持浏览器的编译器,如 Babel Standalone)

**❌ 禁止**:
- 需要 `npm install` 的项目
- 需要构建工具的项目(webpack、vite、rollup)
- 包含 `package.json`、`node_modules/` 的项目
- JSX/TSX 文件(除非通过 CDN 实时编译)
- 需要编译步骤的框架代码

**示例**:

```html
<!-- ✅ 正确: 浏览器可直接运行 -->
<!DOCTYPE html>
<html>
<head>
    <script src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
    <script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
</head>
<body>
    <div id="root"></div>
    <script type="module">
        // 浏览器原生 JavaScript
    </script>
</body>
</html>

<!-- ❌ 错误: 需要构建工具 -->
// App.tsx
import React from 'react'  // 需要 npm install
export default function App() { }
```

**原则**: 插件文件解压后可直接在浏览器中打开 `index.html` 运行,无需任何安装或编译步骤。

---

## 运行时环境

### PluginContext

所有 `match` 和 `execute` 接收相同的 context 对象:

```typescript
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
```

**context 一致性**: React useMemo 保证 `match` 和 `execute` 接收同一实例。

### 插件执行流程

```
用户输入 → 遍历 match(context) → 匹配成功
    ↓
type === "ui" → 导航到 iframe
type === "executable" → 调用 execute(context)
```

---

## 可用 API

### window.runtime (Wails Runtime API)

```typescript
// 事件
EventsEmit(eventName: string, ...data: any): void
EventsOn(eventName: string, callback: (...data: any) => void): () => void
EventsOnce(eventName: string, callback: (...data: any) => void): () => void
EventsOff(eventName: string): void

// 日志
LogTrace(message: string): void
LogDebug(message: string): void
LogInfo(message: string): void
LogWarning(message: string): void
LogError(message: string): void
LogFatal(message: string): void  // 应用退出

// 窗口
WindowShow(): void
WindowHide(): void
WindowSetSize(width: number, height: number): void
WindowGetSize(): Promise<{w: number, h: number}>
WindowSetPosition(x: number, y: number): void
WindowGetPosition(): Promise<{x: number, y: number}>
WindowCenter(): void
WindowMaximise(): void
WindowMinimise(): void
WindowFullscreen(): void
WindowSetTitle(title: string): void
WindowSetAlwaysOnTop(b: boolean): void

// 剪贴板
ClipboardGetText(): Promise<string>
ClipboardSetText(text: string): Promise<boolean>

// 浏览器
BrowserOpenURL(url: string): void

// 应用
Quit(): void
Hide(): void
Show(): void
Environment(): Promise<{buildType: string, platform: string, arch: string}>

// 拖拽
OnFileDrop(callback: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean): void
OnFileDropOff(): void
```

### window.watools (WaTools Custom API)

```typescript
// 通用 HTTP 代理(绕过 CORS)
HttpProxy(request: HttpProxyRequest): Promise<HttpProxyResponse>

// 打开文件夹
OpenFolder(folderPath: string): Promise<void>

// 保存 Base64 图片
SaveBase64Image(base64String: string): Promise<string>

// 插件存储 API (持久化键值存储，自动注入 packageId)
StorageGet(key: string): Promise<any>
StorageSet(key: string, value: any): Promise<void>
StorageRemove(key: string): Promise<void>
StorageClear(): Promise<void>
StorageKeys(): Promise<string[]>

// 类型
type HttpProxyRequest = {
    url: string
    method?: string
    headers?: Record<string, string>
    body?: string
    timeout?: number
}

type HttpProxyResponse = {
    status_code: number
    headers: Record<string, any>
    body: string
    error: string | null
}
```

---

## 开发约束

### 安全约束

- 插件运行在 iframe 沙箱,无法访问主应用
- 只能调用 `window.runtime` 和 `window.watools` API
- 无法直接读写文件系统
- 无法绕过 CORS(必须使用 HttpProxy)

### 性能约束

- `match` 函数: < 10ms,禁止异步/网络请求
- `execute` 函数: 无超时限制,建议提供进度反馈

### 数据持久化

推荐使用 `window.watools.StorageXxx` API (跨平台、自动同步到数据库):

```javascript
// ✅ 推荐: watools Storage API (后端持久化)
await window.watools.StorageSet('apiKey', 'your-api-key')
const apiKey = await window.watools.StorageGet('apiKey')
await window.watools.StorageRemove('apiKey')
await window.watools.StorageClear()
const keys = await window.watools.StorageKeys()

// ✅ 也可使用: localStorage (仅限浏览器)
localStorage.setItem('key', JSON.stringify(data))
const data = JSON.parse(localStorage.getItem('key') || '{}')
```

**区别**:
- `watools.StorageXxx`: 后端数据库持久化,插件卸载后数据保留
- `localStorage`: 浏览器本地存储,清除缓存后数据丢失

---

## 开发模式

### match 函数模式

```javascript
// ✅ 简单快速
match: (context) => {
    return context.input.value.trim().startsWith('keyword')
}

// ✅ 剪贴板类型匹配
match: (context) => {
    return context.clipboard?.contentType === 'image'
}

// ❌ 禁止异步
match: async (context) => { }  // 错误!

// ❌ 禁止复杂计算
match: (context) => {
    return /complex/.test(expensiveOperation())  // 可能超时
}
```

### HttpProxy 使用模式

```javascript
const response = await window.watools.HttpProxy({
    url: 'https://api.example.com',
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data),
    timeout: 30000
})

if (response.error || response.status_code !== 200) {
    throw new Error(response.error || `HTTP ${response.status_code}`)
}

const result = JSON.parse(response.body)
```

### Storage API 使用模式

```javascript
// 存储配置
await window.watools.StorageSet('apiKey', 'sk-xxx')
await window.watools.StorageSet('config', {theme: 'dark', lang: 'zh'})

// 读取配置
const apiKey = await window.watools.StorageGet('apiKey')
const config = await window.watools.StorageGet('config') || {theme: 'light'}

// 删除单个键
await window.watools.StorageRemove('apiKey')

// 清空所有数据
await window.watools.StorageClear()

// 列出所有键
const keys = await window.watools.StorageKeys()
console.log(keys)  // ['apiKey', 'config']
```

### 多 Entry 模式

```javascript
const entry = [
    // Entry 1: 快速执行
    {
        type: "executable",
        match: (context) => context.input.value.startsWith('calc'),
        execute: async (context) => {
            const result = calculate(context.input.value)
            await window.runtime.ClipboardSetText(result)
        }
    },

    // Entry 2: 完整 UI
    {
        type: "ui",
        match: (context) => context.input.value === 'calculator',
        file: "index.html"
    }
];
```

### 模块化开发

```javascript
// app.js
execute: async (context) => {
    const { calculate } = await import('./calculator-core.js')
    const result = await calculate(context.input.value)
}

// calculator-core.js
export function calculate(expression) {
    return eval(expression)
}
```

---

## 实际案例

### 案例 1: Calculator (混合插件)

**结构**:
```
watools.plugin.calculator/
├── manifest.json (uiEnabled: true)
├── app.js (2 个 entry)
├── index.html
└── calculator-core.js
```

**关键点**:
- Executable entry: 匹配数学表达式,快速计算
- UI entry: 匹配关键词,打开完整界面
- 懒加载: `import('./calculator-core.js')`

### 案例 2: Common (纯 Executable)

**结构**:
```
watools.plugin.common/
├── manifest.json (uiEnabled: false)
└── app.js (3 个 entry)
```

**关键点**:
- Entry 1: 打开 URL/文件夹
- Entry 2: 复制文件路径(基于 `clipboardContentType === "files"`)
- Entry 3: 保存剪贴板图片(基于 `clipboardContentType === "image"`)

### 案例 3: Translator (纯 UI)

**结构**:
```
watools.plugin.translator/
├── manifest.json (uiEnabled: true)
├── app.js (1 个 ui entry)
└── index.html
```

**关键点**:
- 使用 HttpProxy 调用 DeepL API
- 使用 `window.watools.StorageXxx` 持久化 API Key
- 防抖优化(700ms)

---

## 设计原则

1. **通用 API 优先**: 使用 HttpProxy 而非专用后端 API
2. **插件自包含**: 不依赖主应用状态
3. **简单对象传递**: 避免复杂回调模式
4. **快速匹配**: match 函数 < 10ms
5. **错误处理**: 不让错误传播到主应用

---

## 完整类型定义

```typescript
// PluginEntry
type PluginEntry = {
    type: "executable" | "ui"
    subTitle: string
    match: (context: PluginContext) => boolean
    execute?: (context: PluginContext) => Promise<void>
    icon: string | null
    file?: string
}

// PluginContext
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

// HttpProxy
type HttpProxyRequest = {
    url: string
    method?: string
    headers?: Record<string, string>
    body?: string
    timeout?: number
}

type HttpProxyResponse = {
    status_code: number
    headers: Record<string, any>
    body: string
    error: string | null
}

// Wails Runtime
type Position = { x: number, y: number }
type Size = { w: number, h: number }
type Screen = {
    isCurrent: boolean
    isPrimary: boolean
    width: number
    height: number
}
type EnvironmentInfo = {
    buildType: string
    platform: string
    arch: string
}
```

---

## 预期输出

生成的插件必须满足:

- ✅ 文件结构正确(manifest.json + app.js + [index.html])
- ✅ manifest.json 字段完整
- ✅ app.js 导出有效的 entry 配置
- ✅ match 函数同步且快速(< 10ms)
- ✅ 使用 HttpProxy 而非 fetch
- ✅ 错误处理完善
- ✅ 打包为 .wt 时无父文件夹
- ✅ icon 使用 Lucide Icons / Emoji / null
