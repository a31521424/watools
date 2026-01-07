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

export type WaToolsApi = {
    OpenFolder: typeof OpenFolder;
    SaveBase64Image: typeof SaveBase64Image;
    HttpProxy: typeof HttpProxyApi;
    StorageGet: (key: string) => Promise<any>;
    StorageSet: (key: string, value: any) => Promise<void>;
    StorageRemove: (key: string) => Promise<void>;
    StorageClear: () => Promise<void>;
    StorageKeys: () => Promise<string[]>;
}

// Factory function to create WaToolsApi with explicit packageId
export const createWaToolsApi = (packageId: string): WaToolsApi => ({
    OpenFolder,
    SaveBase64Image,
    HttpProxy: HttpProxyApi,
    StorageGet: (key: string) => GetPluginStorageKeyApi({packageId, key}),
    StorageSet: (key: string, value: any) => SetPluginStorageKeyApi({packageId, key, value}),
    StorageRemove: (key: string) => DeletePluginStorageKeyApi({packageId, key}),
    StorageClear: () => ClearPluginStorageApi({packageId}),
    StorageKeys: () => ListPluginStorageKeysApi({packageId}),
})

// Default instance for main window (empty packageId)
export const WaApi: WaToolsApi = createWaToolsApi('')
