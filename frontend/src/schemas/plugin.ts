import type {WaToolsApi} from "@/api/api";
import {AppClipboardContent, AppInput} from "@/schemas/app";

type PluginIcon = string | null

/**
 * Complete context passed to execute and UI; also used by bounded legacy match.
 */
export type PluginContext = {
    input: AppInput
    clipboard: AppClipboardContent | null
}

/** Bounded candidate matching data. Never contains origin, assets or host APIs. */
export type PluginMatchContext = {
    readonly input: {
        readonly valueType: AppInput['valueType']
        readonly clipboardContentType?: AppInput['clipboardContentType']
        readonly length: number
        readonly head: string
        readonly tail: string
        readonly truncated: boolean
    }
    readonly clipboard: {readonly contentType: AppClipboardContent['contentType']; readonly fileCount: number} | null
}

type PluginEntryMetadata = {
    subTitle: string
    icon: PluginIcon
}

/** Public plugin declaration. Legacy fields are interpreted only at load time. */
export type PluginEntryDefinition = PluginEntryMetadata & {
    /** Preferred for all input sizes. head/tail each contain at most 128 UTF-16 units. */
    matchPreview?: (context: PluginMatchContext) => boolean
    /** Legacy compatibility: invoked only for text of at most 2000 UTF-16 units. */
    match?: (context: PluginContext) => boolean
    /** Opt-in, bounded prefix rules for executable entries with large text. */
    largeTextPrefixes?: readonly string[]
} & (
    {type: "ui"; file: string}
    | {type: "executable"; execute: (context: PluginContext, api?: WaToolsApi) => Promise<void>}
)

export type PluginMatcher =
    | {kind: 'preview'; match: (context: PluginMatchContext) => boolean}
    | {kind: 'legacy'; match: (context: PluginContext) => boolean; largeTextPrefixes: readonly string[]}

/** Validated host entry: one matcher and the payload required by its type. */
export type PluginEntry = PluginEntryMetadata & {matcher: PluginMatcher} & (
    {type: "ui"; file: string}
    | {type: "executable"; execute: (context: PluginContext, api?: WaToolsApi) => Promise<void>}
)

export type Plugin = {
    packageId: string
    name: string
    description: string
    version: string
    author: string
    usage: string
    triggerKeywords: string[]
    usageExamples: string[]
    shortcuts: string[]
    notes: string[]
    uiEnabled: boolean

    enabled: boolean
    loadStatus?: 'disabled' | 'ready' | 'failed'
    loadError?: string
    storage: Record<string, any>
    lastUsedAt: Date | null
    usedCount: number

    homeUrl: string

    entry: PluginEntry[]
}
