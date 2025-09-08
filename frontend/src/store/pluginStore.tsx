import {create} from "zustand";
import {PluginEntry, PluginPackage} from "@/schemas/plugin";

export type PluginState = {
    plugins: PluginPackage[]
    setPlugins: (plugins: PluginPackage[]) => void
    getPlugin: (id: string) => PluginPackage | undefined
    getPlugins: () => PluginPackage[]
    getPluginEntry: (entryID: string) => PluginEntry | undefined
}

export const usePluginStore = create<PluginState>((set, get) => ({
    plugins: [],
    setPlugins: (plugins: PluginPackage[]) => set({plugins}),
    getPlugin: (id: string) => get().plugins.find(plugin => plugin.metadata.id === id),
    getPlugins: () => get().plugins,
    getPluginEntry: (entryID: string) => get().plugins.flatMap(plugin => plugin.allEntries).find(entry => entry.entryID === entryID),
}))

export const usePluginActions = () => {
    const setPlugins = usePluginStore(state => state.setPlugins)
    const getPlugin = usePluginStore(state => state.getPlugin)
    const getPlugins = usePluginStore(state => state.getPlugins)
    const getPluginEntry = usePluginStore(state => state.getPluginEntry)
    return {
        setPlugins,
        getPlugin,
        getPlugins,
        getPluginEntry,
    }
}