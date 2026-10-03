import {
    CheckForUpdatesApi,
    DownloadUpdateApi,
    InstallUpdateApi,
} from "../../bindings/watools/internal/coordinator/waappcoordinator"
import {UpdateInfo, InstallResult} from "../../bindings/watools/internal/update/models"

export type {UpdateInfo, InstallResult}

export async function checkForUpdates(): Promise<UpdateInfo> {
    return UpdateInfo.createFrom(await CheckForUpdatesApi())
}

export async function downloadUpdate(): Promise<UpdateInfo> {
    return UpdateInfo.createFrom(await DownloadUpdateApi())
}

export async function installUpdate(downloadedPath: string): Promise<InstallResult> {
    return InstallResult.createFrom(await InstallUpdateApi(downloadedPath))
}
