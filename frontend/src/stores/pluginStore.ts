import {createUsageFlusher} from "@/lib/usage-buffer";
import {create} from 'zustand'
import {Plugin} from '@/schemas/plugin'
import {getPlugins, updatePluginUsage, togglePlugin as togglePluginApi, uninstallPlugin as uninstallPluginApi, loadPluginEntries, disposePlugin} from "@/api/plugin";
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
    let loadPromise: Promise<void> | null = null

    const flushPending = createUsageFlusher(
        () => get().updateBuffer,
        snapshot => set(state => {
            const remaining = new Map(state.updateBuffer)
            for (const [id, value] of snapshot) {
                if (remaining.get(id) === value) remaining.delete(id)
            }
            return {updateBuffer: remaining}
        }),
        snapshot => updatePluginUsage(Array.from(snapshot, ([packageId, data]) => ({packageId, ...data}))),
    )

    const debouncedFlushUpdates = () => {
        if (debounceTimer) clearTimeout(debounceTimer)
        debounceTimer = setTimeout(() => { void flushBufferUpdates() }, DEBOUNCE_DELAY)
    }

    const fetchPlugins = async () => {
        if (isInitialized) return
        if (loadPromise) return loadPromise

        loadPromise = (async () => {
            set({isLoading: true, error: null})
            try {
                const plugins = await getPlugins()
                set({plugins, isLoading: false})
                isInitialized = true
            } catch (error) {
                Logger.error(`Failed to fetch plugins: ${error}`)
                set({
                    error: error instanceof Error ? error.message : 'Failed to fetch plugins',
                    isLoading: false
                })
            } finally {
                loadPromise = null
            }
        })()

        return loadPromise
    }

    const refreshPlugins = async () => {
        await flushBufferUpdates()
        if (get().updateBuffer.size) throw new Error("使用记录未保存，请重试刷新")
        for (const plugin of get().plugins) await disposePlugin(plugin.packageId)
        isInitialized = false
        await fetchPlugins()
        const error = get().error
        if (error) throw new Error(error)
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
        await flushBufferUpdates();
    }

    const flushBufferUpdates = async () => {
        if (debounceTimer) { clearTimeout(debounceTimer); debounceTimer = null }
        try { await flushPending() }
        catch (error) { Logger.error(`Failed to flush usage: ${error}`); debouncedFlushUpdates() }
    }

    const togglePlugin = async (packageId: string, enabled: boolean) => {
        try {
            await togglePluginApi(packageId, enabled)
            const current = getPluginById(packageId)
            if (!current) return
            const next = {...current, enabled, entry: enabled ? current.entry : []}
            if (enabled) await loadPluginEntries(next)
            else { await disposePlugin(packageId); next.loadStatus = 'disabled'; next.loadError = undefined }
            set(state => ({
                plugins: state.plugins.map(p =>
                    p.packageId === packageId ? next : p
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
            await disposePlugin(packageId)
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
