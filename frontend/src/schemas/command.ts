export const COMMAND_CATEGORY = {
    Application: "Application",
    SystemOperation: "SystemOperation"
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
    id: number
    
    // calculated
    nameInitial: string | null
    pathName: string
}


export type CommandGroupType<T extends CommandType> = {
    category: CommandCategoryType,
    commands: T[]
}
