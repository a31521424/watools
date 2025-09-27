import {defineConfig} from 'vite'
import {readdirSync} from "node:fs";
import react from "@vitejs/plugin-react";

const pluginDir = 'src/plugins'
const pluginFiles = readdirSync(pluginDir, {withFileTypes: true}).filter(file => file.name.endsWith('.tsx')).map(file => file.name.replace('.tsx', ''))

console.log('Building plugins:', pluginFiles)

const entries = pluginFiles.reduce((acc, plugin) => {
    acc[plugin] = `${pluginDir}/${plugin}.tsx`
    return acc
}, {} as Record<string, string>)

// https://vitejs.dev/config/


export default defineConfig({
    plugins: [react()],
    define: {
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
    },
    build: {
        outDir: 'dist-plugins',
        emptyOutDir: true,
        lib: {
            entry: entries,
            formats: ['es'],
            fileName: (_, entryName) => `${entryName}.js`,
        },
        rollupOptions: {
            input: entries,
            external: [
                'react',
                'react-dom',
                'react-dom/client',
                'tailwindcss',
                'tailwind-merge',
                'tailwind-scrollbar-hide'
            ],
            output: {
                format: 'es',
                inlineDynamicImports: true,
            },
        },
        // minify: true,
        // sourcemap: true,
    }
})
