import React from "react";

export type PluginEntry = {
    exec?: (input: string) => void
    ui?: (input: string) => React.ReactNode
    match: (input: string) => boolean
    title: string
    icon?: string
}
export type PluginPackage = {
    allEntries: PluginEntry[]
}