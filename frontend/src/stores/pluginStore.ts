import {create} from 'zustand'
import {Plugin} from '@/schemas/plugin'
import {getPlugins, updatePluginUsage} from "@/api/plugin";
import {Logger} from "@/lib/logger";

interface PluginState {
    plugins: Plugin[]
    isLoading: boolean
    error: string | null
    updateBuffer: Map<string, { lastUsedAt: Date, usedCount: number }>
    fetchPlugins: () => Promise<void>
    refreshPlugins: () => Promise<void>
    getPluginById: (packageId: string) => Plugin | undefined
    getEnabledPlugins: () => Plugin[]
    getPluginsByType: (type: "executable" | "ui") => Plugin[]
    updatePluginUsage: (packageId: string) => Promise<void>
    flushBufferUpdates: () => Promise<void>
}

const DEBOUNCE_DELAY = 60000

export const usePluginStore = create<PluginState>((set, get) => {
    let debounceTimer: ReturnType<typeof setTimeout> | null = null

    const debouncedFlushUpdates = () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer)
        }

        debounceTimer = setTimeout(async () => {
            const { updateBuffer } = get()

            if (updateBuffer.size === 0) return

            try {
                const updates = Array.from(updateBuffer.entries()).map(([packageId, data]) => ({
                    packageId,
                    lastUsedAt: data.lastUsedAt,
                    usedCount: data.usedCount
                }))

                await updatePluginUsage(updates)
                set({ updateBuffer: new Map() })
            } catch (error) {
                Logger.error(`Failed to flush plugin buffer updates: ${error}`)
            }
        }, DEBOUNCE_DELAY)
    }

    return {
        plugins: [],
        isLoading: false,
        error: null,
        updateBuffer: new Map(),

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
                ),
                updateBuffer: new Map(state.updateBuffer).set(packageId, {
                    lastUsedAt: now,
                    usedCount: newUsedCount
                })
            }));

            // Trigger debounced batch update
            debouncedFlushUpdates();
        },

        flushBufferUpdates: async () => {
            if (debounceTimer) {
                clearTimeout(debounceTimer);
                debounceTimer = null;
            }

            const { updateBuffer } = get();
            if (updateBuffer.size === 0) return;

            try {
                const updates = Array.from(updateBuffer.entries()).map(([packageId, data]) => ({
                    packageId,
                    lastUsedAt: data.lastUsedAt,
                    usedCount: data.usedCount
                }));

                await updatePluginUsage(updates);
                set({ updateBuffer: new Map() });
            } catch (error) {
                Logger.error(`Failed to flush plugin buffer updates: ${error}`);
            }
        }
    }
})