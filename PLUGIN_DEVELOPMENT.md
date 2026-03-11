# WaTools 插件开发技术规范

本文档定义 WaTools 插件系统的技术规范,供 LLM 生成符合要求的插件。

## 信任模型

WaTools 当前默认用户**完全了解并主动选择自己安装的插件**。

- 插件应视为用户主动信任的代码
- 当前实现不是面向“不受信任第三方插件市场”的强隔离沙箱
- 不要向用户描述为“安装任意来源插件也绝对安全”

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

### ⚠️ 关键限制 (必读)

**插件运行在 iframe 中,必须特别注意:**

1. **UI 布局限制**:
    - ❌ 禁止使用 `alert()`/`confirm()`/`prompt()` - 会阻塞整个应用
    - ⚠️ Modal/Dialog 使用 `position: fixed` 会受 iframe 边界限制
    - ❌ 不要自己实现插件最外层边框/卡片容器 - WaTools 外层窗口已经有边框
    - ❌ 插件 UI 必须视为 WaTools 主窗口中的**内嵌片段**,不要把插件设计成第二个独立小窗口
    - ❌ 不要默认使用居中 `max-width` 外壳、大外边距、整块圆角背景、嵌套卡片/面板来营造“窗口套窗口”的效果
    - ✅ 优先让核心输入区、结果区、列表区直接填充宿主提供的可用视口,只保留必要内边距和轻量分组
    - ❌ 更进一步,插件页面内部也不应依赖明显的描边边框来构建主视觉层级;优先使用留白、背景色、圆角、阴影或分组间距
    - ✅ 默认采用**极简、键盘优先**的效率型界面: 先保证输入、结果、历史或核心列表可直接工作,再考虑额外视觉层
    - ❌ 非必要不要加入数字面板、装饰性统计卡片、强调色按钮组、说明性大段文案
    - ✅ 如果任务本质是“输入 -> 计算/处理 -> 记录结果”,优先使用 `输入框 + 结果/历史列表 + 少量快捷键提示`
    - ✅ 推荐使用 Toast 通知或 Inline Dialog
    - ✅ 所有覆盖层组件要考虑 iframe viewport 限制

2. **快捷键跨平台支持**:
    - ❌ 禁止只监听 `e.ctrlKey` (macOS 用户无法使用)
    - ✅ 必须同时监听 `e.ctrlKey || e.metaKey`
    - ✅ 例如: `Ctrl+Shift+C` 和 `Meta+Shift+C` 应触发相同功能
    - ✅ 使用统一的快捷键处理函数(见文档示例)

详见文档后续章节的完整说明。

### 快速检查表

LLM 在输出后必须自检:

- [ ] 最终产出是 `dist/` 目录内容(构建模式) 或直接文件(简单模式)
- [ ] `manifest.json` 和 `app.js` 在输出根级别(不在子目录)
- [ ] `.wt` 文件不包含 `src/`、`node_modules/`、`package.json`
- [ ] 解压后打开 `index.html` 能在浏览器直接运行
- [ ] 构建模式必须有 `npm run build` 和打包命令
- [ ] **不使用 alert/confirm/prompt,使用自定义 UI**
- [ ] **所有快捷键同时监听 ctrlKey 和 metaKey**
- [ ] **插件页面不自绘边框/描边,WaTools 外层窗口已提供整体边界**
- [ ] **插件作为宿主内嵌片段铺满可用区域,不要做居中壳层、嵌套圆角窗口或大外边距**
- [ ] **默认先做极简、键盘优先界面,只保留完成任务所需的最少控件**

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

// 写入 Base64 图片到系统剪贴板
CopyBase64ImageToClipboard(base64String: string): Promise<void>

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
        },
        setImage: async (base64String) => {
            // 优先使用宿主 API，避免 WebView/浏览器权限限制
            if (window.watools?.CopyBase64ImageToClipboard) {
                await window.watools.CopyBase64ImageToClipboard(base64String);
                return true;
            }

            // 浏览器降级：仅调试时尝试，生产环境不要依赖
            const blob = await fetch(`data:image/png;base64,${base64String}`).then((res) => res.blob());
            if (navigator.clipboard?.write && window.ClipboardItem) {
                await navigator.clipboard.write([
                    new window.ClipboardItem({[blob.type]: blob})
                ]);
                return true;
            }

            throw new Error('Image clipboard is not available in current context');
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
await api.clipboard.setImage(base64Image)
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

#### 4.1 剪贴板图片写入 (强烈建议走宿主 API)
```javascript
// ❌ 不要默认依赖
navigator.clipboard.write([...])  // WebView 中常因权限/焦点/平台限制失败

// ✅ 推荐
await window.watools.CopyBase64ImageToClipboard(base64Png)
```

**说明**:
- 文本剪贴板通常可用 `window.runtime.ClipboardSetText()`
- 图片剪贴板在 Wails/WebView 中经常无法稳定获得浏览器授权
- 如果插件需要“复制图片”,应优先准备 PNG Base64 并调用 `window.watools.CopyBase64ImageToClipboard()`
- 浏览器环境仅把 `navigator.clipboard.write()` 当作调试降级方案,不要作为正式主路径

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
navigator.clipboard.write()      // 图片写入常受限,推荐 window.watools.CopyBase64ImageToClipboard()

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
| 剪贴板写文本 | ✅ 可用 | window.runtime.ClipboardSetText() |
| 剪贴板写图片 | ⚠️ 常受限 | window.watools.CopyBase64ImageToClipboard() |
| Canvas/Audio/Video | ✅ 可用 | 直接使用 |
| DOM/Timer/Array | ✅ 可用 | 直接使用 |

---

## UI 设计与交互最佳实践

### ⚠️ Iframe 布局限制

插件运行在**隔离的 iframe 环境**中,必须特别注意布局设计:

#### 1. Modal/Dialog 组件设计

```javascript
// ❌ 错误: 使用 fixed 定位可能超出 iframe 边界
.modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 9999;  // 在 iframe 内无法覆盖主应用
}
```

**问题**:
- `position: fixed` 相对于 iframe viewport,而非主窗口
- Modal 可能被 iframe 边界裁剪,无法居中显示在整个应用窗口
- `z-index` 无法穿透 iframe,无法覆盖主应用 UI

**✅ 推荐方案 1: 插件内 Modal (适用于简单场景)**

```javascript
// 使用相对定位,确保在插件可视区域内
.modal-overlay {
    position: absolute;  // 相对于插件容器
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
}

.modal-content {
    background: white;
    border-radius: 8px;
    padding: 24px;
    max-width: 90%;  // 避免超出 iframe 宽度
    max-height: 80vh;  // 避免超出 iframe 高度
    overflow-y: auto;
}

// 确保插件容器启用定位上下文
#app {
    position: relative;
    min-height: 100vh;
}
```

**✅ 推荐方案 2: Inline Dialog (最稳定)**

```javascript
// 不使用遮罩层,直接在页面流中显示对话框
function showInlineDialog(message) {
    const dialog = document.createElement('div');
    dialog.className = 'inline-dialog';
    dialog.innerHTML = `
    <div class="dialog-header">
      <h3>确认操作</h3>
      <button class="close-btn">&times;</button>
    </div>
    <div class="dialog-body">
      <p>${message}</p>
      <div class="dialog-actions">
        <button class="btn-cancel">取消</button>
        <button class="btn-confirm">确定</button>
      </div>
    </div>
  `;

    // 插入到页面当前位置,而非覆盖层
    document.getElementById('dialog-container').appendChild(dialog);

    return new Promise((resolve) => {
        dialog.querySelector('.btn-confirm').onclick = () => {
            dialog.remove();
            resolve(true);
        };
        dialog.querySelector('.btn-cancel').onclick =
            dialog.querySelector('.close-btn').onclick = () => {
                dialog.remove();
                resolve(false);
            };
    });
}

// CSS
.inline-dialog {
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.96);
    margin: 16px 0;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    animation: slideDown 0.2s ease-out;
}

@keyframes slideDown {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
}
```

**✅ 推荐方案 3: Toast 通知 (推荐)**

```javascript
// 使用 fixed 定位但确保在 iframe 可视范围内
function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;

    // 固定在 iframe 的顶部或底部角落
    toast.style.cssText = `
    position: fixed;
    top: 20px;
    right: 20px;
    background: ${type === 'success' ? '#10b981' : '#3b82f6'};
    color: white;
    padding: 12px 20px;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    z-index: 1000;
    max-width: calc(100vw - 40px);  // 防止超出 iframe 宽度
    animation: slideInRight 0.3s ease-out;
  `;

    document.body.appendChild(toast);
    setTimeout(() => {
        toast.style.animation = 'slideOutRight 0.3s ease-in';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}
```

#### 2. Dropdown/Popover 组件

```javascript
// ❌ 错误: 可能超出 iframe 边界被裁剪
.dropdown-menu {
    position: absolute;
    top: 100%;  // 向下展开可能被裁剪
}

// ✅ 推荐: 智能定位,检测空间
function showDropdown(triggerElement, menuItems) {
    const menu = document.createElement('div');
    menu.className = 'dropdown-menu';

    const triggerRect = triggerElement.getBoundingClientRect();
    const spaceBelow = window.innerHeight - triggerRect.bottom;
    const spaceAbove = triggerRect.top;

    // 智能判断向上还是向下展开
    if (spaceBelow < 200 && spaceAbove > spaceBelow) {
        menu.style.bottom = `${window.innerHeight - triggerRect.top}px`;
    } else {
        menu.style.top = `${triggerRect.bottom}px`;
    }

    menu.style.left = `${triggerRect.left}px`;
    menu.style.maxHeight = `${Math.max(spaceBelow, spaceAbove) - 20}px`;
    menu.style.overflowY = 'auto';

    document.body.appendChild(menu);
}
```

#### 3. 全屏覆盖层

```javascript
// ❌ 错误: 无法覆盖整个应用窗口
.fullscreen-overlay {
    position: fixed;
    inset: 0;  // 只能覆盖 iframe 区域
}

// ✅ 推荐: 调整预期,设计适配 iframe
.plugin-overlay {
    position: fixed;
    inset: 0;
    background: rgba(255, 255, 255, 0.95);  // 浅色背景,避免过于突兀
    backdrop-filter: blur(8px);  // 模糊背景增强视觉层次
}

// 或者: 使用页面内的容器
.content-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: white;
    z-index: 100;
}
```

#### 4. 响应式设计原则

```css
/* 插件容器应该是响应式的 */
#app {
    width: 100%;
    min-height: 100vh;
    padding: 16px;
    box-sizing: border-box;
}

/* 所有固定定位元素应该考虑 iframe 边界 */
.fixed-element {
    position: fixed;
    max-width: calc(100vw - 32px);  /* 留出边距 */
    max-height: calc(100vh - 32px);
}

/* 使用 dvh (dynamic viewport height) 替代 vh */
.full-height {
    height: 100dvh;  /* 更准确的视口高度 */
}
```

### ⌨️ 跨平台快捷键支持

插件需要同时支持 **Windows (Ctrl)** 和 **macOS (Command/Meta)** 平台的快捷键。

#### 核心原则

```javascript
// ❌ 错误: 只监听单一修饰键
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'c') {  // macOS 用户无法使用
        copyToClipboard();
    }
});

// ✅ 正确: 同时监听 Ctrl 和 Meta
document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
        e.preventDefault();
        copyToClipboard();
    }
});
```

#### 通用快捷键处理函数

```javascript
// 快捷键工具函数
const keyboard = {
    // 检查是否按下主修饰键 (Ctrl on Windows, Command on macOS)
    isPrimaryKey: (e) => {
        const isMac = /Mac|iPhone|iPod|iPad/.test(navigator.platform);
        return isMac ? e.metaKey : e.ctrlKey;
    },

    // 检查快捷键组合
    match: (e, key, modifiers = {}) => {
        if (e.key.toLowerCase() !== key.toLowerCase()) return false;

        const primaryPressed = e.ctrlKey || e.metaKey;
        const shiftPressed = e.shiftKey;
        const altPressed = e.altKey;

        // 检查是否需要主修饰键
        if (modifiers.primary && !primaryPressed) return false;
        if (!modifiers.primary && primaryPressed) return false;

        // 检查其他修饰键
        if (modifiers.shift !== undefined && modifiers.shift !== shiftPressed) return false;
        if (modifiers.alt !== undefined && modifiers.alt !== altPressed) return false;

        return true;
    }
};

// 使用示例
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + S: 保存
    if (keyboard.match(e, 's', { primary: true })) {
        e.preventDefault();
        handleSave();
    }

    // Ctrl/Cmd + Shift + C: 复制为代码
    if (keyboard.match(e, 'c', { primary: true, shift: true })) {
        e.preventDefault();
        copyAsCode();
    }

    // Ctrl/Cmd + K: 清空
    if (keyboard.match(e, 'k', { primary: true })) {
        e.preventDefault();
        clearContent();
    }

    // ESC: 关闭 (主应用自动处理,通常无需实现)
    if (e.key === 'Escape') {
        closePlugin();
    }
});
```

#### 常用快捷键约定

```javascript
const shortcuts = {
    // 标准编辑快捷键
    copy: { primary: true, key: 'c' },           // Ctrl/Cmd + C
    paste: { primary: true, key: 'v' },          // Ctrl/Cmd + V
    cut: { primary: true, key: 'x' },            // Ctrl/Cmd + X
    undo: { primary: true, key: 'z' },           // Ctrl/Cmd + Z
    redo: { primary: true, shift: true, key: 'z' }, // Ctrl/Cmd + Shift + Z

    // 标准操作快捷键
    save: { primary: true, key: 's' },           // Ctrl/Cmd + S
    find: { primary: true, key: 'f' },           // Ctrl/Cmd + F
    selectAll: { primary: true, key: 'a' },      // Ctrl/Cmd + A

    // 应用快捷键
    newTab: { primary: true, key: 't' },         // Ctrl/Cmd + T
    closeTab: { primary: true, key: 'w' },       // Ctrl/Cmd + W

    // 特殊功能键
    escape: { key: 'Escape' },                    // ESC
    enter: { key: 'Enter' },                      // Enter
    submit: { primary: true, key: 'Enter' },      // Ctrl/Cmd + Enter
};

// 注册快捷键
function registerShortcut(shortcut, handler) {
    document.addEventListener('keydown', (e) => {
        const primaryPressed = e.ctrlKey || e.metaKey;

        if (e.key === shortcut.key) {
            // 检查修饰键
            if (shortcut.primary && !primaryPressed) return;
            if (!shortcut.primary && primaryPressed) return;
            if (shortcut.shift !== undefined && shortcut.shift !== e.shiftKey) return;
            if (shortcut.alt !== undefined && shortcut.alt !== e.altKey) return;

            e.preventDefault();
            handler(e);
        }
    });
}

// 使用示例
registerShortcut(shortcuts.save, () => {
    console.log('保存操作 (跨平台)');
    handleSave();
});

registerShortcut(shortcuts.submit, () => {
    console.log('提交表单 (Ctrl/Cmd + Enter)');
    submitForm();
});
```

#### React 环境下的快捷键处理

```typescript
import { useEffect } from 'react';

// 自定义 Hook: 跨平台快捷键
function useShortcut(
    key: string,
    callback: (e: KeyboardEvent) => void,
    modifiers: { primary?: boolean; shift?: boolean; alt?: boolean } = {}
) {
    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if (e.key.toLowerCase() !== key.toLowerCase()) return;

            const primaryPressed = e.ctrlKey || e.metaKey;
            const shiftPressed = e.shiftKey;
            const altPressed = e.altKey;

            // 检查修饰键
            if (modifiers.primary && !primaryPressed) return;
            if (!modifiers.primary && primaryPressed) return;
            if (modifiers.shift !== undefined && modifiers.shift !== shiftPressed) return;
            if (modifiers.alt !== undefined && modifiers.alt !== altPressed) return;

            e.preventDefault();
            callback(e);
        };

        document.addEventListener('keydown', handler);
        return () => document.removeEventListener('keydown', handler);
    }, [key, callback, modifiers]);
}

// 使用示例
function MyPlugin() {
    // Ctrl/Cmd + S: 保存
    useShortcut('s', handleSave, { primary: true });

    // Ctrl/Cmd + Shift + C: 复制代码
    useShortcut('c', handleCopyCode, { primary: true, shift: true });

    // ESC: 关闭
    useShortcut('Escape', handleClose);

    return <div>Plugin Content</div>;
}
```

#### 快捷键提示 UI

```javascript
// 显示平台相关的快捷键提示
function getShortcutHint(shortcut) {
    const isMac = /Mac|iPhone|iPod|iPad/.test(navigator.platform);
    const primary = isMac ? '⌘' : 'Ctrl';

    const modifiers = [];
    if (shortcut.primary) modifiers.push(primary);
    if (shortcut.shift) modifiers.push('Shift');
    if (shortcut.alt) modifiers.push(isMac ? '⌥' : 'Alt');

    return [...modifiers, shortcut.key.toUpperCase()].join(' + ');
}

// 使用示例
const saveButton = `
  <button title="保存 (${getShortcutHint(shortcuts.save)})">
    保存
  </button>
`;

// 在 macOS 显示: 保存 (⌘ + S)
// 在 Windows 显示: 保存 (Ctrl + S)
```

#### 平台检测工具

```javascript
// 平台检测
const platform = {
    isMac: /Mac|iPhone|iPod|iPad/.test(navigator.platform),
    isWindows: /Win/.test(navigator.platform),
    isLinux: /Linux/.test(navigator.platform),

    // 获取主修饰键名称
    getPrimaryModifier: () => {
        return platform.isMac ? 'Command' : 'Ctrl';
    },

    // 获取主修饰键符号
    getPrimarySymbol: () => {
        return platform.isMac ? '⌘' : 'Ctrl';
    }
};

// 根据平台调整 UI
document.getElementById('hint').textContent =
    `按 ${platform.getPrimarySymbol()} + K 清空内容`;
```

#### 快捷键冲突避免

```javascript
// 避免与主应用快捷键冲突的建议:

// ✅ 安全的快捷键 (不太可能冲突)
// - 功能键: F1-F12
// - Ctrl/Cmd + Shift + 字母
// - Ctrl/Cmd + 数字键 (部分)

// ⚠️ 谨慎使用 (可能冲突)
// - Ctrl/Cmd + T/W/N/R (浏览器标签操作)
// - Ctrl/Cmd + Q (退出应用)
// - Ctrl/Cmd + H (隐藏窗口)

// ❌ 避免使用 (肯定冲突)
// - Alt + Space (WaTools 全局唤起快捷键)
// - ESC (主应用自动处理)

// 检测快捷键是否被占用
function isShortcutSafe(shortcut) {
    const dangerous = [
        { primary: true, key: 'q' },  // 退出
        { primary: true, key: 'w' },  // 关闭
        { alt: true, key: ' ' },      // WaTools 唤起
    ];

    return !dangerous.some(d =>
        d.primary === shortcut.primary &&
        d.alt === shortcut.alt &&
        d.key === shortcut.key
    );
}
```

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
- `CopyBase64ImageToClipboard(base64): Promise<void>` - 写图片到系统剪贴板

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

**UI 设计 (iframe 限制)**:
- [ ] Modal/Dialog 使用 absolute 定位或 inline 设计 (不依赖 fixed 覆盖整个窗口)
- [ ] Dropdown/Popover 有智能定位逻辑 (检测 iframe 边界)
- [ ] Toast 通知考虑了 max-width 限制 (calc(100vw - 40px))
- [ ] 所有覆盖层组件在 iframe 内可正常显示

**快捷键支持 (跨平台)**:
- [ ] 所有快捷键同时监听 `e.ctrlKey || e.metaKey`
- [ ] 快捷键提示 UI 根据平台显示不同符号 (⌘ vs Ctrl)
- [ ] 避免使用会与主应用冲突的快捷键 (Alt+Space, ESC)
- [ ] 提供了统一的快捷键处理函数或 Hook

**打包验证**:
- [ ] .wt 文件内容在根级别 (无父文件夹)
- [ ] 解压后打开 index.html 能在浏览器运行
- [ ] 文件总大小 < 50MB
