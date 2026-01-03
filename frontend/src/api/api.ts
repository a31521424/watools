import {
    HttpProxyApi,
    OpenFolder,
    SaveBase64Image,
    GetPluginStorageKeyApi,
    SetPluginStorageKeyApi,
    DeletePluginStorageKeyApi,
    ClearPluginStorageApi,
    ListPluginStorageKeysApi
} from "../../wailsjs/go/coordinator/WaAppCoordinator";

// Get plugin package ID from window (injected by wa-plugin.tsx)
const getPluginPackageId = (): string => {
    // @ts-ignore
    return window.__PLUGIN_PACKAGE_ID__ || ''
}

// Wrapper functions that auto-inject packageId
const StorageGet = async (key: string): Promise<any> => {
    const packageId = getPluginPackageId()
    return GetPluginStorageKeyApi({packageId, key})
}

const StorageSet = async (key: string, value: any): Promise<void> => {
    const packageId = getPluginPackageId()
    return SetPluginStorageKeyApi({packageId, key, value})
}

const StorageRemove = async (key: string): Promise<void> => {
    const packageId = getPluginPackageId()
    return DeletePluginStorageKeyApi({packageId, key})
}

const StorageClear = async (): Promise<void> => {
    const packageId = getPluginPackageId()
    return ClearPluginStorageApi({packageId})
}

const StorageKeys = async (): Promise<string[]> => {
    const packageId = getPluginPackageId()
    return ListPluginStorageKeysApi({packageId})
}

export const WaApi = {
    OpenFolder,
    SaveBase64Image,
    HttpProxy: HttpProxyApi,
    StorageGet,
    StorageSet,
    StorageRemove,
    StorageClear,
    StorageKeys,
}
