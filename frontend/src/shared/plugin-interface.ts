import React from "react";
import ReactDOM from "react-dom/client";
import {twMerge} from "tailwind-merge";
import tailwindScrollbarHide from "tailwind-scrollbar-hide";

export interface PluginRuntime {
    React: typeof React
    ReactDOM: typeof ReactDOM
    twMerge: typeof twMerge
    tailwindScrollbarHide: typeof tailwindScrollbarHide
}

export interface WatoolsRuntime {
    ClipboardSetText: (text: string) => Promise<boolean>
}

declare global {
    interface Window {
        PluginRuntime: PluginRuntime,
        runtime: WatoolsRuntime
    }
}
