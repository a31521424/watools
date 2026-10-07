import type {ApplicationCommand, ClipboardContent, HostAPI, HttpResponse, OperationCommand, PluginInfo, UpdateInfo, Usage} from './contracts'
import {HostError} from './errors'
import {getTextDisplay, getTextEdges, MAX_INLINE_TEXT_LENGTH} from '@/lib/input-text'

export type MockHostOptions = {
    applications?: ApplicationCommand[]
    // A query boundary for deterministic failure/recovery scenarios. Returned
    // commands are cloned just like the instance's built-in synthetic records.
    applicationQuery?: () => Promise<ApplicationCommand[]>
    operations?: OperationCommand[]
    plugins?: PluginInfo[]
    clipboard?: ClipboardContent | null
    httpResponse?: HttpResponse
    update?: UpdateInfo
}
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T
const invalid = (message: string): never => {throw new HostError('INVALID_ARGUMENT', message)}
const unavailable = (): never => {throw new HostError('UNAVAILABLE', '浏览器模拟不支持此原生操作')}
const packageID = (id: string) => {
    if (typeof id !== 'string' || !/^watools\.plugin\.[a-z0-9][a-z0-9._-]*$/.test(id)) invalid('packageId 格式不正确')
}
const validTimestamp = (text: string): boolean => {
    const parts = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/.exec(text)
    if (!parts) return false
    const [, y, m, d, h, min, sec] = parts
    const year = Number(y), month = Number(m)
    const days = month === 2 ? ((year % 4 === 0 && year % 100 !== 0) || year % 400 === 0 ? 29 : 28) : [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31][month - 1]
    return Number(m) >= 1 && Number(m) <= 12 && Number(d) >= 1 && Number(d) <= days && Number(h) <= 23 && Number(min) <= 59 && Number(sec) <= 59 && Number.isFinite(Date.parse(text))
}
const usage = (id: string, value: Usage) => {
    if (typeof id !== 'string' || !id || typeof value.lastUsedAt !== 'string' || !validTimestamp(value.lastUsedAt)) invalid('lastUsedAt 必须是有效时间，统计对象 ID 不能为空')
    if (!Number.isInteger(value.usedCount) || value.usedCount < 0 || value.usedCount > 1e12) invalid('usedCount 必须是 0 到 1000000000000 的整数')
}
const emptyUpdate: UpdateInfo = {
    currentVersion: 'mock', latestVersion: 'mock', releaseTag: '', releaseName: '', releaseUrl: '', publishedAt: '', notes: '',
    platformKey: 'browser-mock', platformLabel: 'Browser mock', assetName: '', assetUrl: '', assetSha256: '', assetSize: 0,
    installerType: '', hasUpdate: false, downloadedPath: '', requiresManualInstall: true, message: '固定场景：当前无更新',
}
// Each instance owns its synthetic state. It never loads real credentials,
// writes disk, opens windows or makes an outgoing request.
export function createMockHost(options: MockHostOptions = {}): HostAPI {
    let applications: ApplicationCommand[] = clone<ApplicationCommand[]>(options.applications ?? [{id: 'mock-app', triggerId: 'mock-app', name: '示例应用', description: 'Host API 固定场景', category: 'Application', path: '/mock/Example.app', iconPath: '', isUserApp: true, lastUsedAt: null, usedCount: 0}])
    const operations = clone(options.operations ?? [])
    const plugins = clone(options.plugins ?? [{packageId: 'watools.plugin.mock', name: '示例插件', description: '固定模拟数据', version: '1.0.0', author: 'WaTools', usage: '', triggerKeywords: ['mock'], usageExamples: [], shortcuts: [], notes: [], uiEnabled: false, entry: '', enabled: true, lastUsedAt: null, usedCount: 0, homeUrl: ''}])
    let size = {width: 1280, height: 64}
    const storage = new Map<string, Map<string, unknown>>()
    const secrets = new Map<string, Map<string, string>>()
    const clipboardSnapshots = new Map<string, string>()
    let clipboard = clone(options.clipboard ?? null)
    let clipboardVersion = 1
    let clipboardSnapshotId = ''
    const enabled = (id: string) => {
        packageID(id)
        if (!plugins.some(p => p.packageId === id && p.enabled)) throw new HostError('PLUGIN_DISABLED', '插件未安装或已停用')
    }
    const keyAccess = (id: string, key: string) => {packageID(id); if (typeof key !== 'string' || !key) invalid('key 不能为空'); enabled(id)}
    const bucket = <T>(map: Map<string, Map<string, T>>, id: string): Map<string, T> => {
        let values = map.get(id)
        if (!values) {values = new Map(); map.set(id, values)}
        return values
    }
    return {
        GetLauncherSizeApi: async () => ({...size}),
        SetLauncherSizeApi: async (width, height) => {
            if (![width, height].every(v => Number.isInteger(v) && v > 0 && v <= 20000)) invalid('窗口尺寸必须是 1 到 20000 的整数')
            size = {width, height}
        },
        HideAppApi: async () => {}, HideOrShowAppApi: async () => {},
        GetClipboardContentApi: async () => clone(clipboard),
        SetClipboardTextApi: async text => {
            if (typeof text !== 'string') invalid('剪贴板文本必须是字符串')
            clipboard = {contentType: 'text', hasText: true, hasImage: false, hasFiles: false, text, types: ['text']}
            clipboardVersion++
            clipboardSnapshotId = ''
        },
        GetAutomaticClipboardApi: async () => {
            const text = clipboard?.contentType === 'text' ? clipboard.text ?? '' : ''
            if (!clipboardSnapshotId) clipboardSnapshotId = crypto.randomUUID().replaceAll('-', '')
            clipboardSnapshots.set(clipboardSnapshotId, text)
            while (clipboardSnapshots.size > 4) clipboardSnapshots.delete(clipboardSnapshots.keys().next().value!)
            const edges = getTextEdges(text)
            return {version: `mock-${clipboardVersion}`, snapshotId: clipboardSnapshotId, text: text.length <= MAX_INLINE_TEXT_LENGTH ? text : '', preview: getTextDisplay(text).displayValue, prefix: edges.head, suffix: edges.tail, length: text.length}
        },
        GetClipboardSnapshotTextApi: async id => {
            if (id.length !== 32) invalid('剪贴板快照标识无效')
            if (!clipboardSnapshots.has(id)) throw new HostError('UNAVAILABLE', '剪贴板快照已释放或过期，请重新复制文本')
            return clipboardSnapshots.get(id)!
        },
        ReleaseClipboardSnapshotApi: async id => {clipboardSnapshots.delete(id)},
        RetainClipboardSnapshotApi: async id => {
            if (id.length !== 32) invalid('剪贴板快照标识无效')
            if (!clipboardSnapshots.has(id)) throw new HostError('UNAVAILABLE', '剪贴板快照已释放或过期，请重新复制文本')
        },
        GetHotkeyEnvironmentStatusApi: async () => ({secureEventInputEnabled: false, note: '固定浏览器场景，不检测系统状态'}),
        GetHotkeysApi: async () => [{id: 'show-hide-window', name: '显示/隐藏窗口', hotkey: 'cmd+Space', registered: false, registeredHotkey: ''}],
        UpdateHotkeyApi: async request => {
            if (request.id !== 'show-hide-window') invalid('未知快捷键标识')
            if (typeof request.name !== 'string' || typeof request.hotkey !== 'string' || request.name.length > 256 || request.hotkey.length > 128) invalid('快捷键名称或组合无效')
            unavailable()
        },
        GetApplicationCommandsApi: async () => {
            if (options.applicationQuery) applications = clone(await options.applicationQuery())
            return clone(applications)
        },
        GetOperatorCommandsApi: async () => clone(operations),
        TriggerCommandApi: async (id, category) => {
            if (!id || !['Application', 'Operation'].includes(category)) invalid('triggerId 或 category 格式不正确')
            const commands = category === 'Application' ? applications : operations
            if (!commands.some(c => c.triggerId === id)) throw new HostError('OPERATION_FAILED', '未找到命令')
        },
        UpdateApplicationUsageApi: async items => {
            items.forEach(v => usage(v.id, v))
            items.forEach(v => {const c = applications.find(c => c.id === v.id); if (c) Object.assign(c, v)})
        },
        GetPluginsApi: async () => clone(plugins),
        DetachPluginApi: async () => unavailable(),
        GetPluginWindowApi: async () => unavailable(),
        GetPluginWindowStatusApi: async () => unavailable(),
        PluginWindowReadyApi: async () => unavailable(),
        ClosePluginWindowApi: async () => unavailable(),
        GetPluginJsEntryUrlApi: async id => {packageID(id); return ''},
        UpdatePluginUsageApi: async items => {
            items.forEach(v => {packageID(v.packageId); usage(v.packageId, v)})
            items.forEach(v => {const p = plugins.find(p => p.packageId === v.packageId); if (p) Object.assign(p, v)})
        },
        InstallPluginApi: async path => {if (!path) invalid('插件文件路径不能为空'); unavailable()},
        InstallPluginByFileDialogApi: async () => unavailable(),
        UninstallPluginApi: async id => {
            packageID(id)
            const index = plugins.findIndex(p => p.packageId === id)
            if (index < 0) throw new HostError('OPERATION_FAILED', '未找到插件')
            plugins.splice(index, 1); storage.delete(id); secrets.delete(id)
        },
        TogglePluginApi: async (id, value) => {
            packageID(id)
            const p = plugins.find(p => p.packageId === id)
            if (!p) throw new HostError('OPERATION_FAILED', '未找到插件')
            if (typeof value !== 'boolean') invalid('enabled 必须是布尔值')
            p.enabled = value
        },
        OpenFolder: async path => {if (!path) invalid('文件夹路径不能为空'); unavailable()},
        SaveBase64Image: async () => unavailable(), CopyBase64ImageToClipboard: async () => unavailable(),
        HttpProxyApi: async r => {
            let url: URL
            try {if (typeof r.url !== 'string') invalid('url 必须是有效 HTTP(S) 地址'); url = new URL(r.url)} catch {return invalid('url 必须是有效 HTTP(S) 地址')}
            if (!['http:', 'https:'].includes(url.protocol)) invalid('url 必须是有效 HTTP(S) 地址')
            if (r.timeout !== undefined && (!Number.isInteger(r.timeout) || r.timeout < 0 || r.timeout > 60000)) invalid('timeout 必须是 0 到 60000 的整数毫秒')
            if (r.method !== undefined && (typeof r.method !== 'string' || (r.method !== '' && !/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(r.method)))) invalid('HTTP method 格式不正确')
            if (r.body !== undefined && (typeof r.body !== 'string' || new TextEncoder().encode(r.body).length > 4*1024*1024)) invalid('请求体不能超过 4 MiB')
            if (r.headers && (typeof r.headers !== 'object' || Array.isArray(r.headers) || Object.values(r.headers).some(v => typeof v !== 'string'))) invalid('headers 必须是字符串映射')
            return clone(options.httpResponse ?? {status_code: 200, headers: {}, body: '{"mock":true}'})
        },
        CancelHttpProxyApi: async () => {},
        GetPluginStorageKeyApi: async r => {keyAccess(r.packageId, r.key); return clone(bucket(storage, r.packageId).get(r.key) ?? null)},
        SetPluginStorageKeyApi: async r => {
            keyAccess(r.packageId, r.key)
            let json: string | undefined
            try {json = JSON.stringify(r.value)} catch {return invalid('value 必须是有效 JSON')}
            if (json === undefined) invalid('value 不能为空；清空值请显式传 null')
            bucket(storage, r.packageId).set(r.key, JSON.parse(json!))
        },
        DeletePluginStorageKeyApi: async r => {keyAccess(r.packageId, r.key); bucket(storage, r.packageId).delete(r.key)},
        ClearPluginStorageApi: async r => {enabled(r.packageId); storage.delete(r.packageId)},
        ListPluginStorageKeysApi: async r => {enabled(r.packageId); return [...bucket(storage, r.packageId).keys()]},
        GetPluginSecretApi: async (id, key) => {keyAccess(id, key); return bucket(secrets, id).get(key) ?? ''},
        SetPluginSecretApi: async (id, key, value) => {keyAccess(id, key); if (typeof value !== 'string') invalid('value 必须是字符串'); bucket(secrets, id).set(key, value)},
        DeletePluginSecretApi: async (id, key) => {keyAccess(id, key); bucket(secrets, id).delete(key)},
        GetLogDirectoryApi: async () => '/mock/logs', ListLogFilesApi: async () => [],
        QueryLogsApi: async q => {if (q.limit !== undefined && (!Number.isInteger(q.limit) || q.limit < 0 || q.limit > 1000)) invalid('limit 必须是 0 到 1000 的整数'); return {records: [], availableSources: [], availablePluginIds: [], totalMatched: 0}},
        LogFrontendApi: async level => {if (!['debug', 'info', 'warn', 'error'].includes(level)) invalid('日志级别格式不正确')},
        GetLocalInstallationApi: async () => ({configuration: {sourceRoot:'',goExecutable:'',toolPath:'',applicationPath:'',installedAt:'',version:''},officialPlugins:[],jobId:0,state:'idle',scope:'',message:'浏览器模拟不执行本地安装',output:[],canCancel:false,restartRequired:false}),
        ConfigureLocalInstallationApi: async () => unavailable(), StartLocalInstallationApi: async () => unavailable(), CancelLocalInstallationApi: async () => unavailable(),
        CheckForUpdatesApi: async () => clone(options.update ?? emptyUpdate),
        DownloadUpdateApi: async () => unavailable(), InstallUpdateApi: async path => {if (!path) invalid('更新包路径不能为空'); return unavailable()},
    }
}
