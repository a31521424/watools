import {Plugin} from "@/schemas/plugin"
import {GetPluginJsEntryUrlApi, GetPluginsApi} from "../../wailsjs/go/coordinator/WaAppCoordinator"

export const getPlugins = async (): Promise<Plugin[]> => {
    const pluginsData = await GetPluginsApi()
    console.log('fetch plugins', pluginsData)

    let plugins: Plugin[] = []
    if (pluginsData) {
        plugins = pluginsData.filter(plugin => !!plugin.packageId).map((plugin: any) => ({
            packageId: plugin.packageId,
            name: plugin.name || '',
            description: plugin.description || '',
            version: plugin.version || '',
            author: plugin.author || '',
            uiEnabled: plugin.uiEnabled || false,

            enabled: plugin.enabled || false,
            storage: plugin.storage || {},
            lastUsedAt: plugin.lastUsedAt ? new Date(plugin.lastUsedAt) : new Date(0),
            usedCount: plugin.usedCount || 0,

            homeUrl: plugin.homeUrl || '',

            entry: [],
        }))
    }

    await Promise.all(plugins.map(async (plugin) => {
        try {
            const entryUrl = await GetPluginJsEntryUrlApi(plugin.packageId)
            if (entryUrl) {
                const module = await import(/* @vite-ignore */ entryUrl)
                if (module && module.default && Array.isArray(module.default)) {
                    plugin.entry = module.default
                }
            }
        } catch (error) {
            console.error(`Failed to load plugin entry for ${plugin.packageId}:`, error)
        }
    }))


    return plugins
}