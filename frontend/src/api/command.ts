import {GetAllCommands} from "../../wailsjs/go/command/WaLaunchApp";
import {ApplicationCommandType, CommandGroupType} from "@/schemas/command";
import {isContainNonAscii, toPinyinInitial} from "@/lib/search";

export const getApplicationCommands = async (): Promise<CommandGroupType<ApplicationCommandType>> => {
    const commands = await GetAllCommands()
    console.log('fetch commands', commands)

    let filterCommands: ApplicationCommandType[] = []
    if (commands) {
        filterCommands = commands.map(command => ({
            ...command,
            category: 'Application',
            nameInitial: isContainNonAscii(command.name) ? toPinyinInitial(command.name) : null,
            pathName: command.path.split('/').pop() || ''
        }))
    }
    return {
        category: 'Application',
        commands: filterCommands
    }
}