import React from "react";
import ReactDOM from "react-dom/client";

export interface SharedLibs {
    React: typeof React
    ReactDOM: typeof ReactDOM
}


declare global {
    interface Window {
        sharedLibs: SharedLibs,
    }
}


window.sharedLibs = {
    React,
    ReactDOM,
}
