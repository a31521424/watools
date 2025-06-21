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
