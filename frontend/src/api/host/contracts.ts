import type {LogFileInfo, LogQuery, QueryLogsResult} from '@/schemas/log'
import type {PluginContext} from '@/schemas/plugin'
import type {PluginWindowState} from '@/lib/plugin-window-state'

export type Usage = {lastUsedAt: string; usedCount: number}
export type ApplicationCommand = {triggerId: string; name: string; description: string; category: 'Application' | 'Operation'; lastUsedAt: string | null; usedCount: number; id: string; path: string; iconPath: string; isUserApp: boolean}
export type OperationCommand = Pick<ApplicationCommand, 'triggerId' | 'name' | 'description' | 'category' | 'lastUsedAt' | 'usedCount'> & {icon: string}
export type PluginInfo = {packageId: string; name: string; description: string; version: string; author: string; usage: string; triggerKeywords: string[] | null; usageExamples: string[] | null; shortcuts: string[] | null; notes: string[] | null; uiEnabled: boolean; entry: string; enabled: boolean; lastUsedAt: string | null; usedCount: number; homeUrl: string; loadError?: string}
export type ClipboardContent = {contentType: string; hasText: boolean; hasImage: boolean; hasFiles: boolean; text?: string; imageBase64?: string; files?: string[]; types: string[]}
export type AutomaticClipboard = {version: string; snapshotId: string; text: string; preview: string; prefix: string; suffix: string; length: number}
export type HttpRequest = {url: string; method?: string; headers?: Record<string, string>; body?: string; timeout?: number; requestId?: string}
export type HttpResponse = {status_code: number; headers: Record<string, string>; body: string; error?: string}
export type PackageRequest = {packageId: string}
export type StorageKeyRequest = PackageRequest & {key: string}
// Values intentionally remain dynamic JSON for compatibility with the plugin SDK.
export type StorageSetRequest = StorageKeyRequest & {value: unknown}
export type UpdateInfo = {currentVersion: string; latestVersion: string; releaseTag: string; releaseName: string; releaseUrl: string; publishedAt: string; notes: string; platformKey: string; platformLabel: string; assetName: string; assetUrl: string; assetSha256: string; assetSize: number; installerType: string; hasUpdate: boolean; downloadedPath: string; requiresManualInstall: boolean; message: string}
export type InstallResult = {status: string; message: string; openPath: string; shouldQuit: boolean; requiresManualInstall: boolean}
export type DetachPluginRequest = {packageId: string; file: string; homeUrl: string; context: PluginContext; state: PluginWindowState}
export type PluginWindowSession = DetachPluginRequest & {id: string; windowName: string; title: string}

export interface HostAPI {
 GetLauncherSizeApi(): Promise<{width: number; height: number}>
 SetLauncherSizeApi(width: number, height: number): Promise<void>
 HideAppApi(): Promise<void>
 HideOrShowAppApi(): Promise<void>
 GetClipboardContentApi(): Promise<ClipboardContent | null>
 SetClipboardTextApi(text: string): Promise<void>
 GetAutomaticClipboardApi(): Promise<AutomaticClipboard | null>
 GetClipboardSnapshotTextApi(id: string): Promise<string>
 ReleaseClipboardSnapshotApi(id: string): Promise<void>
 RetainClipboardSnapshotApi(id: string): Promise<void>
 GetHotkeyEnvironmentStatusApi(): Promise<{secureEventInputEnabled: boolean; note: string}>
 GetApplicationCommandsApi(): Promise<ApplicationCommand[]>
 GetOperatorCommandsApi(): Promise<OperationCommand[]>
 TriggerCommandApi(id: string, category: string): Promise<void>
 UpdateApplicationUsageApi(items: (Usage & {id: string})[]): Promise<void>
 GetPluginsApi(): Promise<PluginInfo[]>
 DetachPluginApi(request: DetachPluginRequest): Promise<string>
 GetPluginWindowApi(id: string): Promise<PluginWindowSession>
 GetPluginWindowStatusApi(id: string): Promise<{ready: boolean}>
 PluginWindowReadyApi(id: string): Promise<void>
 ClosePluginWindowApi(id: string): Promise<void>
 GetPluginJsEntryUrlApi(id: string): Promise<string>
 UpdatePluginUsageApi(items: (Usage & PackageRequest)[]): Promise<void>
 InstallPluginApi(path: string): Promise<void>
 InstallPluginByFileDialogApi(): Promise<void>
 UninstallPluginApi(id: string): Promise<void>
 TogglePluginApi(id: string, enabled: boolean): Promise<void>
 OpenFolder(path: string): Promise<void>
 SaveBase64Image(data: string): Promise<string>
 CopyBase64ImageToClipboard(data: string): Promise<void>
 HttpProxyApi(request: HttpRequest): Promise<HttpResponse>
 CancelHttpProxyApi(id: string): Promise<void>
 GetPluginStorageKeyApi(request: StorageKeyRequest): Promise<unknown>
 SetPluginStorageKeyApi(request: StorageSetRequest): Promise<void>
 DeletePluginStorageKeyApi(request: StorageKeyRequest): Promise<void>
 ClearPluginStorageApi(request: PackageRequest): Promise<void>
 ListPluginStorageKeysApi(request: PackageRequest): Promise<string[]>
 GetPluginSecretApi(id: string, key: string): Promise<string>
 SetPluginSecretApi(id: string, key: string, value: string): Promise<void>
 DeletePluginSecretApi(id: string, key: string): Promise<void>
 GetLogDirectoryApi(): Promise<string>
 ListLogFilesApi(): Promise<LogFileInfo[]>
 QueryLogsApi(query: LogQuery): Promise<QueryLogsResult>
 LogFrontendApi(level: string, message: string): Promise<void>
 CheckForUpdatesApi(): Promise<UpdateInfo>
 DownloadUpdateApi(): Promise<UpdateInfo>
 InstallUpdateApi(path: string): Promise<InstallResult>
}
