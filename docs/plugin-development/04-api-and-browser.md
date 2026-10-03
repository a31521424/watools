# 插件开发 04: API 与浏览器限制

## `window.runtime`

```typescript
EventsEmit(eventName: string, ...data: any): void
EventsOn(eventName: string, callback: (...data: any) => void): () => void
EventsOnce(eventName: string, callback: (...data: any) => void): () => void
EventsOff(eventName: string): void

LogTrace(message: string): void
LogDebug(message: string): void
LogInfo(message: string): void
LogWarning(message: string): void
LogError(message: string): void
LogFatal(message: string): void

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

ClipboardGetText(): Promise<string>
ClipboardSetText(text: string): Promise<boolean>

BrowserOpenURL(url: string): void
Quit(): void
Hide(): void
Show(): void
Environment(): Promise<{buildType: string, platform: string, arch: string}>

OnFileDrop(callback: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean): void
OnFileDropOff(): void
```

## `window.watools`

```typescript
HttpProxy(request: HttpProxyRequest): Promise<HttpProxyResponse>
CancelHttpProxy(requestId: string): Promise<void>
SecretGet(key: string): Promise<string>
SecretSet(key: string, value: string): Promise<void>
SecretRemove(key: string): Promise<void>
OpenFolder(folderPath: string): Promise<void>
SaveBase64Image(base64String: string): Promise<string>
CopyBase64ImageToClipboard(base64String: string): Promise<void>
StorageGet(key: string): Promise<any>
StorageSet(key: string, value: any): Promise<void>
StorageRemove(key: string): Promise<void>
StorageClear(): Promise<void>
StorageKeys(): Promise<string[]>
```

## 网络请求与敏感数据

`HttpProxyRequest` 可提供 `requestId`；取消时调用 `CancelHttpProxy(requestId)`。请求体最多 4 MiB，响应体最多 8 MiB，超时最多 60000 ms，宿主最多同时执行 8 个请求。退出会取消请求。HTTP(S) 的 localhost 和内网访问仍属于当前可信插件能力。

API 密钥等敏感值使用 `SecretGet/Set/Remove`，不要放入 Storage、localStorage、URL 或日志。普通 Storage 与浏览器存储可以保存非敏感偏好。SecretGet 未设置时返回空字符串；宿主存储失败时 Promise 会拒绝，不额外复制到浏览器存储。

macOS 按用户选择使用 `<data>/secrets/<account>.local` 本地明文文件，目录权限 0700、文件权限 0600，不访问 Keychain；Windows 继续使用当前用户范围的 DPAPI。Secret API 的名称不表示所有平台都加密落盘。package scope 标识数据归属，不限制同源插件访问其他宿主 API；当前插件必须可信。`execute(context, api)` 支持显式 API 参数，旧 `window.watools` 方式仍兼容并串行执行。

## API 包装模板

```javascript
export const api = {
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
            if (window.watools?.CopyBase64ImageToClipboard) {
                await window.watools.CopyBase64ImageToClipboard(base64String);
                return true;
            }
            throw new Error('Image clipboard is not available in current context');
        }
    },
    http: async (request) => {
        if (window.watools?.HttpProxy) {
            return await window.watools.HttpProxy(request);
        }
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
    }
};
```

## 浏览器原生 API 限制

### 禁止使用

- `alert` / `confirm` / `prompt`
- `window.open`
- `window.close`
- `window.resizeTo`
- `window.moveTo`
- File System Access API: `showOpenFilePicker` / `showSaveFilePicker` / `showDirectoryPicker`

### 受限但可降级

- `fetch` 跨域: 推荐 `window.watools.HttpProxy`
- `navigator.clipboard.write()` 图片: 推荐 `window.watools.CopyBase64ImageToClipboard`
- `localStorage`: 可用,但推荐宿主 `StorageXxx`

### 可安全使用

- DOM API
- `setTimeout` / `requestAnimationFrame`
- `JSON` / `URLSearchParams` / `FormData`
- `<canvas>` / 2D context
- `<audio>` / `<video>`
- Drag and Drop API

## 快速参考表

| API 类型 | 状态 | 替代方案 |
|---------|------|---------|
| alert/confirm/prompt | 禁止 | 自定义 UI |
| window.open() | 禁止 | `window.runtime.BrowserOpenURL()` |
| File System Access API | 不可用 | 拖拽 或 `<input type="file">` |
| fetch (跨域) | 受限 | `window.watools.HttpProxy()` |
| localStorage | 可用 | `window.watools.StorageXxx` |
| 剪贴板写文本 | 可用 | `window.runtime.ClipboardSetText()` |
| 剪贴板写图片 | 常受限 | `window.watools.CopyBase64ImageToClipboard()` |
