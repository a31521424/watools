import {createUsageFlusher} from "@/lib/usage-buffer";
import {create} from "zustand";
import {ApplicationCommandType, CommandGroupType} from "@/schemas/command";
import {getApplicationCommands, updateApplicationUsage} from "@/api/command";
import {EventsOn} from "@/lib/legacy-runtime";
import Fuse, {IFuseOptions} from "fuse.js";
import {Logger} from "@/lib/logger";
import {getSearchQuery} from '@/lib/input-text';

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
    loadError: string | null
    updateBuffer: Map<string, { lastUsedAt: Date, usedCount: number }>
}

const initialState: ApplicationCommandState = {
    commandGroup: null,
    fuse: null,
    isLoading: false,
    loadError: null,
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
    let eventRefreshPromise: Promise<void> | null = null
    let eventRefreshPending = false
    let removeApplicationChangedListener: (() => void) | null = null
    let usageVersion = 0
    const usageVersions = new Map<string, number>()

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

    const loadCommands = (): Promise<void> => {
        if (loadPromise) return loadPromise

        const run = async () => {
            set({isLoading: true})
            try {
                const queryUsageVersion = usageVersion
                const fetched = await getApplicationCommands()
                const current = get()
                const local = new Map(current.commandGroup?.commands
                    .filter(command => (usageVersions.get(command.id) ?? 0) > queryUsageVersion)
                    .map(command => [command.id, command]) ?? [])
                // Usage can change and be acknowledged while the query is in
                // flight. Merge from both live commands and unacknowledged data.
                const commandGroup = {...fetched, commands: fetched.commands.map(command => {
                    const previous = local.get(command.id)
                    const pending = current.updateBuffer.get(command.id)
                    const lastUsedAt = [command.lastUsedAt, previous?.lastUsedAt, pending?.lastUsedAt]
                        .reduce<Date | null>((latest, value) => value && (!latest || value > latest) ? value : latest, null)
                    return {...command, lastUsedAt, usedCount: Math.max(command.usedCount, previous?.usedCount ?? 0, pending?.usedCount ?? 0)}
                })}
                const fuse = createFuseInstance(commandGroup.commands)
                set({commandGroup, fuse, loadError: null})
            } catch (error) {
                set({loadError: get().commandGroup?.commands.length
                    ? "应用列表加载失败，请重试；已有应用仍可搜索"
                    : "应用列表加载失败，请重试"})
                throw error
            } finally {
                set({isLoading: false})
            }
        }
        loadPromise = run().finally(() => { loadPromise = null })

        return loadPromise
    }

    const refreshCommands = async () => {
        await flushBufferUpdates()
        if (get().updateBuffer.size) {
            const message = "使用记录尚未保存，请稍后重试刷新"
            set({loadError: message})
            throw new Error(message)
        }
        await loadCommands()
    }

    const searchCommands = (searchKey: string, limit: number = 5): ApplicationCommandType[] => {
        const query = getSearchQuery(searchKey)
        const {fuse} = get()
        if (!fuse || !query) {
            return []
        }

        return fuse.search(query, {limit}).map(result => result.item)
    }

    const updateCommandUsage = async (commandId: string) => {
        const state = get()
        const {commandGroup, updateBuffer} = state

        if (!commandGroup) return

        const command = commandGroup.commands.find(cmd => cmd.id === commandId)
        if (!command) return

        const now = new Date()
        const newUsedCount = command.usedCount + 1
        usageVersions.set(commandId, ++usageVersion)

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
        catch {
            debouncedFlushUpdates()
            void Promise.resolve().then(() => Logger.error("应用使用记录保存失败，将稍后重试")).catch(() => {})
        }
    }

    const refreshFromEvents = () => {
        eventRefreshPending = true
        if (eventRefreshPromise) return
        const run = async () => {
            while (eventRefreshPending) {
                // A query that started before this event may contain an older
                // snapshot. Let it finish, then fetch once for this event burst.
                if (loadPromise) await loadPromise.catch(() => {})
                eventRefreshPending = false
                await refreshCommands().catch(() => {}) // loadError owns user feedback.
            }
        }
        eventRefreshPromise = run().finally(() => {
            eventRefreshPromise = null
            // An event can arrive between the loop's exit and this finalizer.
            if (eventRefreshPending) refreshFromEvents()
        })
        void eventRefreshPromise.catch(() => {})
    }

    const startListening = () => {
        if (removeApplicationChangedListener) return

        removeApplicationChangedListener = EventsOn('watools.applicationChanged', refreshFromEvents)
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

    // Zustand must finish publishing the initial state before an async task
    // updates it. Listen even when the first query fails, so events can recover.
    queueMicrotask(() => {
        startListening()
        void loadCommands().catch(() => {})
    })

    return store
})
