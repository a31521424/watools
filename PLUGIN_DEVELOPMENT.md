# WaTools 插件开发技术规范

本文档定义 WaTools 插件系统的技术规范,供 LLM 生成符合要求的插件。

---

## 产出目标

**⚠️ 核心原则**: 插件的最终产出必须是**浏览器可直接运行的文件**,而不是开发项目的源代码。

### 产出形式

插件可以通过以下两种方式开发:

1. **原生开发** (推荐简单场景)
   - 直接编写 HTML/CSS/JavaScript
   - 无需构建步骤
   - 开发即产出

2. **构建工具开发** (推荐复杂场景)
   - 使用 Vite/Webpack/Rollup 等构建工具
   - 开发时使用 TypeScript/React/Vue 等
   - **必须编译为浏览器可运行的文件**
   - 最终产出是 `dist/` 或 `build/` 目录内容

### 插件文件结构

**基础结构**(原生开发):
```
watools.plugin.{name}/
├── manifest.json    # 元数据(必需)
├── app.js          # 入口配置(必需)
└── index.html      # UI 页面(UI 插件需要)
```

**构建产物结构**(使用构建工具):
```
watools.plugin.{name}/
├── manifest.json    # 元数据(必需)
├── app.js          # 入口配置(必需)
├── index.html      # 主页面(编译后)
├── assets/         # 编译后的资源
│   ├── index-[hash].js
│   └── index-[hash].css
└── ...             # 其他编译产物
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

插件打包为 `.wt` 文件(本质是 ZIP 格式,改后缀名):

#### 原生开发打包

```bash
# 压缩(文件必须在 ZIP 根级别)
cd watools.plugin.{name}
zip -r ../plugin-name.wt manifest.json app.js index.html

# ✅ 正确结构
plugin-name.wt/
├── manifest.json
├── app.js
└── index.html

# ❌ 错误结构
plugin-name.wt/
└── watools.plugin.xxx/  ← 不要包含父文件夹
    ├── manifest.json
    └── ...
```

#### 使用构建工具打包

如果使用 Vite/Webpack 等构建工具,**必须打包编译后的产物**:

```bash
# 1. 构建项目
npm run build  # 或 vite build

# 2. 进入构建产物目录(通常是 dist/)
cd dist

# 3. 确保 manifest.json 和 app.js 在 dist/ 中
# (构建工具需要配置复制这些文件)

# 4. 压缩为 .wt 文件
zip -r ../plugin-name.wt *

# ✅ 正确流程
my-plugin-project/
├── src/              # 源代码(不打包)
├── dist/             # 构建产物(打包这个)
│   ├── manifest.json
│   ├── app.js
│   ├── index.html
│   └── assets/
├── package.json      # 不打包
└── vite.config.ts    # 不打包

# 最终 .wt 文件内容(来自 dist/)
plugin-name.wt/
├── manifest.json
├── app.js
├── index.html
└── assets/
```

**Vite 配置示例**:

```javascript
// vite.config.ts
import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    outDir: 'dist',
    rollupOptions: {
      input: 'index.html'
    }
  },
  plugins: [
    // 自动复制 manifest.json 和 app.js 到 dist/
    {
      name: 'copy-plugin-files',
      closeBundle() {
        const fs = require('fs')
        fs.copyFileSync('manifest.json', 'dist/manifest.json')
        fs.copyFileSync('app.js', 'dist/app.js')
      }
    }
  ]
})
```

**自动化打包脚本**:

```json
// package.json
{
  "scripts": {
    "build": "vite build",
    "package": "cd dist && zip -r ../plugin-name.wt * && cd .."
  }
}
```

### 代码运行环境约束

**关键要求**: **最终产出**的代码必须是**浏览器原生可运行**的。

#### 允许的开发方式

**✅ 原生开发**:
- 原生 HTML/CSS/JavaScript
- ES Module (`<script type="module">`)
- 浏览器直接支持的语法(ES6+)
- CDN 引入的库(React CDN、Vue CDN 等)
- 内联的 TypeScript(如果使用支持浏览器的编译器,如 Babel Standalone)

**✅ 构建工具开发** (Vite/Webpack/Rollup):
- 开发时使用 TypeScript/React/Vue/Svelte 等
- 使用 npm/yarn 管理依赖
- **但必须编译为浏览器可运行的文件**
- 打包时只包含 `dist/` 或 `build/` 目录内容

#### 禁止的产出形式

**❌ 以下内容不能出现在最终 .wt 文件中**:
- `package.json`、`package-lock.json`
- `node_modules/` 目录
- `src/` 源代码目录
- 未编译的 `.tsx`、`.jsx`、`.vue`、`.svelte` 文件
- 构建配置文件(`vite.config.ts`、`webpack.config.js` 等)

#### 示例对比

```html
<!-- ✅ 方式 1: 原生开发(无需构建) -->
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
```

```tsx
// ✅ 方式 2: 构建工具开发(需要编译)
// src/App.tsx (开发时)
import React from 'react'
export default function App() {
  return <div>Hello</div>
}

// ↓ npm run build ↓

// dist/index.html (最终产出)
<!DOCTYPE html>
<html>
<head>
    <script type="module" src="/assets/index-abc123.js"></script>
    <link rel="stylesheet" href="/assets/index-def456.css">
</head>
<body>
    <div id="root"></div>
</body>
</html>

// dist/assets/index-abc123.js (编译后的 bundle)
```

```tsx
// ❌ 错误: 直接打包源代码
plugin.wt/
├── manifest.json
├── app.js
├── src/
│   └── App.tsx  ← 浏览器无法运行
└── package.json  ← 不应该包含
```

**验证原则**: 将 `.wt` 文件解压,直接用浏览器打开 `index.html`,应该能正常运行。

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

## 开发调试最佳实践

### API Hook 策略

**问题**: 在开发插件时,如果直接使用 `window.watools` 或 `window.runtime` API,在浏览器中调试会因为这些 API 不存在而崩溃。

**解决方案**: 根据 API 重要性,使用不同的 hook 策略。

#### 策略 1: 忽略非关键 API (日志、分析等)

对于不影响功能的 API,可以简单忽略:

```javascript
// ✅ 安全的日志包装
const log = {
  info: (...args) => window.runtime?.LogInfo?.(...args) || console.log('[INFO]', ...args),
  error: (...args) => window.runtime?.LogError?.(...args) || console.error('[ERROR]', ...args),
  debug: (...args) => window.runtime?.LogDebug?.(...args) || console.log('[DEBUG]', ...args)
}

// 使用
log.info('插件已加载')  // Wails 环境 → 调用 LogInfo, 浏览器环境 → console.log
```

#### 策略 2: 别名原生 API (剪贴板、通知等)

对于有浏览器原生替代的 API,使用别名:

```javascript
// ✅ 剪贴板包装
const clipboard = {
  // 读取文本
  getText: async () => {
    if (window.runtime?.ClipboardGetText) {
      return await window.runtime.ClipboardGetText()
    }
    // 浏览器环境降级
    try {
      return await navigator.clipboard.readText()
    } catch (e) {
      console.warn('剪贴板读取失败:', e)
      return ''
    }
  },

  // 写入文本
  setText: async (text) => {
    if (window.runtime?.ClipboardSetText) {
      return await window.runtime.ClipboardSetText(text)
    }
    // 浏览器环境降级
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (e) {
      console.warn('剪贴板写入失败:', e)
      return false
    }
  }
}

// 使用
await clipboard.setText('Hello')  // Wails 和浏览器都能运行
```

#### 策略 3: Mock 核心业务 API (HTTP、存储等)

对于核心功能,提供完整 mock:

```javascript
// ✅ HTTP Proxy 包装
const http = {
  proxy: async (request) => {
    if (window.watools?.HttpProxy) {
      return await window.watools.HttpProxy(request)
    }

    // 浏览器环境降级(直接 fetch,会受 CORS 限制)
    console.warn('使用浏览器 fetch 替代 HttpProxy,可能遇到 CORS 问题')
    try {
      const response = await fetch(request.url, {
        method: request.method || 'GET',
        headers: request.headers,
        body: request.body,
        signal: request.timeout ? AbortSignal.timeout(request.timeout) : undefined
      })

      return {
        status_code: response.status,
        headers: Object.fromEntries(response.headers.entries()),
        body: await response.text(),
        error: null
      }
    } catch (error) {
      return {
        status_code: 0,
        headers: {},
        body: '',
        error: error.message
      }
    }
  }
}

// ✅ Storage 包装
const storage = {
  get: async (key) => {
    if (window.watools?.StorageGet) {
      return await window.watools.StorageGet(key)
    }
    // 浏览器环境降级
    const value = localStorage.getItem(key)
    return value ? JSON.parse(value) : null
  },

  set: async (key, value) => {
    if (window.watools?.StorageSet) {
      return await window.watools.StorageSet(key, value)
    }
    // 浏览器环境降级
    localStorage.setItem(key, JSON.stringify(value))
  },

  remove: async (key) => {
    if (window.watools?.StorageRemove) {
      return await window.watools.StorageRemove(key)
    }
    localStorage.removeItem(key)
  },

  clear: async () => {
    if (window.watools?.StorageClear) {
      return await window.watools.StorageClear()
    }
    localStorage.clear()
  }
}
```

#### 策略 4: 环境检测工具

```javascript
// ✅ 环境检测
const env = {
  isWails: () => typeof window.runtime !== 'undefined',
  isBrowser: () => typeof window.runtime === 'undefined',

  // 安全调用
  safeCall: async (watoolsApi, fallback) => {
    if (env.isWails() && watoolsApi) {
      return await watoolsApi()
    }
    if (fallback) {
      return await fallback()
    }
    console.warn('API 不可用且无降级方案')
    return null
  }
}

// 使用
await env.safeCall(
  () => window.watools.HttpProxy({url: 'https://api.com'}),
  () => fetch('https://api.com').then(r => r.json())
)
```

### 推荐的 API 包装模板

创建一个 `watools-api.js` 文件,集中管理所有 API:

```javascript
// watools-api.js
export const WaToolsAPI = {
  // 环境检测
  isWails: () => typeof window.runtime !== 'undefined',

  // 日志 (可忽略)
  log: {
    info: (...args) => window.runtime?.LogInfo?.(...args) || console.log('[INFO]', ...args),
    error: (...args) => window.runtime?.LogError?.(...args) || console.error('[ERROR]', ...args)
  },

  // 剪贴板 (别名原生)
  clipboard: {
    getText: async () => {
      if (window.runtime?.ClipboardGetText) return await window.runtime.ClipboardGetText()
      return await navigator.clipboard.readText().catch(() => '')
    },
    setText: async (text) => {
      if (window.runtime?.ClipboardSetText) return await window.runtime.ClipboardSetText(text)
      return await navigator.clipboard.writeText(text).then(() => true).catch(() => false)
    }
  },

  // HTTP (核心功能 mock)
  http: {
    proxy: async (request) => {
      if (window.watools?.HttpProxy) return await window.watools.HttpProxy(request)

      const response = await fetch(request.url, {
        method: request.method || 'GET',
        headers: request.headers,
        body: request.body
      })
      return {
        status_code: response.status,
        body: await response.text(),
        error: null
      }
    }
  },

  // 存储 (核心功能 mock)
  storage: {
    get: async (key) => {
      if (window.watools?.StorageGet) return await window.watools.StorageGet(key)
      const value = localStorage.getItem(key)
      return value ? JSON.parse(value) : null
    },
    set: async (key, value) => {
      if (window.watools?.StorageSet) return await window.watools.StorageSet(key, value)
      localStorage.setItem(key, JSON.stringify(value))
    }
  },

  // 窗口控制 (可忽略)
  window: {
    hide: () => window.runtime?.Hide?.() || console.log('[MOCK] 隐藏窗口')
  }
}

// 使用
import { WaToolsAPI } from './watools-api.js'

await WaToolsAPI.clipboard.setText('复制内容')
const response = await WaToolsAPI.http.proxy({url: 'https://api.com'})
```

### API 重要性分级

| API 类型 | 重要性 | 策略 | 示例 |
|---------|--------|------|------|
| 日志/调试 | 低 | 忽略(降级 console) | LogInfo, LogDebug |
| 窗口控制 | 低 | 忽略(mock) | Hide, Show |
| 剪贴板 | 中 | 别名原生 API | ClipboardGetText |
| 通知 | 中 | 别名原生 API | Notification API |
| HTTP 请求 | 高 | 完整 mock | HttpProxy |
| 存储 | 高 | 完整 mock | StorageGet/Set |
| 文件操作 | 高 | 完整 mock | OpenFolder |

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
