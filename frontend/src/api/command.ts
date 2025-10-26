import {ApplicationCommandType, CommandGroupType, OperationCommandType} from "@/schemas/command";
import {isContainNonAscii, toPinyinInitial} from "@/lib/search";
import {GetApplicationCommandsApi, GetOperatorCommandsApi, UpdateApplicationUsageApi} from "../../wailsjs/go/coordinator/WaAppCoordinator";

export const getApplicationCommands = async (): Promise<CommandGroupType<ApplicationCommandType>> => {
    const commands = await GetApplicationCommandsApi()
    console.log('fetch application commands', commands)

    let filterCommands: ApplicationCommandType[] = []
    filterCommands = commands.map(command => ({
        ...command,
        lastUsedAt: command.lastUsedAt ? new Date(command.lastUsedAt) : null,
        category: 'Application',
        nameInitial: isContainNonAscii(command.name) ? toPinyinInitial(command.name) : null,
        pathName: command.path.split('/').pop() || ''
    }))
    return {
        category: 'Application',
        commands: filterCommands
    }
}

export const getOperationCommands = async (): Promise<CommandGroupType<OperationCommandType>> => {
    const commands = await GetOperatorCommandsApi()
    console.log('fetch operation commands', commands)

    let filterCommands: OperationCommandType[] = []
    if (commands) {
        filterCommands = commands.map(command => ({
            ...command
        }))
    }
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