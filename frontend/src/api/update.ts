import {host} from "@/api/host"
import type {UpdateInfo, InstallResult} from "@/api/host"

export type {UpdateInfo, InstallResult}

export async function checkForUpdates(): Promise<UpdateInfo> {
    return host.CheckForUpdatesApi()
}

export async function downloadUpdate(): Promise<UpdateInfo> {
    return host.DownloadUpdateApi()
}

export async function installUpdate(downloadedPath: string): Promise<InstallResult> {
    return host.InstallUpdateApi(downloadedPath)
}
