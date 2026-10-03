import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import wails from '@wailsio/runtime/plugins/vite'
// @ts-ignore
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        react(),
        wails("./bindings"),
        tailwindcss(),
    ],
    build: {
        // Desktop-only Wails shell; current main bundle size is acceptable.
        // Keep the warning threshold above the current lucide-driven bundle,
        // but low enough to still catch real regressions.
        chunkSizeWarningLimit: 900,
    },
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        }
    },
    server: {
        host: '127.0.0.1',
        port: Number(process.env.WAILS_VITE_PORT) || 9245,
        strictPort: true,
        watch: {
            ignored: [
                '**/node_modules/**',
                '**/dist/**',
                '**/build/**',
                '**/.git/**',
                '**/.DS_Store',
                '**/*.log',
                '**/*.log.*',
                '../app.go',
                '../main.go',
                '../go.mod',
                '../go.sum',
                '../wails.json',
                '../internal/**',
                '../pkg/**',
            ],
        },
    }
})
