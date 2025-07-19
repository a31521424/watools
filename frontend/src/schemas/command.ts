import {ReactNode} from "react";


export const COMMAND_CATEGORY = {
    Application: "Application",
    SystemOperation: "SystemOperation"
} as const

export type CommandCategoryType = typeof COMMAND_CATEGORY[keyof typeof COMMAND_CATEGORY]

export type CommandType = {
    name: string,
    category: CommandCategoryType,
    description: string
    path: string
    icon?: ReactNode | string | null
    iconPath?: string
}

export type ApplicationCommandType = CommandType & {
    nameInitial: string | null
    pathName: string
}


export type CommandGroupType<T extends CommandType> = {
    category: CommandCategoryType,
    commands: T[]
}
