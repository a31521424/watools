import type {HostAPI} from './contracts'
import {wailsHost} from './wails'
import {createMockHost} from './mock'

// The default dev/browser/desktop flow still uses the real Go host.
export let host: HostAPI = import.meta.env.VITE_WATOOLS_HOST === 'mock' ? createMockHost() : wailsHost
export function setHostAPI(api: HostAPI): void { host = api }
export type {HostAPI, UpdateInfo, InstallResult} from './contracts'
export {HostError} from './errors'
