import {host} from "@/api/host"

export type WaToolsApi = {
    OpenFolder: typeof host.OpenFolder;
    SaveBase64Image: typeof host.SaveBase64Image;
    CopyBase64ImageToClipboard: typeof host.CopyBase64ImageToClipboard;
    HttpProxy: typeof host.HttpProxyApi;
    CancelHttpProxy: typeof host.CancelHttpProxyApi;
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
    OpenFolder: value => host.OpenFolder(value),
    SaveBase64Image: value => host.SaveBase64Image(value),
    CopyBase64ImageToClipboard: value => host.CopyBase64ImageToClipboard(value),
    HttpProxy: request => host.HttpProxyApi(request),
    CancelHttpProxy: id => host.CancelHttpProxyApi(id),
    SecretGet: key => host.GetPluginSecretApi(packageId,key),
    SecretSet: (key,value) => host.SetPluginSecretApi(packageId,key,value),
    SecretRemove: key => host.DeletePluginSecretApi(packageId,key),
    StorageGet: (key: string) => host.GetPluginStorageKeyApi({packageId, key}) as Promise<any>,
    StorageSet: (key: string, value: any) => host.SetPluginStorageKeyApi({packageId, key, value}),
    StorageRemove: (key: string) => host.DeletePluginStorageKeyApi({packageId, key}),
    StorageClear: () => host.ClearPluginStorageApi({packageId}),
    StorageKeys: () => host.ListPluginStorageKeysApi({packageId}),
})

// Default instance for main window (empty packageId)
export const WaApi: WaToolsApi = createWaToolsApi('')
