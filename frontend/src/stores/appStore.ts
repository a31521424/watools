import {create} from 'zustand';
import type {AppClipboardContent} from '@/schemas/app';
import type {AutomaticClipboard} from '@/api/host/contracts';
import type {LauncherInput} from '@/schemas/launcher';
import {createAutomaticClipboardInput, createClipboardInput, createTextInput, EMPTY_LAUNCHER_INPUT, hasLauncherInput} from '@/lib/launcher-input';

type AppStore = {
    input: LauncherInput
    inputRevision: number
    lastClipboardVersion: string | null
    setTextValue: (value: string) => void
    setAutomaticClipboard: (content: AutomaticClipboard, expectedRevision: number) => boolean
    setClipboardContent: (content: AppClipboardContent | null, expectedRevision?: number) => void
    clearValue: (expectedRevision?: number) => boolean
    isPanelOpen: () => boolean
    canClearAssets: () => boolean
};

export const useAppStore = create<AppStore>((set, get) => ({
    input: EMPTY_LAUNCHER_INPUT,
    inputRevision: 0,
    lastClipboardVersion: null,
    setTextValue: value => set(state => ({input: createTextInput(value), inputRevision: state.inputRevision + 1})),
    setAutomaticClipboard: (content, expectedRevision) => {
        const state = get();
        if (expectedRevision !== state.inputRevision || state.isPanelOpen() || !content.length || state.lastClipboardVersion === content.version) return false;
        set({input: createAutomaticClipboardInput(content), inputRevision: state.inputRevision + 1, lastClipboardVersion: content.version});
        return true;
    },
    setClipboardContent: (content, expectedRevision) => {
        const state = get();
        if (!content || (expectedRevision !== undefined && expectedRevision !== state.inputRevision)) return;
        if (content.contentType === 'image' && !content.imageBase64) return;
        if (content.contentType === 'files' && !content.files?.length) return;
        set({input: createClipboardInput(content), inputRevision: state.inputRevision + 1});
    },
    clearValue: expectedRevision => {
        const state = get();
        if (expectedRevision !== undefined && state.inputRevision !== expectedRevision) return false;
        set({input: EMPTY_LAUNCHER_INPUT, inputRevision: state.inputRevision + 1});
        return true;
    },
    isPanelOpen: () => hasLauncherInput(get().input),
    canClearAssets: () => {
        const {input} = get();
        return input.kind === 'inline' && !input.context.input.value.length && hasLauncherInput(input);
    },
}));
