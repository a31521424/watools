import type {PluginContext} from '@/schemas/plugin';

/** A deferred origin has no text value until the user actually selects an action. */
export type ClipboardSnapshot = {
    readonly id: string
    readonly length: number
    readonly head: string
    readonly tail: string
    readonly preview: string
};

/** The launcher's single source of truth. Inline contexts always contain real data. */
export type LauncherInput =
    | {readonly kind: 'inline'; readonly context: PluginContext}
    | {readonly kind: 'clipboard-snapshot'; readonly snapshot: ClipboardSnapshot};
