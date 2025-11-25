import {AppClipboardContent, AppInput} from "@/schemas/app";

type PluginIcon = string | null

/**
 * Context object passed to plugin match and execute functions
 */
export type PluginContext = {
    input: AppInput
    clipboard: AppClipboardContent | null
}

/**
 * Plugin entry point definition
 * - match: Determines if this plugin should handle the current input
 * - execute: Executes the plugin action (required for "executable" type)
 * - file: UI path for iframe loading (required for "ui" type)
 */
export type PluginEntry = {
    type: "executable" | "ui"
    subTitle: string
    match: (context: PluginContext) => boolean
    execute?: (context: PluginContext) => Promise<void>
    icon: PluginIcon
    file?: string
}

export type Plugin = {
    packageId: string
    name: string
    description: string
    version: string
    author: string
    uiEnabled: boolean

    enabled: boolean
    storage: Record<string, any>
    lastUsedAt: Date | null
    usedCount: number

    homeUrl: string

    entry: PluginEntry[]
}