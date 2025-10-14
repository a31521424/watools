type PluginIcon = string | null

export type PluginEntry = {
    type: "executable" | "ui"
    subTitle: string
    match: (input: string) => boolean
    execute: (input: string) => Promise<void>
    icon: PluginIcon
}

export type Plugin = {
    packageId: string
    name: string
    description: string
    version: string
    author: string
    uiEnabled: boolean
    entry: PluginEntry[]
    icon: PluginIcon
    isActive: boolean
}