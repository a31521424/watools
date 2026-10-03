export type HostErrorCode = 'INVALID_ARGUMENT' | 'OPERATION_FAILED' | 'CANCELLED' | 'TIMEOUT' | 'UNAVAILABLE' | 'PLUGIN_DISABLED' | 'TRANSPORT_ERROR'
export class HostError extends Error {
    constructor(public readonly code: HostErrorCode, message: string, public readonly operation?: string, options?: ErrorOptions) {
        super(message, options)
        this.name = 'HostError'
    }
}
const codes = new Set<HostErrorCode>(['INVALID_ARGUMENT', 'OPERATION_FAILED', 'CANCELLED', 'TIMEOUT', 'UNAVAILABLE', 'PLUGIN_DISABLED', 'TRANSPORT_ERROR'])
export function normalizeHostError(error: unknown): HostError {
    if (error instanceof HostError) return error
    const cause = error && typeof error === 'object' && 'cause' in error ? error.cause : undefined
    if (cause && typeof cause === 'object' && 'code' in cause && 'message' in cause &&
        typeof cause.code === 'string' && codes.has(cause.code as HostErrorCode) && typeof cause.message === 'string') {
        return new HostError(cause.code as HostErrorCode, cause.message,
            'operation' in cause && typeof cause.operation === 'string' ? cause.operation : undefined, {cause: error})
    }
    return new HostError('TRANSPORT_ERROR', '宿主连接失败，请重试', undefined, {cause: error})
}
// Bindings return cancellable promises; the public SDK historically exposes
// ordinary promises and provides HTTP cancellation through requestId.
export function withHostErrors<T extends object>(api: T): T {
    return new Proxy(api, {get(target, key, receiver) {
        const method = Reflect.get(target, key, receiver)
        if (typeof method !== 'function') return method
        return async (...args: unknown[]) => {
            try { return await method.apply(target, args) } catch (error) { throw normalizeHostError(error) }
        }
    }})
}
