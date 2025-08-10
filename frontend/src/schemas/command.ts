export const COMMAND_CATEGORY = {
    Application: "Application",
    Operation: "Operation"
} as const

export type CommandCategoryType = typeof COMMAND_CATEGORY[keyof typeof COMMAND_CATEGORY]

export type CommandType = {
    triggerId: string
    name: string,
    description: string
    category: CommandCategoryType,
}

export type ApplicationCommandType = CommandType & {
    path: string
    iconPath: string
    id: string

    // calculated
    nameInitial: string | null
    pathName: string
}

export type OperationCommandType = CommandType


export type CommandGroupType<T extends CommandType> = {
    category: CommandCategoryType,
    commands: T[]
}
