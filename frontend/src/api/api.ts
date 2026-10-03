import {
    CopyBase64ImageToClipboard,
    HttpProxyApi,
    CancelHttpProxyApi,
    GetPluginSecretApi,
    SetPluginSecretApi,
    DeletePluginSecretApi,
    OpenFolder,
    SaveBase64Image,
    GetPluginStorageKeyApi,
    SetPluginStorageKeyApi,
    DeletePluginStorageKeyApi,
    ClearPluginStorageApi,
    ListPluginStorageKeysApi
} from "../../bindings/watools/internal/coordinator/waappcoordinator";

export type WaToolsApi = {
    OpenFolder: typeof OpenFolder;
    SaveBase64Image: typeof SaveBase64Image;
    CopyBase64ImageToClipboard: typeof CopyBase64ImageToClipboard;
    HttpProxy: typeof HttpProxyApi;
    CancelHttpProxy: typeof CancelHttpProxyApi;
    SecretGet: (key: string) => Promise<string>;
    SecretSet: (key: string, value: string) => Promise<void>;
    SecretRemove: (key: string) => Promise<void>;
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
    CopyBase64ImageToClipboard,
    HttpProxy: HttpProxyApi,
    CancelHttpProxy: CancelHttpProxyApi,
    SecretGet: key => GetPluginSecretApi(packageId,key),
    SecretSet: (key,value) => SetPluginSecretApi(packageId,key,value),
    SecretRemove: key => DeletePluginSecretApi(packageId,key),
    StorageGet: (key: string) => GetPluginStorageKeyApi({packageId, key}),
    StorageSet: (key: string, value: any) => SetPluginStorageKeyApi({packageId, key, value}),
    StorageRemove: (key: string) => DeletePluginStorageKeyApi({packageId, key}),
    StorageClear: () => ClearPluginStorageApi({packageId}),
    StorageKeys: () => ListPluginStorageKeysApi({packageId}),
})

// Default instance for main window (empty packageId)
export const WaApi: WaToolsApi = createWaToolsApi('')
