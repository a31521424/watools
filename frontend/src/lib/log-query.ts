import type {LogQuery, LogRecord, QueryLogsResult} from "@/schemas/log"

export type LogQueryState = {
    records: LogRecord[]
    selectedRecordId: string | null
    availableSources: string[]
    availablePluginIds: string[]
    nextCursor: string | null
    totalMatched: number
    error: string | null
    isLoadingLogs: boolean
    isLoadingMore: boolean
}

export const initialLogQueryState: LogQueryState = {
    records: [], selectedRecordId: null, availableSources: [], availablePluginIds: [],
    nextCursor: null, totalMatched: 0, error: null, isLoadingLogs: false, isLoadingMore: false,
}

type LogQueryAction =
    | {type: "reset"; preserveError?: boolean}
    | {type: "invalidated"}
    | {type: "error"; message: string}
    | {type: "start"; append: boolean}
    | {type: "result"; result: QueryLogsResult; append: boolean}
    | {type: "failed"; message: string; append: boolean}
    | {type: "settled"; append: boolean}
    | {type: "select"; id: string}

// Related fields change together; React can replay this reducer without side effects.
export function logQueryReducer(state: LogQueryState, action: LogQueryAction): LogQueryState {
    switch (action.type) {
        case "reset":
            return action.preserveError ? {...initialLogQueryState, error: state.error} : initialLogQueryState
        case "invalidated":
            return {...state, nextCursor: null, error: null, isLoadingLogs: false, isLoadingMore: false}
        case "error":
            return {...state, error: action.message}
        case "select":
            return {...state, selectedRecordId: action.id}
        case "start":
            return action.append
                ? {...state, error: null, isLoadingMore: true}
                : {...state, records: [], nextCursor: null, totalMatched: 0, error: null,
                    isLoadingLogs: true, isLoadingMore: false}
        case "result": {
            const records = action.append ? [...state.records, ...action.result.records] : action.result.records
            const selectedRecordId = records.some(record => record.id === state.selectedRecordId)
                ? state.selectedRecordId : records[0]?.id ?? null
            return {...state, records, selectedRecordId, availableSources: action.result.availableSources,
                availablePluginIds: action.result.availablePluginIds,
                nextCursor: action.result.nextCursor || null, totalMatched: action.result.totalMatched}
        }
        case "failed":
            return action.append ? {...state, error: action.message}
                : {...state, records: [], selectedRecordId: null, nextCursor: null, totalMatched: 0, error: action.message}
        case "settled":
            return action.append ? {...state, isLoadingMore: false} : {...state, isLoadingLogs: false}
    }
}

// A page cursor belongs to exactly one completed query. Invalidation also makes
// errors/finalizers harmless after a filter change, refresh, or unmount.
export function createLogQueryController(
    queryLogs: (query: LogQuery) => Promise<QueryLogsResult>,
    dispatch: (action: LogQueryAction) => void,
) {
    let generation = 0
    let query: LogQuery | null = null
    let cursor: string | null = null
    let pending = false

    const invalidate = () => {
        generation += 1
        query = null
        cursor = null
        pending = false
    }

    const run = async (request: LogQuery, append: boolean) => {
        const ownGeneration = generation
        pending = true
        dispatch({type: "start", append})
        try {
            const result = await queryLogs(request)
            if (ownGeneration !== generation) return
            cursor = result.nextCursor || null
            dispatch({type: "result", result, append})
        } catch (error) {
            if (ownGeneration !== generation) return
            dispatch({type: "failed", append,
                message: error instanceof Error ? error.message : "Failed to query logs"})
        } finally {
            if (ownGeneration === generation) {
                pending = false
                dispatch({type: "settled", append})
            }
        }
    }

    return {
        invalidate,
        reset(preserveError = false) {
            invalidate()
            dispatch({type: "reset", preserveError})
        },
        replace(nextQuery: LogQuery) {
            invalidate()
            // Retain a snapshot for pagination, rather than current UI filters.
            query = {...nextQuery, files: nextQuery.files?.slice(), levels: nextQuery.levels?.slice(),
                sources: nextQuery.sources?.slice(), pluginIds: nextQuery.pluginIds?.slice(), cursor: undefined}
            return run(query, false)
        },
        async append() {
            if (!query || !cursor || pending) return
            await run({...query, cursor}, true)
        },
    }
}
