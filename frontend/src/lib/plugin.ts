import type {Plugin, PluginEntry, PluginEntryDefinition, PluginMatcher} from "@/schemas/plugin";
import {MAX_MATCH_TEXT_EDGE_LENGTH} from '@/lib/input-text';

const isNonEmptyString = (value: unknown): value is string => {
    return typeof value === "string" && value.trim().length > 0;
}

export const normalizePluginAssetPath = (value: unknown): string | null => {
    if (!isNonEmptyString(value)) {
        return null;
    }

    const normalized = value.trim().replaceAll("\\", "/");
    if (normalized.startsWith("/")) {
        return null;
    }

    const segments = normalized.split("/");
    if (segments.some(segment => segment.length === 0 || segment === "." || segment === "..")) {
        return null;
    }

    return segments.join("/");
}

const doesPluginAssetExist = async (assetUrl: string): Promise<boolean> => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);
    try {
        const response = await fetch(assetUrl, {method: "HEAD", signal: controller.signal});
        return response.ok;
    } catch (error) {
        console.error(`Failed to validate plugin asset ${assetUrl}:`, error);
        return false;
    } finally { clearTimeout(timer); }
}

export const sanitizePluginEntries = async (plugin: Plugin, entries: unknown[]): Promise<PluginEntry[]> => {
    const sanitizedEntries = await Promise.all(entries.map(async (entry): Promise<PluginEntry | null> => {
        if (!entry || typeof entry !== "object") {
            return null;
        }

        const candidate = entry as Partial<PluginEntryDefinition>;
        let matcher: PluginMatcher;
        if (typeof candidate.matchPreview === 'function') {
            matcher = {kind: 'preview', match: candidate.matchPreview};
        } else if (typeof candidate.match === 'function') {
            const largeTextPrefixes = Array.isArray(candidate.largeTextPrefixes)
                ? candidate.largeTextPrefixes.filter((value): value is string => typeof value === 'string' && value.length > 0 && value.length <= MAX_MATCH_TEXT_EDGE_LENGTH).slice(0, 32).map(value => value.toLowerCase())
                : [];
            matcher = {kind: 'legacy', match: candidate.match, largeTextPrefixes};
        } else {
            return null;
        }
        if (!isNonEmptyString(candidate.subTitle)) {
            return null;
        }
        if (candidate.icon !== null && typeof candidate.icon !== "string") {
            return null;
        }

        switch (candidate.type) {
            case 'executable':
                if (typeof candidate.execute !== 'function') return null;
                return {
                    type: candidate.type,
                    subTitle: candidate.subTitle,
                    matcher,
                    execute: candidate.execute,
                    icon: candidate.icon,
                };
            case 'ui': {
                const file = normalizePluginAssetPath(candidate.file);
                if (!file) return null;
                if (!await doesPluginAssetExist(`${plugin.homeUrl}/${file}`)) {
                    console.error(`Plugin UI file not found for ${plugin.packageId}: ${file}`);
                    return null;
                }
                return {
                    type: candidate.type,
                    subTitle: candidate.subTitle,
                    matcher,
                    icon: candidate.icon,
                    file,
                };
            }
            default:
                return null;
        }
    }));

    return sanitizedEntries.filter((entry): entry is PluginEntry => entry !== null);
}
