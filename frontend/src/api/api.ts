import {
    HttpProxyApi,
    OpenFolder,
    SaveBase64Image,
    PluginStorageGetApi,
    PluginStorageSetApi,
    PluginStorageRemoveApi,
    PluginStorageClearApi,
    PluginStorageKeysApi
} from "../../wailsjs/go/coordinator/WaAppCoordinator";

// Get plugin package ID from window (injected by wa-plugin.tsx)
const getPluginPackageId = (): string => {
    // @ts-ignore
    return window.__PLUGIN_PACKAGE_ID__ || ''
}

export const WaApi = {
    OpenFolder,
    SaveBase64Image,
    HttpProxy: HttpProxyApi,

    // Plugin storage API
    storage: {
        get: async (key: string): Promise<any> => {
            const packageId = getPluginPackageId()
            return PluginStorageGetApi({packageId, key})
        },
        set: async (key: string, value: any): Promise<void> => {
            const packageId = getPluginPackageId()
            return PluginStorageSetApi({packageId, key, value})
        },
        remove: async (key: string): Promise<void> => {
            const packageId = getPluginPackageId()
            return PluginStorageRemoveApi({packageId, key})
        },
        clear: async (): Promise<void> => {
            const packageId = getPluginPackageId()
            return PluginStorageClearApi({packageId})
        },
        keys: async (): Promise<string[]> => {
            const packageId = getPluginPackageId()
            return PluginStorageKeysApi({packageId})
        }
    }
}
