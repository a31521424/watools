import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './app'
import './style.css'
import {legacyRuntime,createPluginRuntime} from './lib/legacy-runtime'

// Existing trusted .wt plugins receive this compatibility API in their iframe.
const sessionId=new URLSearchParams(window.location.search).get('pluginSession')
Object.assign(window, {runtime: sessionId ? createPluginRuntime(`plugin-${sessionId}`,`${sessionId}:shell`) : legacyRuntime})


const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
