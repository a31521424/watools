const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const root = path.resolve(__dirname, '..');
const ts = require(path.join(root, 'frontend/node_modules/typescript'));
const tick = () => new Promise(resolve => setImmediate(resolve));

function loadTS(file, mocks = {}, globals = {}) {
    const output = ts.transpileModule(fs.readFileSync(path.join(root, file), 'utf8'), {
        compilerOptions: {module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022},
    }).outputText;
    const context = {...globals, exports: {}, console, setTimeout, clearTimeout, Map, Date, Promise, URLSearchParams,
        require(name) {
            if (name in mocks) return mocks[name];
            if (name.startsWith('@/lib/')) return loadTS('frontend/src/lib/' + name.slice(6) + '.ts', mocks);
            if (name === 'zustand') return require(path.join(root, 'frontend/node_modules/zustand'));
            throw new Error(`Unmocked import ${name}`);
        }};
    vm.runInNewContext(output, context);
    return context.exports;
}

test('clearing input cannot revive a pending query; typing discards clipboard attachments', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    store.getState().setTextValue('pending');
    store.getState().clearValue();
    await new Promise(resolve => setTimeout(resolve, 80));
    assert.equal(store.getState().value, '');
    store.getState().setClipboardContent({contentType: 'image', imageBase64: 'image'});
    store.getState().setTextValue('json');
    assert.equal(store.getState().value, 'json', 'raw input must be current before command execution');
    assert.equal(store.getState().clipboardContentType, null);
    assert.equal(store.getState().imageBase64, null);
});

function jsonCore() {
    const html = fs.readFileSync(path.join(root, 'plugins/official/watools.plugin.json/plugin/index.html'), 'utf8');
    const context = {URL, URLSearchParams, console};
    vm.createContext(context);
    vm.runInContext(html.slice(html.indexOf('    const formatPathSegment'), html.indexOf('    const findDeepestNodeAtOffset')) +
        '\nthis.parse = text => new JsonTreeParser(text).parse().value;', context);
    vm.runInContext(html.slice(html.indexOf('    const smartDecode'), html.indexOf('    const appendQueryPair')) +
        '\nthis.query = parseQueryCandidate;', context);
    return context;
}

test('JSON core rejects incomplete values and control characters', () => {
    const core = jsonCore();
    for (const input of ['{"a":1,', '[1,', '{"x":"raw\nnewline"}', '[1,]', '{"a":1,}', '01', 'true false']) {
        assert.throws(() => core.parse(input), `must reject ${JSON.stringify(input)}`);
    }
});

test('all JSON and query keys survive serialization as data', () => {
    const core = jsonCore();
    const input = '{"__proto__":{"hidden":true},"constructor":"a","toString":"b"}';
    assert.equal(JSON.stringify(core.parse(input)), input);
    assert.equal(JSON.stringify(core.query('__proto__=safe&constructor=abc&toString=def&constructor=again')),
        '{"__proto__":"safe","constructor":["abc","again"],"toString":"def"}');
});

module.exports = {loadTS, tick};

function windowKeyHandler(file, globals) {
    const source = ts.createSourceFile(file, fs.readFileSync(path.join(root, file), 'utf8'),
        ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    let initializer;
    function visit(node) {
        if (ts.isVariableDeclaration(node) && node.name.getText(source) === 'handleHotkey') {
            initializer = node.initializer.getText(source);
        }
        ts.forEachChild(node, visit);
    }
    visit(source);
    assert.ok(initializer, `keyboard handler missing in ${file}`);
    const context = {...globals, exports: {}};
    const output = ts.transpileModule(`exports.handle = ${initializer};`, {
        compilerOptions: {target: ts.ScriptTarget.ES2022},
    }).outputText;
    vm.runInNewContext(output, context);
    return context.exports.handle;
}

test('IME candidate keys leave launcher input and plugin/management routes intact', async () => {
    for (const name of ['wa-command', 'wa-plugin', 'wa-plugin-management', 'wa-log-management']) {
        const actions = [];
        const handler = windowKeyHandler(`frontend/src/components/watools/${name}.tsx`, {
            getIsPanelOpen: () => true, getCanClearAssets: () => true,
            clearValue: () => actions.push('clear'), navigate: () => actions.push('navigate'),
            inputRef: {current: {focus: () => actions.push('focus')}},
            Promise, flushApplicationUsage: async () => {}, flushPluginUsage: async () => {},
            host: {HideOrShowAppApi: () => actions.push('hide')},
        });
        for (const key of ['Escape', 'Tab', 'Backspace']) {
            for (const composition of [{isComposing: true, keyCode: 0}, {isComposing: false, keyCode: 229}]) {
                handler({key, ...composition, preventDefault: () => actions.push('prevent'),
                    stopPropagation: () => actions.push('stop')});
                assert.deepEqual(actions, [], `${name}: ${key} must remain available to the IME`);
            }
        }
        handler({key: 'Escape', isComposing: false, keyCode: 27,
            preventDefault() {}, stopPropagation() {}});
        await tick();
        assert.deepEqual(actions, [name === 'wa-command' ? 'clear' : 'navigate'],
            `${name}: ordinary Escape must retain its existing behavior`);
    }
});


test('a disabled plugin loads its entry on enable; writes during flush remain durable', async () => {
    let releaseFirst;
    const writes = [];
    const api = {
        getPlugins: async () => [{packageId: 'probe', enabled: false, entry: [], usedCount: 0}],
        togglePlugin: async () => {}, uninstallPlugin: async () => {}, disposePlugin: async () => {},
        loadPluginEntries: async plugin => { plugin.entry = [{type: 'ui', file: 'index.html'}]; plugin.loadStatus = 'ready'; },
        updatePluginUsage: updates => {
            writes.push(updates);
            return writes.length === 1 ? new Promise(resolve => {releaseFirst = resolve}) : Promise.resolve();
        },
    };
    const {usePluginStore: store} = loadTS('frontend/src/stores/pluginStore.ts', {
        '@/api/plugin': api, '@/lib/logger': {Logger: {error() {}}},
    });
    await tick();
    await store.getState().togglePlugin('probe', true);
    assert.equal(store.getState().plugins[0].entry.length, 1);
    const firstUpdate = store.getState().updatePluginUsage('probe');
    const flushing = store.getState().flushBufferUpdates();
    const secondUpdate = store.getState().updatePluginUsage('probe');
    const overlapping = store.getState().flushBufferUpdates();
    releaseFirst();
    await Promise.all([firstUpdate, secondUpdate, flushing, overlapping]);
    assert.equal(writes.length, 2);
    assert.equal(writes[1][0].usedCount, 2);
    assert.equal(store.getState().updateBuffer.size, 0);
});

test('usage retry preserves a failed snapshot', async () => {
    let buffer = new Map([['plugin', {usedCount: 1}]]);
    let attempts = 0;
    const {createUsageFlusher} = loadTS('frontend/src/lib/usage-buffer.ts');
    const flush = createUsageFlusher(() => buffer, snapshot => {
        for (const [key, value] of snapshot) if (buffer.get(key) === value) buffer.delete(key);
    }, async () => { if (++attempts === 1) throw new Error('disk unavailable'); });
    await assert.rejects(flush(), /disk unavailable/);
    assert.equal(buffer.size, 1);
    await flush();
    assert.equal(buffer.size, 0);
});

test('translation accepts the latest input while busy and ignores cleared/stale responses', async () => {
    const html = fs.readFileSync(path.join(root, 'plugins/official/watools.plugin.translate/plugin/index.html'), 'utf8');
    const pending = [];
    const context = {console, Date, sourceTextNode:{value:'first'}, targetTextNode:{value:''},
        detectedLabel:{textContent:''}, apiKey:'secret', isTranslating:false, latestRequestId:0,
        activeHttpRequestId:null, lastTargetLang:'EN', deeplEndpoint:'https://example.com',
        setStatus(){}, setSettingsVisible(){}, updateDirectionUI(){},
        window:{watools:{CancelHttpProxy:async()=>{}, HttpProxy:()=>new Promise(resolve=>pending.push(resolve))}}};
    vm.createContext(context);
    const invalidate = html.slice(html.indexOf('    const invalidateTranslation'), html.indexOf('    };', html.indexOf('    const invalidateTranslation'))+7);
    const translate = html.slice(html.indexOf('    const translateText ='), html.indexOf('    const queueAutoTranslate ='));
    vm.runInContext(invalidate + translate + '\nthis.translate = translateText; this.invalidate = invalidateTranslation;', context);
    const old = context.translate();
    context.sourceTextNode.value = 'second'; context.invalidate();
    const latest = context.translate();
    assert.equal(pending.length, 2, 'busy translation must not discard new input');
    const response = text => ({status_code:200,body:JSON.stringify({translations:[{text}]})});
    pending[1](response('new result')); await latest;
    pending[0](response('old result')); await old;
    assert.equal(context.targetTextNode.value,'new result');
    context.sourceTextNode.value = 'third'; const outstanding=context.translate();
    context.sourceTextNode.value = ''; await context.translate();
    pending[2](response('revived')); await outstanding;
    assert.equal(context.targetTextNode.value,'');
});

test('secret migration deletes legacy copies only after a successful host write', async () => {
    const html=fs.readFileSync(path.join(root,'plugins/official/watools.plugin.translate/plugin/index.html'),'utf8');
    const calls=[];let fail=true;
    const context={apiKeyStorageKey:'key',apiKeyLocalStorageKey:'legacy',window:{
        localStorage:{removeItem:()=>calls.push('local-remove')},watools:{
            SecretSet:async()=>{calls.push('secure-write');if(fail)throw Error('locked')},
            StorageRemove:async()=>calls.push('database-remove')
        }}};
    vm.createContext(context);
    vm.runInContext(html.slice(html.indexOf('    const persistApiKey ='),html.indexOf('    const loadApiKey ='))+'\nthis.persist=persistApiKey;',context);
    await assert.rejects(context.persist('private'),/locked/);
    assert.deepEqual(calls,['secure-write']);
    fail=false;calls.length=0;await context.persist('private');
    assert.deepEqual(calls,['secure-write','database-remove','local-remove']);
});

test('plugin launch context is a stable in-memory snapshot and can be released', () => {
    const {buildPluginContext,persistPluginLaunchContext,readPluginLaunchContext,releasePluginLaunchContext}=loadTS('frontend/src/lib/plugin-context.ts');
    const initial=buildPluginContext('private input','clipboard','files',null,['first']);
    const id=persistPluginLaunchContext(initial);initial.input.value='cleared';initial.clipboard.files.push('second');
    const stored=readPluginLaunchContext(id);
    assert.equal(stored.input.value,'private input');assert.equal(stored.clipboard.files.length,1);
    releasePluginLaunchContext(id);assert.equal(readPluginLaunchContext(id),null);
});

test('async executable plugins retain their own API and restore the host API', async () => {
    const original={packageId:'host'}, host={watools:original};let release;
    const observations=[];
    const {withPluginApi}=loadTS('frontend/src/lib/plugin-execution.ts', {
        '@/api/api':{createWaToolsApi:packageId=>({packageId})},
    },{window:host});
    const first=withPluginApi('first',async api=>{
        observations.push([api.packageId,host.watools.packageId]);
        await new Promise(resolve=>release=resolve);
        observations.push([api.packageId,host.watools.packageId]);
    });
    const second=withPluginApi('second',api=>observations.push([api.packageId,host.watools.packageId]));
    await tick();assert.equal(observations.length,1);release();await Promise.all([first,second]);
    assert.deepEqual(observations,[['first','first'],['first','first'],['second','second']]);
    assert.equal(host.watools,original);
    await assert.rejects(withPluginApi('broken',()=>{throw Error('broken')}),/broken/);
    await withPluginApi('after-error',()=>{});assert.equal(host.watools,original);
});

test('late clipboard and focus reads cannot replace cleared or newly typed input', () => {
    const {useAppStore:store}=loadTS('frontend/src/stores/appStore.ts');
    const initialRevision=store.getState().inputRevision;
    store.getState().clearValue();
    store.getState().setValueAuto('late focus','clipboard',undefined,initialRevision);
    store.getState().setClipboardContent({contentType:'text',text:'late paste'},initialRevision);
    assert.equal(store.getState().value,'');
    const revision=store.getState().inputRevision;
    store.getState().setTextValue('new input');
    store.getState().setClipboardContent({contentType:'image',imageBase64:'stale image'},revision);
    assert.equal(store.getState().value,'new input');assert.equal(store.getState().imageBase64,null);
    const current=store.getState().inputRevision;
    store.getState().setClipboardContent({contentType:'text',text:'fresh paste'},current);
    assert.equal(store.getState().value,'fresh paste');assert.equal(store.getState().valueType,'clipboard');
});

test('v2 plugin runtime preserves clipboard booleans, window sizes and variadic event payloads', async () => {
    const callbacks = new Map();
    let failClipboard = false;
    let target;
    const cancelled = [];
    const register = (name, callback) => {callbacks.set(name, callback); return () => {callbacks.delete(name); cancelled.push(name)}};
    const {legacyRuntime: runtime} = loadTS('frontend/src/lib/legacy-runtime.ts', {
        '@wailsio/runtime': {
            Application: {}, Browser: {}, Screens: {}, System: {},
            Clipboard: {Text: async () => 'probe', SetText: async () => {if (failClipboard) throw Error('clipboard unavailable')}},
            Window: {Get: name => {target = name; return {Size: async () => ({width: 920, height: 64})}}},
            Events: {On: register, Once: register, OnMultiple: register,
                Emit: async (name, data) => callbacks.get(name)?.({name, data}),
                Off: (...names) => names.forEach(name => callbacks.delete(name)), OffAll: () => callbacks.clear()},
        },
        '@/api/host': {host: {LogFrontendApi: async () => {}, GetLauncherSizeApi: async () => ({width:920, height:64})}},
    });
    assert.equal(target, 'launcher');
    assert.equal(await runtime.ClipboardSetText('probe'), true);
    failClipboard = true;
    await assert.rejects(runtime.ClipboardSetText('probe'), /clipboard unavailable/);
    assert.equal(JSON.stringify(await runtime.WindowGetSize()), '{"w":920,"h":64}');
    const observations = [];
    const off = runtime.EventsOn('probe', (...args) => observations.push(args));
    await runtime.EventsEmit('probe', 'first', 3);
    callbacks.get('probe')({data: {value: 'host object'}});
    callbacks.get('probe')({data: null});
    assert.equal(JSON.stringify(observations), '[["first",3],[{"value":"host object"}],[]]');
    off();
    await runtime.EventsEmit('probe', 'ignored');
    assert.equal(observations.length, 3);
    assert.deepEqual(cancelled, ['probe']);
});

test('mock Host API and plugin SDK preserve scoped JSON, nulls and disabled-plugin errors', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {URL, TextEncoder});
    const host = createMockHost();
    const {createWaToolsApi} = loadTS('frontend/src/api/api.ts', {'@/api/host': {host}});
    const sdk = createWaToolsApi('watools.plugin.mock');
    const value = JSON.parse('{"__proto__":"data","nested":[1,null]}');
    await sdk.StorageSet('probe', value);
    value.nested.push('changed');
    assert.equal(JSON.stringify(await sdk.StorageGet('probe')), '{"__proto__":"data","nested":[1,null]}');
    await sdk.StorageSet('probe', null);
    assert.equal(await sdk.StorageGet('probe'), null);
    assert.equal(JSON.stringify(await sdk.StorageKeys()), '["probe"]');
    await assert.rejects(sdk.StorageSet('missing', undefined), error => error.code === 'INVALID_ARGUMENT');
    await sdk.SecretSet('key', 'synthetic-value');
    assert.equal(await sdk.SecretGet('key'), 'synthetic-value');
    await host.TogglePluginApi('watools.plugin.mock', false);
    await assert.rejects(sdk.StorageGet('probe'), error => error.code === 'PLUGIN_DISABLED');
    await assert.rejects(sdk.SecretGet('key'), error => error.code === 'PLUGIN_DISABLED');
    assert.equal((await host.GetPluginsApi())[0].enabled, false);
    await host.TogglePluginApi('watools.plugin.mock', true);
    await sdk.StorageClear();
    assert.equal(JSON.stringify(await sdk.StorageKeys()), '[]');
    await sdk.SecretRemove('key');
    assert.equal(await sdk.SecretGet('key'), '');
    const another = createMockHost();
    assert.equal(JSON.stringify(await another.ListPluginStorageKeysApi({packageId:'watools.plugin.mock'})), '[]');
    assert.equal((await sdk.HttpProxy({url:'https://example.invalid', method:'GET'})).status_code, 200);
    await assert.rejects(sdk.OpenFolder('/mock'), error => error.code === 'UNAVAILABLE');
});

test('Host API usage validates the complete batch and returns independent snapshots', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {URL, TextEncoder});
    const host = createMockHost();
    const valid = {id:'mock-app', lastUsedAt:'2026-10-04T00:00:00Z', usedCount:2};
    for (const usedCount of [-1, 1.5, undefined, 1e12+1]) {
        await assert.rejects(host.UpdateApplicationUsageApi([valid, {...valid, usedCount}]), error => error.code === 'INVALID_ARGUMENT');
        assert.equal((await host.GetApplicationCommandsApi())[0].usedCount, 0);
    }
    await host.UpdateApplicationUsageApi([valid]);
    const snapshot = await host.GetApplicationCommandsApi(); snapshot[0].usedCount = 99;
    assert.equal((await host.GetApplicationCommandsApi())[0].usedCount, 2);
    await assert.rejects(host.HttpProxyApi({url:'https://example.invalid', timeout:1.5}), error => error.code === 'INVALID_ARGUMENT');
    await assert.rejects(host.HttpProxyApi({url:'file:///tmp/data'}), error => error.code === 'INVALID_ARGUMENT');
});

test('Wails error normalization retains context and displayable codes without raw transport messages', async () => {
    const {withHostErrors, HostError} = loadTS('frontend/src/api/host/errors.ts');
    const original = new Error('raw internal transport details', {cause:{code:'TIMEOUT',message:'操作超时，请重试',operation:'system.http'}});
    const host = withHostErrors({request:async () => {throw original}});
    await assert.rejects(host.request(), error => error instanceof HostError && error.code === 'TIMEOUT' && error.operation === 'system.http' && error.message === '操作超时，请重试' && error.cause === original);
    const broken = withHostErrors({request:async () => {throw new Error('private raw data')}});
    await assert.rejects(broken.request(), error => error.code === 'TRANSPORT_ERROR' && !error.message.includes('private'));
});

test('generated bindings stay confined to the Wails adapter and both adapters cover every public method', () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {URL, TextEncoder});
    const host = createMockHost();
    const bound = fs.readdirSync(path.join(root, 'frontend/bindings/watools/internal/host')).filter(name => name.endsWith('service.ts')).flatMap(name => {
        const text = fs.readFileSync(path.join(root, 'frontend/bindings/watools/internal/host',name),'utf8');
        return [...text.matchAll(/export function (\w+)\(/g)].map(match => match[1]);
    });
    assert.deepEqual(Object.keys(host).sort(), bound.sort());
    function check(directory) {
        for (const entry of fs.readdirSync(directory, {withFileTypes:true})) {
            const file = path.join(directory, entry.name);
            if (entry.isDirectory()) check(file);
            else if (/\.tsx?$/.test(file) && file !== path.join(root,'frontend/src/api/host/wails.ts')) {
                assert.ok(!fs.readFileSync(file,'utf8').includes('bindings/watools/'), `${file} bypasses Host API`);
            }
        }
    }
    check(path.join(root,'frontend/src'));
});

test('mock rejects null mutations, oversized UTF-8 bodies and impossible calendar dates like the real host', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {URL, TextEncoder});
    const host = createMockHost();
    const id = 'watools.plugin.mock';
    await host.SetPluginSecretApi(id, 'key', 'synthetic-original');
    for (const enabled of [null, 'false', 0]) await assert.rejects(host.TogglePluginApi(id, enabled), error => error.code === 'INVALID_ARGUMENT');
    await assert.rejects(host.SetPluginSecretApi(id, 'key', null), error => error.code === 'INVALID_ARGUMENT');
    assert.equal((await host.GetPluginsApi())[0].enabled, true);
    assert.equal(await host.GetPluginSecretApi(id, 'key'), 'synthetic-original');
    for (const body of ['x'.repeat(4*1024*1024+1), '中'.repeat(1400000)]) {
        await assert.rejects(host.HttpProxyApi({url:'https://example.invalid', body}), error => error.code === 'INVALID_ARGUMENT');
    }
    for (const lastUsedAt of ['2026-02-30T00:00:00Z', '2026-04-31T00:00:00Z']) {
        await assert.rejects(host.UpdatePluginUsageApi([{packageId:id,lastUsedAt,usedCount:1}]), error => error.code === 'INVALID_ARGUMENT');
    }
    await host.UpdatePluginUsageApi([{packageId:id,lastUsedAt:'2024-02-29T00:00:00Z',usedCount:1}]);
    assert.equal((await host.GetPluginsApi())[0].usedCount, 1);
});

test('mock launcher sizing uses Host API and preserves the legacy runtime size shape', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {URL, TextEncoder});
    const host = createMockHost();
    const {legacyRuntime: runtime} = loadTS('frontend/src/lib/legacy-runtime.ts', {
        '@wailsio/runtime': {Application:{},Browser:{},Clipboard:{},Events:{},Screens:{},System:{},Window:{Get:()=>({Size:()=>{throw Error('must not call Wails in mock mode')}})}},
        '@/api/host': {host},
    });
    await runtime.WindowSetSize(920,760);
    assert.equal(JSON.stringify(await runtime.WindowGetSize()), '{"w":920,"h":760}');
    await assert.rejects(host.SetLauncherSizeApi(0,760), error => error.code === 'INVALID_ARGUMENT');
    assert.equal(JSON.stringify(await runtime.WindowGetSize()), '{"w":920,"h":760}');
});
