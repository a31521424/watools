# WaTools 插件开发技术规范

本文档定义 WaTools 插件系统的技术规范,供 LLM 生成符合要求的插件。

---

## ⚡️ LLM 快速指南 (必读)

### 核心原则

**最终产出 = 浏览器可直接运行的文件 + .wt 压缩包**

不是项目源代码,而是编译后的可运行文件!

### 构建流程决策树

```
START: 用户需要插件
    ↓
┌─────────────────────────────────┐
│ 1. 判断复杂度                    │
└─────────────────────────────────┘
    ↓
复杂吗? (需要 TypeScript/React/框架?)
    │
    ├─ 否 → 使用【简单模式】
    │        ├─ 直接编写 HTML/JS/CSS
    │        ├─ 创建 manifest.json + app.js + index.html
    │        └─ 跳转到【步骤 5: 打包】
    │
    └─ 是 → 使用【构建模式】
             ↓
        ┌─────────────────────────────────┐
        │ 2. 创建项目结构                  │
        └─────────────────────────────────┘
        my-plugin-project/
        ├── src/              ← 开发源码
        ├── public/           ← 静态资源
        │   ├── manifest.json ← 必须放这里!
        │   └── app.js        ← 必须放这里!
        ├── package.json
        └── vite.config.ts    ← 配置构建
             ↓
        ┌─────────────────────────────────┐
        │ 3. 配置 Vite                     │
        └─────────────────────────────────┘
        // vite.config.ts
        export default defineConfig({
          build: { outDir: 'dist' },
          publicDir: 'public'  // 自动复制到 dist/
        })
             ↓
        ┌─────────────────────────────────┐
        │ 4. 构建项目                      │
        └─────────────────────────────────┘
        $ npm run build

        产物结构:
        dist/
        ├── manifest.json    ← 自动复制
        ├── app.js          ← 自动复制
        ├── index.html      ← 编译产物
        └── assets/         ← 编译产物
            ├── index-abc.js
            └── index-def.css
             ↓
        ┌─────────────────────────────────┐
        │ 5. 打包为 .wt                    │
        └─────────────────────────────────┘
        $ cd dist
        $ zip -r ../plugin-name.wt *

        验证: 解压后直接打开 index.html 能运行!
             ↓
        ✅ 完成! 产出 plugin-name.wt
```

### LLM 输出清单

**【简单模式】输出**:
```
✅ manifest.json (配置文件)
✅ app.js (入口配置)
✅ index.html (完整 HTML,包含所有代码)
✅ 打包命令: zip -r plugin.wt manifest.json app.js index.html
```

**【构建模式】输出**:
```
✅ 完整项目目录结构
✅ package.json (含 build 和 package 脚本)
✅ vite.config.ts (publicDir: 'public')
✅ public/manifest.json
✅ public/app.js
✅ src/App.tsx (或其他源文件)
✅ 构建命令:
   npm install
   npm run build
   cd dist && zip -r ../plugin-name.wt *
```

### 快速检查表

LLM 在输出后必须自检:

- [ ] 最终产出是 `dist/` 目录内容(构建模式) 或直接文件(简单模式)
- [ ] `manifest.json` 和 `app.js` 在输出根级别(不在子目录)
- [ ] `.wt` 文件不包含 `src/`、`node_modules/`、`package.json`
- [ ] 解压后打开 `index.html` 能在浏览器直接运行
- [ ] 构建模式必须有 `npm run build` 和打包命令

---

## 标准模板

### 【简单模式】完整示例

适用场景: 简单 UI、计算器、文本处理等

**文件清单**:
```
watools.plugin.example/
├── manifest.json
├── app.js
└── index.html
```

**manifest.json**:
```json
{
  "packageId": "watools.plugin.example",
  "name": "示例插件",
  "description": "功能描述",
  "version": "0.0.1",
  "author": "作者",
  "uiEnabled": true,
  "entry": "app.js"
}
```

**app.js**:
```javascript
const entry = [{
    type: "ui",
    subTitle: "打开插件界面",
    icon: "star",
    match: (context) => context.input.value.startsWith('example'),
    file: "index.html"
}];
export default entry;
```

**index.html** (完整可运行):
```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Example Plugin</title>
    <style>
        body { font-family: sans-serif; padding: 20px; }
    </style>
</head>
<body>
    <h1>Hello WaTools!</h1>
    <button id="btn">复制文本</button>
    <script type="module">
        // API 包装 (防止浏览器调试崩溃)
        const api = {
            clipboard: {
                setText: async (text) => {
                    if (window.runtime?.ClipboardSetText) {
                        return await window.runtime.ClipboardSetText(text);
                    }
                    await navigator.clipboard.writeText(text);
                }
            }
        };

        // 自定义 toast (不能用 alert!)
        function showToast(msg) {
            const toast = document.createElement('div');
            toast.style.cssText = 'position:fixed;top:20px;right:20px;background:#333;color:#fff;padding:12px 20px;border-radius:4px;';
            toast.textContent = msg;
            document.body.appendChild(toast);
            setTimeout(() => toast.remove(), 2000);
        }

        // 业务逻辑
        document.getElementById('btn').addEventListener('click', async () => {
            await api.clipboard.setText('Hello from plugin!');
            showToast('已复制');  // ✅ 使用自定义 toast
        });
    </script>
</body>
</html>
```

**打包**:
```bash
cd watools.plugin.example
zip -r ../example.wt manifest.json app.js index.html
```

---

### 【构建模式】完整示例

适用场景: 复杂 UI、TypeScript、React/Vue 项目

**步骤 1: 项目结构**
```
my-plugin/
├── src/
│   ├── App.tsx          ← 开发源码
│   └── main.tsx
├── public/
│   ├── manifest.json    ← 必须放这里!
│   └── app.js          ← 必须放这里!
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

**步骤 2: public/manifest.json** (与简单模式相同)

**步骤 3: public/app.js** (与简单模式相同)

**步骤 4: package.json**
```json
{
  "name": "my-plugin",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "package": "cd dist && zip -r ../my-plugin.wt * && cd .."
  },
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  },
  "devDependencies": {
    "@types/react": "^18.0.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0",
    "@vitejs/plugin-react": "^4.0.0"
  }
}
```

**步骤 5: vite.config.ts** (关键配置!)
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist'
  },
  publicDir: 'public'  // 自动复制 manifest.json 和 app.js 到 dist/
})
```

**步骤 6: src/App.tsx** (示例代码)
```tsx
import { useState } from 'react'

// API 包装 (防止浏览器调试崩溃)
const api = {
  clipboard: {
    setText: async (text: string) => {
      if ((window as any).runtime?.ClipboardSetText) {
        return await (window as any).runtime.ClipboardSetText(text);
      }
      await navigator.clipboard.writeText(text);
    }
  }
};

export default function App() {
  const [text, setText] = useState('Hello WaTools!');
  const [toast, setToast] = useState('');

  const handleCopy = async () => {
    await api.clipboard.setText(text);
    setToast('已复制');  // ✅ 使用状态控制 toast
    setTimeout(() => setToast(''), 2000);
  };

  return (
    <div>
      <input value={text} onChange={(e) => setText(e.target.value)} />
      <button onClick={handleCopy}>复制</button>
      {toast && <div className="toast">{toast}</div>}
    </div>
  );
}
```

**步骤 7: 构建和打包**
```bash
# 安装依赖
npm install

# 构建 (产物在 dist/)
npm run build

# 验证产物结构
ls dist/
# 输出: manifest.json app.js index.html assets/

# 打包为 .wt
npm run package

# 验证
unzip -l my-plugin.wt
# 应该看到: manifest.json app.js index.html assets/
```

---

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

## 常见错误与解决

### ❌ 错误 1: .wt 文件包含源代码

```bash
# 错误的打包
plugin.wt/
├── src/              ← 不应该存在!
├── node_modules/     ← 不应该存在!
├── package.json      ← 不应该存在!
└── manifest.json
```

**解决**: 只打包 `dist/` 目录内容
```bash
cd dist && zip -r ../plugin.wt *
```

### ❌ 错误 2: manifest.json 和 app.js 在子目录

```bash
# 错误的结构
plugin.wt/
└── public/           ← 不应该有父目录!
    ├── manifest.json
    └── app.js
```

**解决**: 使用 `publicDir: 'public'` 让 Vite 自动复制到 dist/ 根级别

### ❌ 错误 3: 浏览器调试时 API 崩溃

```javascript
// 错误: 直接调用 Wails API
await window.runtime.ClipboardSetText(text)  // 浏览器中会报错!
```

**解决**: 使用 API 包装
```javascript
const setText = async (text) => {
  if (window.runtime?.ClipboardSetText) {
    return await window.runtime.ClipboardSetText(text);
  }
  await navigator.clipboard.writeText(text);  // 浏览器降级
};
```

### ❌ 错误 4: 使用浏览器原生弹窗

```javascript
// 错误: 使用 alert/confirm/prompt
alert('操作成功');  // ❌ 会阻塞整个应用!
if (confirm('确定删除?')) {  // ❌ 会阻塞整个应用!
  // ...
}
```

**解决**: 使用自定义 UI 组件
```javascript
// ✅ 推荐: 自定义 toast
function showToast(message) {
  const toast = document.createElement('div');
  toast.style.cssText = 'position:fixed;top:20px;right:20px;background:#333;color:#fff;padding:12px 20px;border-radius:4px;';
  toast.textContent = message;
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}

showToast('操作成功');
```

---

## 核心配置参考

### manifest.json 字段说明

```json
{
  "packageId": "watools.plugin.xxx",  // 必须以 watools.plugin. 开头
  "name": "插件名称",                  // 显示名称
  "description": "功能描述",           // 简短描述
  "version": "0.0.1",                 // 语义化版本
  "author": "作者",                   // 开发者
  "uiEnabled": true,                  // true=包含UI插件, false=纯Executable
  "entry": "app.js"                   // 入口文件,固定为 app.js
}
```

### app.js 配置说明

```javascript
const entry = [
    {
        type: "ui" | "executable",      // ui=打开界面, executable=后台执行
        subTitle: "操作描述",            // 显示在搜索结果的副标题
        icon: "star",                   // Lucide Icons 名称 或 Emoji 或 null
        match: (context) => boolean,    // 匹配函数,必须同步返回
        file: "index.html",             // UI 插件必须指定
        execute: async (context) => {}  // Executable 插件必须指定
    }
];
export default entry;
```

### Vite 配置模板

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist'
  },
  publicDir: 'public'  // 关键! 自动复制 public/ 下的文件到 dist/
})
```

---

## 运行时环境

### PluginContext 对象

`match` 和 `execute` 函数接收相同的 context:

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

**示例**:
```javascript
match: (context) => {
    // 匹配文本输入
    if (context.input.value.startsWith('calc')) return true;

    // 匹配剪贴板图片
    if (context.clipboard?.contentType === 'image') return true;

    return false;
}
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

## API 包装模板 (防止浏览器调试崩溃)

**推荐做法**: 创建 API 包装层,兼容浏览器和 Wails 环境

```javascript
// watools-api.js
export const api = {
  // 剪贴板 (别名浏览器原生 API)
  clipboard: {
    getText: async () => {
      if (window.runtime?.ClipboardGetText) {
        return await window.runtime.ClipboardGetText();
      }
      return await navigator.clipboard.readText().catch(() => '');
    },
    setText: async (text) => {
      if (window.runtime?.ClipboardSetText) {
        return await window.runtime.ClipboardSetText(text);
      }
      return await navigator.clipboard.writeText(text).then(() => true).catch(() => false);
    }
  },

  // HTTP (核心功能,提供降级)
  http: async (request) => {
    if (window.watools?.HttpProxy) {
      return await window.watools.HttpProxy(request);
    }
    // 浏览器降级 (受 CORS 限制)
    const response = await fetch(request.url, {
      method: request.method || 'GET',
      headers: request.headers,
      body: request.body
    });
    return {
      status_code: response.status,
      body: await response.text(),
      error: null
    };
  },

  // 存储 (核心功能,提供降级)
  storage: {
    get: async (key) => {
      if (window.watools?.StorageGet) return await window.watools.StorageGet(key);
      const value = localStorage.getItem(key);
      return value ? JSON.parse(value) : null;
    },
    set: async (key, value) => {
      if (window.watools?.StorageSet) return await window.watools.StorageSet(key, value);
      localStorage.setItem(key, JSON.stringify(value));
    }
  },

  // 日志 (可忽略,降级到 console)
  log: {
    info: (...args) => window.runtime?.LogInfo?.(...args) || console.log('[INFO]', ...args),
    error: (...args) => window.runtime?.LogError?.(...args) || console.error('[ERROR]', ...args)
  }
};
```

**使用**:
```javascript
import { api } from './watools-api.js'

await api.clipboard.setText('复制内容')
const response = await api.http({url: 'https://api.com'})
const apiKey = await api.storage.get('apiKey')
```

---

## 浏览器原生 API 限制

### ❌ 禁止使用的浏览器原生 API

插件运行在 Wails 的 Webview 环境中，以下浏览器原生 API **不可用或行为异常**:

#### 1. 交互弹窗 (全部禁止)
```javascript
// ❌ 禁止使用
alert('消息')              // 会阻塞整个应用
confirm('确认?')           // 会阻塞整个应用
prompt('输入:')            // 会阻塞整个应用
```

**替代方案**: 使用自定义 UI 组件
```javascript
// ✅ 推荐: 自定义 toast/modal
function showToast(message) {
  const toast = document.createElement('div');
  toast.className = 'toast';
  toast.textContent = message;
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}

// ✅ 推荐: 自定义确认框
function showConfirm(message, onConfirm) {
  const modal = document.createElement('div');
  modal.innerHTML = `
    <div class="modal">
      <p>${message}</p>
      <button id="confirm-yes">确定</button>
      <button id="confirm-no">取消</button>
    </div>
  `;
  document.body.appendChild(modal);
  document.getElementById('confirm-yes').onclick = () => {
    onConfirm(true);
    modal.remove();
  };
  document.getElementById('confirm-no').onclick = () => {
    onConfirm(false);
    modal.remove();
  };
}
```

#### 2. 文件系统访问 (受限)
```javascript
// ❌ 不可用或行为异常
window.showOpenFilePicker()       // File System Access API
window.showSaveFilePicker()
window.showDirectoryPicker()

// ⚠️ 可用但有限制
const input = document.createElement('input');
input.type = 'file';
input.click();  // 可以用,但推荐使用 Wails 的文件选择 API
```

**替代方案**: 使用 Wails Runtime API 或拖拽
```javascript
// ✅ 推荐: 使用文件拖拽
window.runtime.OnFileDrop((x, y, paths) => {
  console.log('拖入文件:', paths);
}, false);

// ✅ 或者: 使用 <input type="file">
document.getElementById('file-input').addEventListener('change', (e) => {
  const files = e.target.files;
  // 处理文件
});
```

#### 3. 窗口操作 (部分禁止)
```javascript
// ❌ 禁止使用
window.open(url)              // 可能无法正常工作
window.close()                // 使用 window.runtime.Quit()
window.resizeTo(w, h)         // 使用 window.runtime.WindowSetSize()
window.moveTo(x, y)           // 使用 window.runtime.WindowSetPosition()

// ⚠️ 可用但不推荐
location.href = 'new-url'     // 会导航离开插件,避免使用
history.pushState()           // 插件内路由可用,但需谨慎
```

**替代方案**: 使用 Wails Runtime API
```javascript
// ✅ 推荐
window.runtime.WindowSetSize(800, 600);
window.runtime.WindowCenter();
window.runtime.BrowserOpenURL('https://example.com');  // 在外部浏览器打开
```

#### 4. 本地存储 (部分可用)
```javascript
// ✅ 可用: localStorage/sessionStorage
localStorage.setItem('key', 'value');  // 可用但数据仅在浏览器缓存

// ✅ 推荐: 使用 watools Storage API (后端持久化)
await window.watools.StorageSet('key', 'value');  // 数据库持久化
```

#### 5. 网络请求 (受 CORS 限制)
```javascript
// ⚠️ 受 CORS 限制
fetch('https://api.example.com')  // 会遇到 CORS 问题

// ✅ 推荐: 使用 HttpProxy
await window.watools.HttpProxy({
  url: 'https://api.example.com'
});
```

#### 6. 其他受限 API
```javascript
// ❌ 可能不可用或行为异常
window.print()                // 打印功能,可能无法正常工作
navigator.geolocation         // 地理位置,需要权限且可能不可用
navigator.mediaDevices        // 摄像头/麦克风,需要权限
Notification API              // 系统通知,使用 Wails 事件系统替代
ServiceWorker                 // 不支持
WebSocket                     // 可用,但推荐通过 Wails 后端处理
```

### ✅ 可以安全使用的浏览器 API

以下浏览器原生 API 在 Wails 环境中**可以正常使用**:

```javascript
// ✅ DOM 操作
document.querySelector()
document.createElement()
element.addEventListener()

// ✅ 定时器
setTimeout() / setInterval()
requestAnimationFrame()

// ✅ 数据处理
JSON.parse() / JSON.stringify()
Array/Object/String 方法
FormData / URLSearchParams

// ✅ 剪贴板 (推荐用 Wails API)
navigator.clipboard.readText()   // 可用但推荐 window.runtime.ClipboardGetText()
navigator.clipboard.writeText()  // 可用但推荐 window.runtime.ClipboardSetText()

// ✅ Canvas/图形
<canvas> 元素
CanvasRenderingContext2D
WebGL (如果系统支持)

// ✅ 音视频
<audio> / <video> 元素
Web Audio API

// ✅ 拖拽
Drag and Drop API
window.runtime.OnFileDrop()  // Wails 增强版
```

### 📋 快速参考表

| API 类型 | 状态 | 替代方案 |
|---------|------|---------|
| alert/confirm/prompt | ❌ 禁止 | 自定义 UI 组件 |
| window.open() | ❌ 禁止 | window.runtime.BrowserOpenURL() |
| File System Access API | ❌ 不可用 | 拖拽 或 `<input type="file">` |
| fetch (跨域) | ⚠️ 受限 | window.watools.HttpProxy() |
| localStorage | ✅ 可用 | window.watools.Storage (推荐) |
| Canvas/Audio/Video | ✅ 可用 | 直接使用 |
| DOM/Timer/Array | ✅ 可用 | 直接使用 |

---

## 附录: 完整 API 参考

### window.runtime (Wails Runtime)

**剪贴板**:
- `ClipboardGetText(): Promise<string>`
- `ClipboardSetText(text: string): Promise<boolean>`

**窗口控制**:
- `Hide() / Show() / Quit()`
- `WindowCenter() / WindowMaximise() / WindowMinimise()`
- `WindowSetSize(w, h) / WindowGetSize()`

**日志**:
- `LogInfo(msg) / LogError(msg) / LogDebug(msg)`

**其他**:
- `BrowserOpenURL(url: string)`
- `Environment(): Promise<{platform, arch, buildType}>`

### window.watools (WaTools Custom API)

**核心 API**:
- `HttpProxy(request): Promise<response>` - HTTP 代理
- `StorageGet/Set/Remove/Clear/Keys()` - 持久化存储
- `OpenFolder(path)` - 打开文件夹
- `SaveBase64Image(base64): Promise<path>` - 保存图片

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

## 最终检查清单 (LLM 必读)

输出插件前,必须确认:

**文件结构**:
- [ ] manifest.json 在根目录 (不在 public/ 或 src/)
- [ ] app.js 在根目录
- [ ] index.html 在根目录 (UI 插件)
- [ ] 无 package.json、node_modules、src/ (构建模式必须输出 dist/ 内容)

**配置文件**:
- [ ] manifest.json 包含所有必需字段
- [ ] packageId 格式: `watools.plugin.xxx`
- [ ] app.js 正确导出 `export default entry`
- [ ] match 函数同步返回 boolean

**构建模式专项**:
- [ ] vite.config.ts 配置 `publicDir: 'public'`
- [ ] package.json 包含 `build` 和 `package` 脚本
- [ ] manifest.json 和 app.js 放在 public/ 目录

**API 使用**:
- [ ] 使用 API 包装 (防止浏览器调试崩溃)
- [ ] HTTP 请求使用 `window.watools.HttpProxy`
- [ ] 存储使用 `window.watools.StorageXxx`
- [ ] 不使用 alert/confirm/prompt (用自定义 UI)
- [ ] 不使用 window.open() (用 BrowserOpenURL)

**打包验证**:
- [ ] .wt 文件内容在根级别 (无父文件夹)
- [ ] 解压后打开 index.html 能在浏览器运行
- [ ] 文件总大小 < 50MB
