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


export type CommandGroupType = {
    category: CommandCategoryType,
    commands: CommandType[]
}
