import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import {readdirSync} from 'node:fs';
import * as path from 'node:path';

const pluginDir = path.resolve(__dirname, 'src/plugins');
const entries = readdirSync(pluginDir, {withFileTypes: true})
    .filter(dirent => dirent.isFile() && dirent.name.endsWith('.tsx'))
    .reduce((acc, dirent) => {
        const name = dirent.name.replace('.tsx', '');
        acc[name] = path.resolve(pluginDir, dirent.name);
        return acc;
    }, {} as Record<string, string>);

console.log('Building plugins:', Object.keys(entries));

export default defineConfig(({mode}) => ({
    plugins: [
        react(),
    ],
    define: {
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
    },
    build: {
        outDir: 'dist-plugins',
        sourcemap: mode !== 'production',
        minify: mode === 'production',
        lib: {
            entry: entries,
            formats: ['iife'],
            name: '_WailsAppPlugins',
            fileName: (format, entryName) => `${entryName}.js`,
        },
        rollupOptions: {
            external: [
                'react',
                'react-dom',
                'react-dom/client',
            ],
            output: {
                globals: {
                    'react': 'sharedLibs.React',
                    'react-dom': 'sharedLibs.ReactDOM',
                    'react-dom/client': 'sharedLibs.ReactDOM',
                },
            },
        },
    },
}));