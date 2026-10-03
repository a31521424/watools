import {Plugin} from "@/schemas/plugin"
import {host} from "@/api/host"
import {sanitizePluginEntries} from "@/lib/plugin";

const normalizeStringList = (value: unknown): string[] => {
    if (!Array.isArray(value)) {
        return []
    }

    return value
        .filter((item): item is string => typeof item === 'string')
        .map(item => item.trim())
        .filter(Boolean)
}

const dedupePluginsByPackageId = (plugins: Plugin[]): Plugin[] => {
    const uniquePlugins = new Map<string, Plugin>()
    for (const plugin of plugins) {
        if (!uniquePlugins.has(plugin.packageId)) {
            uniquePlugins.set(plugin.packageId, plugin)
        }
    }
    return Array.from(uniquePlugins.values())
}

export const getPlugins = async (): Promise<Plugin[]> => {
    const pluginsData = await host.GetPluginsApi()

    let plugins: Plugin[] = []
    if (pluginsData) {
        plugins = pluginsData.filter(plugin => plugin && typeof plugin.packageId === 'string' && plugin.packageId).map(plugin => ({
            packageId: plugin.packageId,
            name: plugin.name || '',
            description: plugin.description || '',
            version: plugin.version || '',
            author: plugin.author || '',
            usage: plugin.usage || '',
            triggerKeywords: normalizeStringList(plugin.triggerKeywords),
            usageExamples: normalizeStringList(plugin.usageExamples),
            shortcuts: normalizeStringList(plugin.shortcuts),
            notes: normalizeStringList(plugin.notes),
            uiEnabled: plugin.uiEnabled || false,

            enabled: plugin.enabled || false,
            loadStatus: plugin.enabled ? 'failed' : 'disabled',
            loadError: typeof plugin.loadError === 'string' ? plugin.loadError : undefined,
            storage: {},
            lastUsedAt: plugin.lastUsedAt ? new Date(plugin.lastUsedAt) : new Date(0),
            usedCount: plugin.usedCount || 0,

            homeUrl: plugin.homeUrl || '',

            entry: [],
        }))
    }

    plugins = dedupePluginsByPackageId(plugins)

    const enabled = plugins.filter(plugin => plugin.enabled && !plugin.loadError)
    for (let offset = 0; offset < enabled.length; offset += 4) {
        await Promise.all(enabled.slice(offset, offset + 4).map(loadPluginEntries))
    }

    return plugins
}

const pluginDisposers = new Map<string, () => unknown>()

export const disposePlugin = async (packageId: string) => {
    const dispose = pluginDisposers.get(packageId)
    pluginDisposers.delete(packageId)
    if (dispose) {
        let timer: ReturnType<typeof setTimeout> | undefined
        try { await Promise.race([Promise.resolve().then(dispose), new Promise((_, reject) => {
            timer = setTimeout(() => reject(new Error('插件清理超时')), 3000)
        })]) } catch (error) { console.error(`Plugin cleanup failed: ${packageId}`, error) }
        finally { if (timer !== undefined) clearTimeout(timer) }
    }
}

export const loadPluginEntries = async (plugin: Plugin): Promise<void> => {
    let timer: ReturnType<typeof setTimeout> | undefined
    try {
        const pending = (async () => {
            const entryUrl = await host.GetPluginJsEntryUrlApi(plugin.packageId)
            if (!entryUrl) throw new Error('插件入口不存在或已停用')
            const module = await import(/* @vite-ignore */ entryUrl)
            if (!Array.isArray(module.default)) throw new Error('插件入口格式无效')
            const entry = await sanitizePluginEntries(plugin, module.default)
            return {entry, dispose: typeof module.dispose === 'function' ? module.dispose : undefined}
        })()
        const loaded = await Promise.race([pending, new Promise<never>((_, reject) => {
            timer = setTimeout(() => reject(new Error('插件加载超时')), 10000)
        })])
        plugin.entry = loaded.entry
        plugin.loadStatus = 'ready'
        plugin.loadError = undefined
        if (loaded.dispose) pluginDisposers.set(plugin.packageId, loaded.dispose)
    } catch (error) {
        plugin.entry = []
        plugin.loadStatus = 'failed'
        plugin.loadError = error instanceof Error ? error.message : '插件加载失败'
    } finally {
        if (timer !== undefined) clearTimeout(timer)
    }
}

export const updatePluginUsage = async (updates: Array<{ packageId: string; lastUsedAt: Date; usedCount: number }>) => {
    const formattedUpdates = updates.map(update => ({
        packageId: update.packageId,
        lastUsedAt: update.lastUsedAt.toISOString(),
        usedCount: update.usedCount
    }));
    return host.UpdatePluginUsageApi(formattedUpdates);
}

export const installPlugin = async (wtFilePath: string) => {
    return host.InstallPluginApi(wtFilePath)
}

export const uninstallPlugin = async (packageId: string) => {
    return host.UninstallPluginApi(packageId)
}

export const togglePlugin = async (packageId: string, enabled: boolean) => {
    return host.TogglePluginApi(packageId, enabled)
}
