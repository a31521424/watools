import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './app'
import './style.css'
import {legacyRuntime} from './lib/legacy-runtime'

// Existing trusted .wt plugins receive this compatibility API in their iframe.
Object.assign(window, {runtime: legacyRuntime})


const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
