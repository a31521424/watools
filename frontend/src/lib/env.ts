export function isDevMode(): boolean {
    if (typeof process !== "undefined" && process.env && process.env.NODE_ENV) {
        return process.env.NODE_ENV === "development";
    }
    if (typeof import.meta !== "undefined" && import.meta.env && import.meta.env.MODE) {
        return import.meta.env.MODE === "development";
    }
    return false;
}