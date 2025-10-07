import {defineConfig} from "vite";
import * as fs from "node:fs";
import path from "node:path";

const VENDOR_ENTRY = [
    'react',
    'react-dom',
    'react-dom/client',
    'tailwindcss',
    'tailwind-merge',
    'tailwind-scrollbar-hide'
]

const generateImportMap = () => {
    const requireMap = VENDOR_ENTRY.reduce((acc, current) => {
        acc[current] = `./vendor/${current.replace('/', '-')}.js`
        return acc
    }, {} as Record<string, string>)

    const importMap = ` // AUTO-GENERATED Import Map
(function() {
    const importMap = ${JSON.stringify(requireMap, null, 2)};
    const script = document.createElement('script');
    script.type = 'importmap';
    script.textContent = JSON.stringify(importMap, null, 2);
    document.head.appendChild(script);

    console.log('Import Map loaded:', importMap);
})()
`
    const publicDir = path.resolve(__dirname, 'public')
    if (!fs.existsSync(publicDir)) {
        fs.mkdirSync(publicDir, {recursive: true})
    }

    fs.writeFileSync(path.resolve(__dirname, 'public', 'importmap.js'), importMap)
}

const parseVendorEsm = (vendor: string) => {
    const [pureVendor, extraPath] = vendor.split("/")

    // 对于子路径依赖，使用主包的版本号
    const packagePath = path.resolve(__dirname, 'node_modules', pureVendor, 'package.json')
    const packageJson = fs.readFileSync(packagePath, 'utf-8')
    const version = JSON.parse(packageJson).version

    if (extraPath) {
        return `${pureVendor}@${version}/${extraPath}`
    }
    return `${pureVendor}@${version}`
}


const downloadVendor = async () => {
    const vendorDir = path.resolve(__dirname, 'public', 'vendor')

    // 确保目录存在
    if (!fs.existsSync(vendorDir)) {
        fs.mkdirSync(vendorDir, {recursive: true})
    }

    for (const vendor of VENDOR_ENTRY) {
        const vendorPath = parseVendorEsm(vendor)
        const filePath = path.resolve(vendorDir, `${vendor.replace('/', '-')}.js`)
        try {
            // 使用 ?bundle 参数获取完整的 bundle，避免重定向
            const response = await fetch(`https://esm.sh/${vendorPath}?bundle`)
            if (!response.ok) {
                console.error(`Failed to access ${vendorPath}`)
                return
            }
            const content = await response.text()
            fs.writeFileSync(filePath, content)
            console.log(`Downloaded ${filePath} success`)
        } catch (e) {
            console.error(`Failed to download ${filePath}: ${e}`)
            return
        }
    }
}

export default defineConfig({
    plugins: [{
        name: "custom-prepare-vendor",
        buildStart: async () => {
            console.log('Building vendor...')
            await downloadVendor()
            generateImportMap()
        }
    }],
    publicDir: false,
    build: {
        write: false,
        rollupOptions: {
            input: 'data:text/javascript,// build vendor',
            external: (id) => id !== 'data:text/javascript,// build vendor',
            output: {}
        }
    }
})