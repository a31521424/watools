type PluginIcon = string | null

export type PluginEntry = {
    type: "executable" | "ui"
    subTitle: string
    match: (input: string) => boolean
    execute?: (input: string) => Promise<void>
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
    lastUsedTime: Date | null
    usedCount: number

    homeUrl: string

    entry: PluginEntry[]
}