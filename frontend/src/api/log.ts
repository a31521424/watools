import {GetLogDirectoryApi, ListLogFilesApi, QueryLogsApi} from "../../wailsjs/go/coordinator/WaAppCoordinator"
import {LogFileInfo, LogQuery, LogRecord, QueryLogsResult} from "@/schemas/log"

const normalizeLogRecord = (record: Partial<LogRecord>): LogRecord => ({
    id: record.id || "",
    file: record.file || "",
    filePath: record.filePath || "",
    line: typeof record.line === "number" ? record.line : 0,
    raw: record.raw || "",
    parseStatus: record.parseStatus === "structured" ? "structured" : "raw",
    timestamp: record.timestamp,
    level: record.level,
    message: record.message || record.raw || "",
    source: record.source,
    pluginId: record.pluginId,
    error: record.error,
    fields: record.fields || undefined,
})

export const getLogDirectory = async (): Promise<string> => {
    return GetLogDirectoryApi()
}

export const listLogFiles = async (): Promise<LogFileInfo[]> => {
    const files = await ListLogFilesApi()
    return (files || []).map((file: Partial<LogFileInfo>) => ({
        name: file.name || "",
        path: file.path || "",
        size: typeof file.size === "number" ? file.size : 0,
        updatedAt: file.updatedAt || "",
    }))
}

export const queryLogs = async (query: LogQuery): Promise<QueryLogsResult> => {
    const result = await QueryLogsApi(query)

    return {
        records: Array.isArray(result?.records)
            ? result.records.map(record => normalizeLogRecord(record as Partial<LogRecord>))
            : [],
        nextCursor: typeof result?.nextCursor === "string" ? result.nextCursor : undefined,
        availableSources: Array.isArray(result?.availableSources)
            ? result.availableSources.filter((item: unknown): item is string => typeof item === "string")
            : [],
        availablePluginIds: Array.isArray(result?.availablePluginIds)
            ? result.availablePluginIds.filter((item: unknown): item is string => typeof item === "string")
            : [],
        totalMatched: typeof result?.totalMatched === "number" ? result.totalMatched : 0,
    }
}
