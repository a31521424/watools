import {create} from 'zustand'
import {Plugin} from '@/schemas/plugin'
import {getPlugins, updatePluginUsage} from "@/api/plugin";
import {Logger} from "@/lib/logger";

interface PluginState {
    plugins: Plugin[]
    isLoading: boolean
    error: string | null
    fetchPlugins: () => Promise<void>
    refreshPlugins: () => Promise<void>
    fetchPluginsAsync: () => void  // fire-and-forget version
    getPluginById: (packageId: string) => Plugin | undefined
    getEnabledPlugins: () => Plugin[]
    getPluginsByType: (type: "executable" | "ui") => Plugin[]
    updatePluginUsage: (packageId: string) => Promise<void>
}

export const usePluginStore = create<PluginState>((set, get) => ({
    plugins: [],
    isLoading: false,
    error: null,

    fetchPlugins: async () => {
        set({isLoading: true, error: null})
        try {
            const plugins = await getPlugins()
            set({plugins, isLoading: false})
        } catch (error) {
            Logger.error(`Failed to fetch plugins: ${error}`)
            set({
                error: error instanceof Error ? error.message : 'Failed to fetch plugins',
                isLoading: false
            })
        }
    },

    refreshPlugins: async () => {
        await get().fetchPlugins()
    },

    fetchPluginsAsync: () => {
        // Fire-and-forget version with error handling
        get().fetchPlugins().catch(error => {
            console.error('Background plugin fetch failed:', error)
        })
    },

    getPluginById: (packageId: string) => {
        return get().plugins.find(plugin => plugin.packageId === packageId)
    },

    getEnabledPlugins: () => {
        return get().plugins.filter(plugin => plugin.enabled)
    },

    getPluginsByType: (type: "executable" | "ui") => {
        return get().plugins.filter(plugin =>
            plugin.entry.some(entry => entry.type === type)
        )
    },

    updatePluginUsage: async (packageId: string) => {
        const plugin = get().getPluginById(packageId);
        if (!plugin) return;

        const now = new Date();
        const newUsedCount = plugin.usedCount + 1;

        // Update local state immediately for UI responsiveness
        set(state => ({
            plugins: state.plugins.map(p =>
                p.packageId === packageId
                    ? { ...p, usedCount: newUsedCount, lastUsedAt: now }
                    : p
            )
        }));

        try {
            await updatePluginUsage([{
                packageId,
                lastUsedAt: now,
                usedCount: newUsedCount
            }]);
        } catch (error) {
            Logger.error(`Failed to update plugin usage: ${error}`);
            // Revert on error
            set(state => ({
                plugins: state.plugins.map(p =>
                    p.packageId === packageId
                        ? { ...p, usedCount: plugin.usedCount, lastUsedAt: plugin.lastUsedAt }
                        : p
                )
            }));
        }
    }
}))