import {LogDebug, LogError, LogInfo} from "@/lib/legacy-runtime";

export class Logger {
    static info = (message: string) => LogInfo(`[Fronted] ${message}`)
    static error = (message: string) => LogError(`[Fronted] ${message}`)
    static debug = (message: string) => LogDebug(`[Fronted] ${message}`)
    static warn = (message: string) => LogError(`[Fronted] ${message}`)
}