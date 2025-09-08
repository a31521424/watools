export type PluginEntry = {
    entryID: string
    exec?: (input: string) => void
    render?: (container: Element, input: string) => void
    match: (input: string) => boolean
    title: string
    icon?: string
}

export type PluginMetadata = {
    id: string
    packageID: string
    name: string
    version: string
    description: string
    author: string
}

export type PluginPackage = {
    metadata: PluginMetadata
    allEntries: PluginEntry[]
}