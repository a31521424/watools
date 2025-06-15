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
    icon?: ReactNode | string | null
}


export type CommandGroupType = {
    category: CommandCategoryType,
    commands: CommandType[]
}

export const mockCommandGroups: CommandGroupType[] = [
    {
        category: COMMAND_CATEGORY.Application, // 應用程式群組的標題
        commands: [
            {
                name: "Launch Visual Studio Code",
                category: COMMAND_CATEGORY.Application, // 指令的分類
                description: "Open the VS Code editor to start coding",
                icon: "💻",
            },
            {
                name: "Open Figma",
                category: COMMAND_CATEGORY.Application,
                description: "Start the Figma design tool for UI/UX work",
                icon: "🎨",
            },
            {
                name: "Run Terminal",
                category: COMMAND_CATEGORY.Application,
                description: "Open a new terminal or command prompt window",
                icon: "셸", // This is a shell emoji, if not rendered properly, you can use ">_"
            },
        ],
    },
    {
        category: COMMAND_CATEGORY.SystemOperation, // 系統操作群組的標題
        commands: [
            {
                name: "Sleep",
                category: COMMAND_CATEGORY.SystemOperation,
                description: "Put the computer into sleep mode to save power",
                icon: "💤",
            },
            {
                name: "Lock Screen",
                category: COMMAND_CATEGORY.SystemOperation,
                description: "Secure your computer by locking the screen",
                icon: "🔒",
            },
            {
                name: "Empty Trash",
                category: COMMAND_CATEGORY.SystemOperation,
                description: "Permanently delete all items in the Trash",
                icon: "🗑️",

            },
            {
                name: "Toggle Dark Mode",
                category: COMMAND_CATEGORY.SystemOperation,
                description: "Switch between light and dark system themes",
                icon: "🌓",
            },
        ],
    },
];