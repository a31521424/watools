export type LogParseStatus = "structured" | "raw"

export type LogFileInfo = {
    name: string
    path: string
    size: number
    updatedAt: string
}

export type LogRecord = {
    id: string
    file: string
    filePath: string
    line: number
    raw: string
    parseStatus: LogParseStatus
    timestamp?: string
    level?: string
    message: string
    source?: string
    pluginId?: string
    error?: string
    fields?: Record<string, string>
}

export type LogQuery = {
    files?: string[]
    keyword?: string
    levels?: string[]
    sources?: string[]
    pluginIds?: string[]
    cursor?: string
    limit?: number
}

export type QueryLogsResult = {
    records: LogRecord[]
    nextCursor?: string
    availableSources: string[]
    availablePluginIds: string[]
    totalMatched: number
}
