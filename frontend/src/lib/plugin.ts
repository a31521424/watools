import {Plugin, PluginEntry} from "@/schemas/plugin";

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
    try {
        const response = await fetch(assetUrl, {method: "HEAD"});
        return response.ok;
    } catch (error) {
        console.error(`Failed to validate plugin asset ${assetUrl}:`, error);
        return false;
    }
}

export const sanitizePluginEntries = async (plugin: Plugin, entries: unknown[]): Promise<PluginEntry[]> => {
    const sanitizedEntries = await Promise.all(entries.map(async (entry): Promise<PluginEntry | null> => {
        if (!entry || typeof entry !== "object") {
            return null;
        }

        const candidate = entry as Partial<PluginEntry>;
        if (candidate.type !== "executable" && candidate.type !== "ui") {
            return null;
        }
        if (!isNonEmptyString(candidate.subTitle)) {
            return null;
        }
        if (candidate.icon !== null && typeof candidate.icon !== "string") {
            return null;
        }

        if (candidate.type === "executable") {
            if (typeof candidate.execute !== "function" || typeof candidate.match !== "function") {
                return null;
            }
            return {
                type: candidate.type,
                subTitle: candidate.subTitle,
                match: candidate.match,
                execute: candidate.execute,
                icon: candidate.icon ?? null,
            };
        }

        const file = normalizePluginAssetPath(candidate.file);
        if (!file || typeof candidate.match !== "function") {
            return null;
        }

        const assetExists = await doesPluginAssetExist(`${plugin.homeUrl}/${file}`);
        if (!assetExists) {
            console.error(`Plugin UI file not found for ${plugin.packageId}: ${file}`);
            return null;
        }

        return {
            type: candidate.type,
            subTitle: candidate.subTitle,
            match: candidate.match,
            icon: candidate.icon ?? null,
            file,
        };
    }));

    return sanitizedEntries.filter((entry): entry is PluginEntry => entry !== null);
}
