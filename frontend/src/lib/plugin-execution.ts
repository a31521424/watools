import {createWaToolsApi, WaToolsApi} from '@/api/api';

let executionQueue: Promise<unknown> = Promise.resolve();

// Legacy plugins read window.watools. Serialize their executions so each async
// command keeps its own API until it finishes. New commands can use the argument.
export const withPluginApi = <T>(packageId: string, execute: (api: WaToolsApi) => Promise<T> | T): Promise<T> => {
    const pending = executionQueue.then(async () => {
        const host = window as Window & {watools?: WaToolsApi};
        const previous = host.watools;
        const api = createWaToolsApi(packageId);
        host.watools = api;
        try { return await execute(api); }
        finally {
            if (previous === undefined) delete host.watools;
            else host.watools = previous;
        }
    });
    executionQueue = pending.catch(() => {});
    return pending;
};
