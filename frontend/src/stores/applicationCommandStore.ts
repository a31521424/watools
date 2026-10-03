import {createUsageFlusher} from "@/lib/usage-buffer";
import {create} from "zustand";
import {ApplicationCommandType, CommandGroupType} from "@/schemas/command";
import {getApplicationCommands, updateApplicationUsage} from "@/api/command";
import {EventsOn} from "@/lib/legacy-runtime";
import Fuse, {IFuseOptions} from "fuse.js";
import {Logger} from "@/lib/logger";

export const WaApplicationCommandFuseConfig: IFuseOptions<ApplicationCommandType> = {
    threshold: 0.15,
    minMatchCharLength: 2,
    useExtendedSearch: true,
    ignoreLocation: true,
    shouldSort: false,
    keys: [{
        name: 'name',
        weight: 1.0
    }, {
        name: 'namePinyin',
        weight: 0.95
    }, {
        name: 'nameInitial',
        weight: 0.9
    }, {
        name: 'pathName',
        weight: 0.6
    }]
}

type ApplicationCommandState = {
    commandGroup: CommandGroupType<ApplicationCommandType> | null
    fuse: Fuse<ApplicationCommandType> | null
    isLoading: boolean
    updateBuffer: Map<string, { lastUsedAt: Date, usedCount: number }>
}

const initialState: ApplicationCommandState = {
    commandGroup: null,
    fuse: null,
    isLoading: false,
    updateBuffer: new Map()
}

type ApplicationCommandStore = ApplicationCommandState & {
    loadCommands: () => Promise<void>
    refreshCommands: () => Promise<void>
    searchCommands: (searchKey: string, limit?: number) => ApplicationCommandType[]
    updateCommandUsage: (commandId: string) => Promise<void>
    flushBufferUpdates: () => Promise<void>
    startListening: () => void
    stopListening: () => void
}

const DEBOUNCE_DELAY = 60000

export const useApplicationCommandStore = create<ApplicationCommandStore>((set, get) => {
    let debounceTimer: ReturnType<typeof setTimeout> | null = null
    let loadPromise: Promise<void> | null = null
    let removeApplicationChangedListener: (() => void) | null = null

    const createFuseInstance = (commands: ApplicationCommandType[]) => {
        const sortedCommands = [...commands].sort((a, b) => {
            // Prioritize user applications over system applications
            if (a.isUserApp !== b.isUserApp) {
                return a.isUserApp ? -1 : 1
            }

            // Then sort by usage count
            if (a.usedCount !== b.usedCount) {
                return b.usedCount - a.usedCount
            }

            // Then by last used time
            if (a.lastUsedAt && b.lastUsedAt) {
                return b.lastUsedAt.getTime() - a.lastUsedAt.getTime()
            }
            if (a.lastUsedAt && !b.lastUsedAt) return -1
            if (!a.lastUsedAt && b.lastUsedAt) return 1

            // Finally by name alphabetically
            return a.name.localeCompare(b.name)
        })

        return new Fuse(sortedCommands, WaApplicationCommandFuseConfig)
    }

    const flushPending = createUsageFlusher(
        () => get().updateBuffer,
        snapshot => set(state => {
            const remaining = new Map(state.updateBuffer)
            for (const [id, value] of snapshot) {
                if (remaining.get(id) === value) remaining.delete(id)
            }
            return {updateBuffer: remaining}
        }),
        snapshot => updateApplicationUsage(Array.from(snapshot, ([id, data]) => ({id, ...data}))),
    )

    const debouncedFlushUpdates = () => {
        if (debounceTimer) clearTimeout(debounceTimer)
        debounceTimer = setTimeout(() => { void flushBufferUpdates() }, DEBOUNCE_DELAY)
    }

    const loadCommands = async () => {
        if (loadPromise) return loadPromise

        loadPromise = (async () => {
            set({isLoading: true})
            try {
                const commandGroup = await getApplicationCommands()
                const fuse = createFuseInstance(commandGroup.commands)
                set({commandGroup, fuse, isLoading: false})
                startListening()
            } catch (error) {
                Logger.error(`Failed to load application commands: ${error}`)
                set({isLoading: false})
            } finally {
                loadPromise = null
            }
        })()

        return loadPromise
    }

    const refreshCommands = async () => {
        await flushBufferUpdates()
        if (get().updateBuffer.size) throw new Error("使用记录未保存，请重试刷新")
        await loadCommands()
    }

    const searchCommands = (searchKey: string, limit: number = 5): ApplicationCommandType[] => {
        const {fuse} = get()
        if (!fuse || !searchKey.trim()) {
            return []
        }

        return fuse.search(searchKey, {limit}).map(result => result.item)
    }

    const updateCommandUsage = async (commandId: string) => {
        const state = get()
        const {commandGroup, updateBuffer} = state

        if (!commandGroup) return

        const command = commandGroup.commands.find(cmd => cmd.id === commandId)
        if (!command) return

        const now = new Date()
        const newUsedCount = command.usedCount + 1

        command.lastUsedAt = now
        command.usedCount = newUsedCount

        const newBuffer = new Map(updateBuffer)
        newBuffer.set(commandId, {lastUsedAt: now, usedCount: newUsedCount})

        const fuse = createFuseInstance(commandGroup.commands)

        set({
            commandGroup: {...commandGroup},
            fuse,
            updateBuffer: newBuffer
        })

        await flushBufferUpdates()
    }

    const flushBufferUpdates = async () => {
        if (debounceTimer) { clearTimeout(debounceTimer); debounceTimer = null }
        try { await flushPending() }
        catch (error) { Logger.error(`Failed to flush usage: ${error}`); debouncedFlushUpdates() }
    }

    const startListening = () => {
        if (removeApplicationChangedListener) return

        removeApplicationChangedListener = EventsOn('watools.applicationChanged', refreshCommands)
    }

    const stopListening = () => {
        if (!removeApplicationChangedListener) return
        removeApplicationChangedListener()
        removeApplicationChangedListener = null
    }

    const store = {
        ...initialState,
        loadCommands,
        refreshCommands,
        searchCommands,
        updateCommandUsage,
        flushBufferUpdates,
        startListening,
        stopListening
    }

    // Auto-initialize data immediately
    void loadCommands()

    return store
})
