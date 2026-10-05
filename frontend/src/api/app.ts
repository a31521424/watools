import {host} from "@/api/host"
import {AppClipboardContent, AppClipboardContentType} from "@/schemas/app";
import type {AutomaticClipboard} from '@/api/host/contracts';
import type {PluginContext} from '@/schemas/plugin';
import type {LauncherInput} from '@/schemas/launcher';
import {snapshotPluginContext} from '@/lib/plugin-context';
import {getClipboardSnapshotId} from '@/lib/launcher-input';

let automaticRead: Promise<AutomaticClipboard | null> | null = null;
export const getAutomaticClipboard = (): Promise<AutomaticClipboard | null> => {
    if (!automaticRead) {
        automaticRead = host.GetAutomaticClipboardApi().finally(() => { automaticRead = null });
    }
    return automaticRead;
};

export const resolvePluginContext = async (input: LauncherInput): Promise<PluginContext> => {
    if (input.kind === 'inline') return snapshotPluginContext(input.context);
    const text = await host.GetClipboardSnapshotTextApi(input.snapshot.id);
    return {
        input: {value: text, valueType: 'clipboard', clipboardContentType: 'text'},
        clipboard: {contentType: 'text', text, imageBase64: null, files: null},
    };
};

// The lease belongs to launcher state, not to a route component. In particular,
// cancelling an iframe before it loads must leave its source usable on return.
export const watchClipboardSnapshot = (source: {getState: () => {input: LauncherInput}; subscribe: (listener: () => void) => () => void}, onError: () => void): (() => void) => {
    let currentId: string | undefined;
    let lease: ReturnType<typeof setInterval> | undefined;
    const refresh = () => {
        const id = getClipboardSnapshotId(source.getState().input);
        if (id === currentId) return;
        if (lease !== undefined) clearInterval(lease);
        if (currentId) void host.ReleaseClipboardSnapshotApi(currentId).catch(onError);
        currentId = id;
        if (id) {
            void host.RetainClipboardSnapshotApi(id).catch(onError);
            lease = setInterval(() => {void host.RetainClipboardSnapshotApi(id).catch(onError)}, 60_000);
        }
    };
    refresh();
    const unsubscribe = source.subscribe(refresh);
    return () => {
        unsubscribe();
        if (lease !== undefined) clearInterval(lease);
    };
};

export const getClipboardContent = async (): Promise<AppClipboardContent | null> => {
    const content = await host.GetClipboardContentApi()
    if (!content || content.contentType === "empty") {
        return null
    }
    const data: AppClipboardContent = {
        contentType: content.contentType as AppClipboardContentType,
        text: null,
        imageBase64: null,
        files: null,
    }
    if (content.hasFiles) {
        data.files = content.files as string[]
    }
    if (content.hasImage) {
        data.imageBase64 = content.imageBase64 as string
    }
    if (content.hasText) {
        data.text = content.text as string
    }
    return data
}
