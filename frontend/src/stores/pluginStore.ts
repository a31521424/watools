import {create} from 'zustand'
import {Plugin} from '@/schemas/plugin'
import {getPlugins, updatePluginUsage, togglePlugin as togglePluginApi, uninstallPlugin as uninstallPluginApi} from "@/api/plugin";
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
    togglePlugin: (packageId: string, enabled: boolean) => Promise<void>
    uninstallPlugin: (packageId: string) => Promise<void>
}

const DEBOUNCE_DELAY = 60000

export const usePluginStore = create<PluginState>((set, get) => {
    let debounceTimer: ReturnType<typeof setTimeout> | null = null
    let isInitialized = false

    const debouncedFlushUpdates = () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer)
        }

        debounceTimer = setTimeout(async () => {
            const {updateBuffer} = get()

            if (updateBuffer.size === 0) return

            try {
                const updates = Array.from(updateBuffer.entries()).map(([packageId, data]) => ({
                    packageId,
                    lastUsedAt: data.lastUsedAt,
                    usedCount: data.usedCount
                }))

                await updatePluginUsage(updates)
                set({updateBuffer: new Map()})
            } catch (error) {
                Logger.error(`Failed to flush plugin buffer updates: ${error}`)
            }
        }, DEBOUNCE_DELAY)
    }

    const fetchPlugins = async () => {
        if (isInitialized) return

        set({isLoading: true, error: null})
        try {
            const plugins = await getPlugins()
            set({plugins, isLoading: false})
            isInitialized = true
            console.log('fetched plugins', plugins)
        } catch (error) {
            Logger.error(`Failed to fetch plugins: ${error}`)
            set({
                error: error instanceof Error ? error.message : 'Failed to fetch plugins',
                isLoading: false
            })
        }
    }

    const refreshPlugins = async () => {
        isInitialized = false
        await fetchPlugins()
    }

    const getPluginById = (packageId: string) => {
        return get().plugins.find(plugin => plugin.packageId === packageId)
    }

    const getEnabledPlugins = () => {
        return get().plugins.filter(plugin => plugin.enabled)
    }

    const getPluginsByType = (type: "executable" | "ui") => {
        return get().plugins.filter(plugin =>
            plugin.entry.some(entry => entry.type === type)
        )
    }

    const updatePluginUsageMethod = async (packageId: string) => {
        const plugin = getPluginById(packageId);
        if (!plugin) return;

        const now = new Date();
        const newUsedCount = plugin.usedCount + 1;

        // Update local state immediately for UI responsiveness
        set(state => ({
            plugins: state.plugins.map(p =>
                p.packageId === packageId
                    ? {...p, usedCount: newUsedCount, lastUsedAt: now}
                    : p
            ),
            updateBuffer: new Map(state.updateBuffer).set(packageId, {
                lastUsedAt: now,
                usedCount: newUsedCount
            })
        }));

        // Trigger debounced batch update
        debouncedFlushUpdates();
    }

    const flushBufferUpdates = async () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
            debounceTimer = null;
        }

        const {updateBuffer} = get();
        if (updateBuffer.size === 0) return;

        try {
            const updates = Array.from(updateBuffer.entries()).map(([packageId, data]) => ({
                packageId,
                lastUsedAt: data.lastUsedAt,
                usedCount: data.usedCount
            }));

            await updatePluginUsage(updates);
            set({updateBuffer: new Map()});
        } catch (error) {
            Logger.error(`Failed to flush plugin buffer updates: ${error}`);
        }
    }

    const togglePlugin = async (packageId: string, enabled: boolean) => {
        try {
            await togglePluginApi(packageId, enabled)
            // Update local state
            set(state => ({
                plugins: state.plugins.map(p =>
                    p.packageId === packageId ? {...p, enabled} : p
                )
            }))
        } catch (error) {
            Logger.error(`Failed to toggle plugin: ${error}`)
            throw error
        }
    }

    const uninstallPlugin = async (packageId: string) => {
        try {
            await uninstallPluginApi(packageId)
            // Remove from local state
            set(state => ({
                plugins: state.plugins.filter(p => p.packageId !== packageId)
            }))
        } catch (error) {
            Logger.error(`Failed to uninstall plugin: ${error}`)
            throw error
        }
    }

    const store = {
        plugins: [],
        isLoading: false,
        error: null,
        updateBuffer: new Map(),
        fetchPlugins,
        refreshPlugins,
        getPluginById,
        getEnabledPlugins,
        getPluginsByType,
        updatePluginUsage: updatePluginUsageMethod,
        flushBufferUpdates,
        togglePlugin,
        uninstallPlugin
    }

    // Auto-initialize data immediately
    void fetchPlugins()

    return store
})