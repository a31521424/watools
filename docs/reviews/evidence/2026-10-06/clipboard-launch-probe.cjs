const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const os = require('node:os');
const {spawnSync} = require('node:child_process');
const {performance} = require('node:perf_hooks');
const root = process.env.WATOOLS_PROBE_ROOT || process.cwd();
const Fuse = require(path.join(root, 'frontend/node_modules/fuse.js'));
const ts = require(path.join(root, 'frontend/node_modules/typescript'));
function loadTS(file) {
    const code = ts.transpileModule(fs.readFileSync(path.join(root, file), 'utf8'), {
        compilerOptions: {module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022},
    }).outputText;
    const context = {exports: {}, Map, Date, console, require(name) {
        if (name === 'zustand') return require(path.join(root, 'frontend/node_modules/zustand'));
        throw Error(`Unexpected runtime import: ${name}`);
    }};
    vm.runInNewContext(code, context);
    return context.exports;
}
function input(shape, length) {
    const unit = {words: 'alpha beta gamma delta ', unbroken: 'a', json: '  "key": "value",\n'}[shape];
    return unit.repeat(Math.ceil(length / unit.length)).slice(0, length);
}
function measure(run) {
    const start = performance.now();
    const result = run();
    return {elapsedMs: +(performance.now() - start).toFixed(3), result};
}
if (process.argv[2] === '--case') {
    const [stage, shape, size] = process.argv.slice(3);
    const value = input(shape, +size);
    let run;
    if (stage === 'extended-parser') {
        const fuse = new Fuse([], {keys: ['name'], useExtendedSearch: true});
        run = () => fuse.search(value, {limit: 10}).length;
    } else if (stage === 'application-search' || stage === 'bounded-query' || stage === 'application-no-extended') {
        const commands = Array.from({length: 100}, (_, index) => ({
            name: `Application ${index}`, namePinyin: `application${index}`,
            nameInitial: `app${index}`, pathName: `Application${index}.app`,
        }));
        const source = fs.readFileSync(path.join(root, 'frontend/src/stores/applicationCommandStore.ts'), 'utf8');
        const declaration = source.slice(source.indexOf('export const WaApplicationCommandFuseConfig'), source.indexOf('\ntype ApplicationCommandState'));
        const sandbox = {exports: {}};
        vm.runInNewContext(ts.transpileModule(declaration, {compilerOptions: {module: ts.ModuleKind.CommonJS}}).outputText, sandbox);
        const config = {...sandbox.exports.WaApplicationCommandFuseConfig};
        if (stage === 'application-no-extended') config.useExtendedSearch = false;
        const fuse = new Fuse(commands, config);
        run = () => fuse.search(stage === 'bounded-query' ? value.slice(0, 128) : value, {limit: 10}).length;
    } else if (stage === 'store-auto' || stage === 'store-paste') {
        const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
        run = () => {
            if (stage === 'store-auto') store.getState().setValueAuto(value, 'clipboard');
            else store.getState().setClipboardContent({contentType: 'text', text: value, imageBase64: null, files: null});
            return {originLength: store.getState().value.length, displayLength: store.getState().displayValue.length};
        };
    } else if (stage === 'ranking') {
        const {createRankingInputContext} = loadTS('frontend/src/lib/command-ranking.ts');
        run = () => createRankingInputContext({value, valueType: 'clipboard', clipboardContentType: 'text'}).key.length;
    } else if (stage === 'launch-json' || stage === 'launch-shallow') {
        const {buildPluginContext, persistPluginLaunchContext} = loadTS('frontend/src/lib/plugin-context.ts');
        const context = buildPluginContext(value, 'clipboard', 'text');
        run = stage === 'launch-json' ? () => Boolean(persistPluginLaunchContext(context)) : () => {
            const snapshot = {input: {...context.input}, clipboard: {...context.clipboard, files: context.clipboard.files?.slice() ?? null}};
            return snapshot.input.value.length;
        };
    } else if (stage === 'official-match') {
        const matches = [];
        const calculator = loadTS('plugins/official/watools.plugin.calculator/plugin/calculator-core.js');
        for (const name of ['json', 'textstats', 'translate', 'calculator', 'common', 'qr']) {
            let source = fs.readFileSync(path.join(root, `plugins/official/watools.plugin.${name}/plugin/app.js`), 'utf8');
            source = source.replace(/^import .*;\n/gm, '').replace('export default entry;', 'this.entries = entry;');
            const context = {...calculator};
            vm.runInNewContext(source, context);
            matches.push(...context.entries.map(entry => entry.match));
        }
        run = () => matches.filter(match => match({input: {value, valueType: 'clipboard', clipboardContentType: 'text'}, clipboard: {contentType: 'text', text: value}})).length;
    } else throw Error(`Unknown stage: ${stage}`);
    console.log(JSON.stringify({stage, shape, length: +size, ...measure(run)}));
} else {
    const rows = [];
    const cases = [];
    for (const shape of ['words', 'unbroken', 'json']) {
        for (const length of [2000, 20000, 100000, 1000000]) cases.push(['extended-parser', shape, length]);
    }
    for (const length of [2000, 20000, 100000, 1000000]) {
        cases.push(['application-search', 'words', length], ['application-search', 'unbroken', length], ['bounded-query', 'words', length]);
    }
    cases.push(['application-no-extended', 'words', 100000]);
    for (const stage of ['store-auto', 'store-paste', 'ranking', 'launch-json', 'launch-shallow', 'official-match']) {
        for (const length of [100000, 1000000, 10000000]) cases.push([stage, 'words', length]);
    }
    for (const [stage, shape, length] of cases) {
        const child = spawnSync(process.execPath, [__filename, '--case', stage, shape, String(length)], {timeout: 5000, encoding: 'utf8'});
        const row = child.status === 0 ? JSON.parse(child.stdout) : {
            stage, shape, length, timeoutMs: 5000, status: child.error?.code || `exit-${child.status}`, error: child.stderr.trim().slice(0, 300),
        };
        rows.push(row);
        console.error(JSON.stringify(row));
    }
    const store = loadTS('frontend/src/stores/appStore.ts').useAppStore;
    const original = '  exact text\n';
    store.getState().setValueAuto(original, 'clipboard');
    const autoPreservesWhitespace = store.getState().value === original;
    store.getState().clearValue();
    store.getState().setClipboardContent({contentType: 'text', text: original, imageBase64: null, files: null});
    const pastePreservesWhitespace = store.getState().value === original;
    store.getState().clearValue();
    store.getState().setValueAuto('x'.repeat(1000000), 'clipboard');
    store.getState().setTextValue(store.getState().displayValue + '!');
    const editedOriginLength = store.getState().value.length;
    console.log(JSON.stringify({environment: {node: process.version, platform: process.platform, arch: process.arch, cpu: os.cpus()[0]?.model, fuse: Fuse.version},
        methodology: 'Single cold operation per isolated Node process; 100 synthetic applications; 5s child timeout includes startup/transpilation; no native clipboard or WebKit access.',
        semantics: {autoPreservesWhitespace, pastePreservesWhitespace, editedOriginLength}, rows}, null, 2));
}
