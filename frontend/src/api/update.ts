import {
    CheckForUpdatesApi,
    DownloadUpdateApi,
    InstallUpdateApi,
} from "../../wailsjs/go/coordinator/WaAppCoordinator"
import {update} from "../../wailsjs/go/models"

export type UpdateInfo = update.UpdateInfo
export type InstallResult = update.InstallResult

export async function checkForUpdates(): Promise<UpdateInfo> {
    return update.UpdateInfo.createFrom(await CheckForUpdatesApi())
}

export async function downloadUpdate(): Promise<UpdateInfo> {
    return update.UpdateInfo.createFrom(await DownloadUpdateApi())
}

export async function installUpdate(downloadedPath: string): Promise<InstallResult> {
    return update.InstallResult.createFrom(await InstallUpdateApi(downloadedPath))
}
