import {create} from "zustand";
import {ApplicationCommandType, CommandGroupType} from "@/schemas/command";
import {getApplicationCommands, updateApplicationUsage} from "@/api/command";
import {EventsOff, EventsOn} from "../../wailsjs/runtime";
import Fuse, {IFuseOptions} from "fuse.js";

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
    let isListening = false
    let debounceTimer: ReturnType<typeof setTimeout> | null = null
    let isLoading = false

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

    const debouncedFlushUpdates = () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer)
        }

        debounceTimer = setTimeout(async () => {
            const {updateBuffer} = get()

            if (updateBuffer.size === 0) return

            try {
                const updates = Array.from(updateBuffer.entries()).map(([id, data]) => ({
                    id,
                    lastUsedAt: data.lastUsedAt,
                    usedCount: data.usedCount
                }))

                await updateApplicationUsage(updates)
                set({updateBuffer: new Map()})
            } catch (error) {
                console.error('Failed to flush buffer updates:', error)
            }
        }, DEBOUNCE_DELAY)
    }

    const loadCommands = async () => {
        if (isLoading) return

        set({isLoading: true})
        try {
            const commandGroup = await getApplicationCommands()
            const fuse = createFuseInstance(commandGroup.commands)
            set({commandGroup, fuse, isLoading: false})
            startListening()
        } catch (error) {
            console.error('Failed to load application commands:', error)
        } finally {
            set({isLoading: false})
        }
    }

    const refreshCommands = async () => {
        console.log('Application commands changed, refreshing...')
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

        debouncedFlushUpdates()
    }

    const flushBufferUpdates = async () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer)
            debounceTimer = null
        }

        const {updateBuffer} = get()

        if (updateBuffer.size === 0) return

        try {
            const updates = Array.from(updateBuffer.entries()).map(([id, data]) => ({
                id,
                lastUsedAt: data.lastUsedAt,
                usedCount: data.usedCount
            }))

            await updateApplicationUsage(updates)
            set({updateBuffer: new Map()})
        } catch (error) {
            console.error('Failed to flush buffer updates:', error)
        }
    }

    const startListening = () => {
        if (isListening) return

        isListening = true
        EventsOn('watools.applicationChanged', refreshCommands)
    }

    const stopListening = () => {
        if (!isListening) return

        isListening = false
        EventsOff('watools.applicationChanged')
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
