import {ApplicationCommandType, CommandGroupType, OperationCommandType} from "@/schemas/command";
import {isContainNonAscii, toPinyin, toPinyinInitial} from "@/lib/search";
import {GetApplicationCommandsApi, GetOperatorCommandsApi, UpdateApplicationUsageApi} from "../../wailsjs/go/coordinator/WaAppCoordinator";

const dedupeBy = <T>(items: T[], getKey: (item: T) => string): T[] => {
    const uniqueItems = new Map<string, T>()
    for (const item of items) {
        const key = getKey(item)
        if (!uniqueItems.has(key)) {
            uniqueItems.set(key, item)
        }
    }
    return Array.from(uniqueItems.values())
}

export const getApplicationCommands = async (): Promise<CommandGroupType<ApplicationCommandType>> => {
    const commands = await GetApplicationCommandsApi()

    let filterCommands: ApplicationCommandType[] = []
    filterCommands = dedupeBy(commands.map(command => ({
        ...command,
        lastUsedAt: command.lastUsedAt ? new Date(command.lastUsedAt) : null,
        category: 'Application',
        namePinyin: isContainNonAscii(command.name) ? toPinyin(command.name) : null,
        nameInitial: isContainNonAscii(command.name) ? toPinyinInitial(command.name) : null,
        pathName: command.path.split('/').pop() || ''
    })), command => command.id || command.triggerId)
    return {
        category: 'Application',
        commands: filterCommands
    }
}

export const getOperationCommands = async (): Promise<CommandGroupType<OperationCommandType>> => {
    const commands = await GetOperatorCommandsApi()

    let filterCommands: OperationCommandType[] = []
    filterCommands = dedupeBy(commands.map(command => ({
        ...command
    })), command => command.triggerId)
    return {
        category: 'Operation',
        commands: filterCommands
    }
}

export const updateApplicationUsage = async (usageUpdates: Array<{
    id: string
    lastUsedAt: Date
    usedCount: number
}>): Promise<void> => {
    const updates = usageUpdates.map(update => ({
        id: update.id,
        lastUsedAt: update.lastUsedAt.toISOString(),
        usedCount: update.usedCount
    }))

    await UpdateApplicationUsageApi(updates)
}
