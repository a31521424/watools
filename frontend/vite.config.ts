import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
// @ts-ignore
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        react(),
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
        proxy: {
            '/api/': {
                bypass: () => {
                    return false
                }
            }
        },
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
