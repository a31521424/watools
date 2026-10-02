import {AppClipboardContent, AppClipboardContentType, AppInputValueType} from "@/schemas/app";
import {PluginContext} from "@/schemas/plugin";

const PLUGIN_LAUNCH_STORAGE_PREFIX = "watools.plugin.launch.";
const PLUGIN_LAUNCH_MAX_AGE_MS = 5 * 60 * 1000;

type StoredPluginLaunchContext = {
    context: PluginContext
    createdAt: number
}

const createEmptyPluginContext = (): PluginContext => ({
    input: {
        value: "",
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
})

const launchContexts = new Map<string, StoredPluginLaunchContext>();

const createLaunchId = (): string => {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
        return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

const sanitizeClipboardContent = (
    inputValue: string,
    clipboardContentType?: AppClipboardContentType,
    clipboardImageBase64?: string | null,
    clipboardFiles?: string[] | null,
): AppClipboardContent | null => {
    if (!clipboardContentType) {
        return null;
    }

    return {
        contentType: clipboardContentType,
        text: clipboardContentType === "text" ? inputValue : null,
        imageBase64: clipboardImageBase64 ?? null,
        files: clipboardFiles ?? null,
    };
}

export const buildPluginContext = (
    inputValue: string,
    valueType: AppInputValueType,
    clipboardContentType?: AppClipboardContentType,
    clipboardImageBase64?: string | null,
    clipboardFiles?: string[] | null,
): PluginContext => ({
    input: {
        value: inputValue,
        valueType,
        clipboardContentType: clipboardContentType ?? undefined,
    },
    clipboard: sanitizeClipboardContent(inputValue, clipboardContentType, clipboardImageBase64, clipboardFiles),
})

export const createSeedPluginContext = (seed: string): PluginContext => ({
    input: {
        value: seed,
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
})

export const hasPluginContextPayload = (context: PluginContext | null | undefined): context is PluginContext => {
    if (!context) {
        return false;
    }

    if (context.input.value.trim()) {
        return true;
    }

    if (!context.clipboard) {
        return false;
    }

    return context.clipboard.contentType === "image"
        ? Boolean(context.clipboard.imageBase64)
        : context.clipboard.contentType === "files"
            ? Array.isArray(context.clipboard.files) && context.clipboard.files.length > 0
            : Boolean(context.clipboard.text?.trim());
}

export const persistPluginLaunchContext = (context: PluginContext): string => {
    const now = Date.now();
    for (const [id, stored] of launchContexts) {
        if (now - stored.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) launchContexts.delete(id);
    }
    while (launchContexts.size >= 20) launchContexts.delete(launchContexts.keys().next().value!);
    // Remove plaintext launch records left by older host versions.
    try {
        if (typeof window !== 'undefined') {
            for (let index = window.sessionStorage.length - 1; index >= 0; index--) {
                const key = window.sessionStorage.key(index);
                if (key?.startsWith(PLUGIN_LAUNCH_STORAGE_PREFIX)) window.sessionStorage.removeItem(key);
            }
        }
    } catch { /* session storage may be unavailable; in-memory delivery still works */ }
    const id = createLaunchId();
    launchContexts.set(id, {context: JSON.parse(JSON.stringify(context)), createdAt: now});
    return id;
}

export const readPluginLaunchContext = (launchId: string | null | undefined): PluginContext | null => {
    if (!launchId) return null;
    const stored = launchContexts.get(launchId);
    if (!stored || Date.now() - stored.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) {
        launchContexts.delete(launchId);
        return null;
    }
    return stored.context;
}

export const releasePluginLaunchContext = (launchId: string | null | undefined) => {
    if (launchId) launchContexts.delete(launchId);
}

export const resolvePluginLaunchContext = ({
    launchId,
    seed,
    liveContext,
}: {
    launchId?: string | null
    seed?: string | null
    liveContext: PluginContext
}): PluginContext => {
    const storedContext = readPluginLaunchContext(launchId);
    if (storedContext) {
        return storedContext;
    }

    if (seed && seed.trim()) {
        return createSeedPluginContext(seed);
    }

    if (hasPluginContextPayload(liveContext)) {
        return liveContext;
    }

    return createEmptyPluginContext();
}

export const getLegacySeedValue = (context: PluginContext): string => context.input.value.trim();
