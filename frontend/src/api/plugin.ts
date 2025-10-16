import {Plugin} from "@/schemas/plugin"
import {GetPluginsApi} from "../../wailsjs/go/coordinator/WaAppCoordinator"

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
            lastUsedTime: plugin.lastUsedTime ? new Date(plugin.lastUsedTime) : new Date(0),
            usedCount: plugin.usedCount || 0,

            entry: [],
        }))
    }


    return plugins
}