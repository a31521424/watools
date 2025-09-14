import React from 'react'
import ReactDOM from 'react-dom/client'
import {twMerge} from 'tailwind-merge'
import tailwindScrollbarHide from 'tailwind-scrollbar-hide'
import {PluginRuntime} from "@/shared/plugin-interface";


const pluginRuntime: PluginRuntime = {
    React,
    ReactDOM,
    twMerge,
    tailwindScrollbarHide
}

window.PluginRuntime = pluginRuntime
