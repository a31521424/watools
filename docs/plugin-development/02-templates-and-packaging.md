# 插件开发 02: 模板与打包

## 简单模式模板

适用场景: 简单 UI、计算器、文本处理等。

**文件清单**:

```text
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

**打包**:

```bash
cd watools.plugin.example
zip -r ../example.wt manifest.json app.js index.html
```

## 构建模式模板

适用场景: React / TypeScript / Vue 等复杂 UI。

**项目结构**:

```text
my-plugin/
├── src/
│   ├── App.tsx
│   └── main.tsx
├── public/
│   ├── manifest.json
│   └── app.js
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

**package.json**:

```json
{
  "name": "my-plugin",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "package": "cd dist && zip -r ../my-plugin.wt * && cd .."
  }
}
```

**vite.config.ts**:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    build: {
        outDir: 'dist'
    },
    publicDir: 'public'
})
```

**构建和打包**:

```bash
npm install
npm run build
npm run package
```

## 核心配置参考

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

约束:

- `packageId` 必须以 `watools.plugin.` 开头
- `uiEnabled: true` 表示包含 UI 插件
- `entry` 固定为 `app.js`

### app.js

```javascript
const entry = [
    {
        type: "ui" | "executable",
        subTitle: "操作描述",
        icon: "icon-name" | "🔢" | null,
        match: (context) => boolean,
        file: "index.html",
        execute: async (context) => {}
    }
];

export default entry;
```

约束:

- 可导出多个 entry
- `match` 必须同步返回 boolean
- `execute` 必须返回 Promise
- `icon` 可为 Lucide icon 名称、Emoji 或 `null`

### index.html

- 使用标准 HTML5 文档
- 使用 `<script type="module">`
- 新 UI 插件统一从 `window.pluginContext` 读取启动上下文

## 常见错误

### 错误 1: .wt 文件包含源代码

错误:

```text
plugin.wt/
├── src/
├── node_modules/
├── package.json
└── manifest.json
```

解决:

```bash
cd dist && zip -r ../plugin.wt *
```

### 错误 2: `manifest.json` 和 `app.js` 在子目录

错误:

```text
plugin.wt/
└── public/
    ├── manifest.json
    └── app.js
```

解决:

- 使用 `publicDir: 'public'`
- 保证打包的是 `dist/` 根内容

### 错误 3: 浏览器调试时 API 崩溃

错误:

```javascript
await window.runtime.ClipboardSetText(text);
```

解决:

```javascript
const setText = async (text) => {
    if (window.runtime?.ClipboardSetText) {
        return await window.runtime.ClipboardSetText(text);
    }
    await navigator.clipboard.writeText(text);
};
```

### 错误 4: 使用浏览器原生弹窗

错误:

```javascript
alert('操作成功');
confirm('确定删除?');
```

解决:

- 使用自定义 toast
- 使用 inline dialog 或插件内 modal
