import {copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync} from 'node:fs';
import path from 'node:path';

// Follows the v3 macOS template: native Go binary, .app layout, icon/plist,
// then ad-hoc codesigning. It never installs or launches the produced bundle.
export async function buildMacBundle({root, run, wails, debug = false, development = false, server = false}) {
    const project = JSON.parse(readFileSync(path.join(root, 'wails.json'), 'utf8'));
    const output = path.join(root, development ? '.cache/dev/bin' : 'build/bin');
    const bundle = path.join(output, server ? 'watools.browser.app' : development ? 'watools.dev.app' : 'watools.app');
    const contents = path.join(bundle, 'Contents');
    mkdirSync(path.join(contents, 'MacOS'), {recursive: true});
    mkdirSync(path.join(contents, 'Resources'), {recursive: true});
    await run('go', ['build', ...(debug ? ['-gcflags=all=-l', '-tags', server ? 'server' : 'private_mac_apis'] : ['-tags', 'production,private_mac_apis', '-trimpath', '-ldflags=-w -s']), '-buildvcs=false', '-o', path.join(contents, 'MacOS/watools'), '.']);
    const icon = path.join(output, 'icons.icns');
    await wails('generate', 'icons', '-input', 'build/appicon.png', '-macfilename', icon, '-windowsfilename', path.join(output, 'icon.ico'));
    copyFileSync(icon, path.join(contents, 'Resources/icons.icns'));
    const values = {
        ProductName: project.info.productName,
        BinaryName: 'watools',
        ProductIdentifier: 'com.wails.watools',
        ProductVersion: project.version,
        ProductComments: project.info.comments,
        ProductCopyright: project.info.copyright,
    };
    const xml = value => String(value).replace(/[<>&"']/g, char => ({'<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;', "'": '&apos;'}[char]));
    const template = readFileSync(path.join(root, 'build/darwin', debug ? 'Info.dev.plist' : 'Info.plist'), 'utf8');
    const plist = template.replace(/{{\.(\w+)}}/g, (_, key) => {
        if (!(key in values)) throw new Error(`Unknown macOS plist field: ${key}`);
        return xml(values[key]);
    });
    if (plist.includes('{{')) throw new Error('Unsupported macOS plist template directive.');
    writeFileSync(path.join(contents, 'Info.plist'), plist);
    if (existsSync(path.join(root, 'build/darwin/Assets.car'))) copyFileSync(path.join(root, 'build/darwin/Assets.car'), path.join(contents, 'Resources/Assets.car'));
    await run('codesign', ['--force', '--deep', '--sign', '-', bundle]);
    console.log(`Built ${bundle}`);
}
