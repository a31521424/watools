import {defineConfig} from 'vite';
import * as path from "node:path";
import {createRequire} from "node:module";
import * as fs from "node:fs";

const require = createRequire(import.meta.url);

const VENDOR_ENTRIES = {
    'react': 'react',
    'react-dom-client': 'react-dom/client',
    'lucide-react': 'lucide-react',
    'radix-ui': 'radix-ui',
    'zustand': 'zustand',
};

const entries = Object.fromEntries(
    Object.entries(VENDOR_ENTRIES).map(([key, value]) => [key, require.resolve(value)])
);

const importMap = Object.fromEntries(
    Object.entries(VENDOR_ENTRIES).map(([_, value]) => [value, `./vendor/${value}.js`])
)

const importMapJs = ` // AUTO-GENERATED Import Map
(function() {
    const importMap = ${JSON.stringify({
    imports: importMap
}, null, 4)};
    const script = document.createElement('script');
    script.type = 'importmap';
    script.textContent = JSON.stringify(importMap, null, 2);
    document.head.appendChild(script);
    
    console.log('Import Map loaded:', importMap);
})()
`

fs.writeFileSync(path.resolve(__dirname, 'public', 'importmap.js'), importMapJs);


export default defineConfig({
    publicDir: false,
    build: {
        outDir: path.resolve(__dirname, 'public', 'vendor'),
        emptyOutDir: true,

        lib: {
            entry: entries,
            formats: ['es'],
            fileName: (_, entryName) => `${entryName}.js`,
        },

        rollupOptions: {
            output: {
                chunkFileNames: 'shared/chunk-[hash].js',
            }
        },
        minify: true,
    }
});