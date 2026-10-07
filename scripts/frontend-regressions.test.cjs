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
        fileName: file,
        compilerOptions: {module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX},
    }).outputText;
    const context = {...globals, exports: {}, console, setTimeout, clearTimeout, queueMicrotask, Map, Date, Promise, URLSearchParams,
        require(name) {
            if (name in mocks) return mocks[name];
            if (name.startsWith('@/lib/')) return loadTS('frontend/src/lib/' + name.slice(6) + '.ts', mocks);
            if (name === 'zustand') return require(path.join(root, 'frontend/node_modules/zustand'));
            throw new Error(`Unmocked import ${name}`);
        }};
    Object.assign(context, globals);
    vm.runInNewContext(output, context);
    return context.exports;
}

test('clearing input cannot revive a pending query; typing discards clipboard attachments', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    store.getState().setTextValue('pending');
    assert.equal(store.getState().clearValue(), true);
    await new Promise(resolve => setTimeout(resolve, 80));
    assert.equal(store.getState().input.kind, 'inline');
    assert.equal(store.getState().input.context.input.value, '');
    store.getState().setClipboardContent({contentType: 'image', imageBase64: 'image'});
    store.getState().setTextValue('json');
    assert.equal(store.getState().input.context.input.value, 'json', 'raw input must be current before command execution');
    assert.equal(store.getState().input.context.clipboard, null);
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

test('inline clipboard and paste keep exact origins, bounded displays and Unicode edges', () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    const {getLauncherPresentation} = loadTS('frontend/src/lib/launcher-input.ts');
    const origin = '  \0😀' + 'word \n'.repeat(170000) + '😀\n  ';
    store.getState().setClipboardContent({contentType: 'text', text: origin});
    const input = store.getState().input;
    assert.equal(input.kind, 'inline');
    assert.equal(input.context.input.value, origin);
    assert.equal(input.context.clipboard.text, origin);
    const presentation = getLauncherPresentation(input);
    assert.ok(presentation.displayValue.length < 140);
    assert.ok(presentation.displayValue.isWellFormed());
    assert.equal(presentation.compressedDisplay, true);
    store.getState().setTextValue('new query');
    assert.equal(store.getState().input.context.clipboard, null);
});

test('matching edges preserve original Unicode pairs and never use the display summary', () => {
    const {getTextEdges} = loadTS('frontend/src/lib/input-text.ts');
    const {createTextInput, createAutomaticClipboardInput, getPluginMatchContext} = loadTS('frontend/src/lib/launcher-input.ts');
    const origin = 'a'.repeat(127) + '😀' + 'middle '.repeat(10000) + '😀' + 'z'.repeat(127);
    const edges = getTextEdges(origin);
    assert.equal(edges.head, 'a'.repeat(127), 'a pair crossing the head boundary stays outside the snippet');
    assert.equal(edges.tail, 'z'.repeat(127), 'a pair crossing the tail boundary stays outside the snippet');
    assert.equal(edges.head, origin.slice(0, edges.head.length));
    assert.equal(edges.tail, origin.slice(-edges.tail.length));
    for (const edge of [edges.head, edges.tail]) {
        assert.ok(edge.length <= 128);
        assert.ok(edge.isWellFormed());
    }
    const short = ' \0😀\n ';
    assert.equal(getTextEdges(short).head, short);
    assert.equal(getTextEdges(short).tail, short);
    const inlinePreview = getPluginMatchContext(createTextInput(origin, 'clipboard'));
    const snapshotPreview = getPluginMatchContext(createAutomaticClipboardInput({
        snapshotId: 'unicode-source', version: '1', text: '', length: origin.length,
        prefix: edges.head, suffix: edges.tail, preview: 'different display ...... summary',
    }));
    assert.equal(snapshotPreview.input.head, inlinePreview.input.head);
    assert.equal(snapshotPreview.input.tail, inlinePreview.input.tail);
    assert.equal(snapshotPreview.input.length, origin.length);
    assert.equal(snapshotPreview.input.head.includes('different display'), false);
});

test('official text, translation and QR pages keep clipboard payloads literal', () => {
    for (const plugin of ['textstats', 'translate', 'qr']) {
        const html = fs.readFileSync(path.join(root, `plugins/official/watools.plugin.${plugin}/plugin/index.html`), 'utf8');
        const start = html.indexOf('    const extractSeedText =');
        const end = html.indexOf('\n    };', start) + 7;
        const context = {exports: {}, stripTriggerPrefix() {throw Error('raw clipboard treated as a command')}};
        vm.runInNewContext(html.slice(start, end) + '\nexports.extract = extractSeedText;', context);
        for (const text of ['  count words\n  ', '  fy hello\n  ', ' qr data\n', ' \n\t', '\0😀']) {
            assert.equal(context.exports.extract({input: {valueType: 'clipboard', value: text}, clipboard: {text}}), text);
        }
    }
});

test('JSON editor imports raw clipboard objects, arrays and scalars without requiring a command prefix', () => {
    const html = fs.readFileSync(path.join(root, 'plugins/official/watools.plugin.json/plugin/index.html'), 'utf8');
    const extract = name => {
        const start = html.indexOf(`    const ${name} =`);
        assert.ok(start >= 0, `missing JSON runtime function ${name}`);
        return html.slice(start, html.indexOf('\n    };', start) + 7);
    };
    let editorText = '';
    const sandbox = {
        exports: {}, URLSearchParams, window: {location: {search: '?seed=json%20wrong'}}, state: {editor: {}},
        normalizeLineEndings: value => value.replace(/\r\n?/g, '\n'),
        replaceEditorValue: value => {editorText = value}, validateNow() {}, focusEditor() {}, setStatus() {},
        triggerPrefixPattern: /^(?:json|url2params|query2params|qs2params)\s/i,
        parseJsonText: value => ({ok: true, value: JSON.parse(value)}), convertUrlParamsInput: () => null,
    };
    vm.runInNewContext(['parseCommandInput', 'extractCommandSeed', 'shouldAutoFormatJsonText', 'applyHostContext'].map(extract).join('\n') + '\nexports.apply = applyHostContext;', sandbox);
    const largeJson = ' \n' + JSON.stringify({items: Array.from({length: 10000}, (_, index) => ({index, text: '😀'}))}) + '\n ';
    const rawContext = text => ({input: {valueType: 'clipboard', value: text}, clipboard: {contentType: 'text', text}});
    for (const text of [largeJson, '  [1,2,3]\n ', ' true ', '123', 'null', '"json literal"', 'json {"literal":true}', '\0😀']) {
        sandbox.exports.apply(rawContext(text));
        assert.equal(editorText, text);
    }
    // Clipboard has priority over compatibility URL seed; typed commands retain
    // their original prefix parsing and automatic formatting behavior.
    sandbox.window.location.search = '';
    sandbox.exports.apply({input: {valueType: 'text', value: 'json {"hello":"world"}'}, clipboard: null});
    assert.equal(editorText, JSON.stringify({hello: 'world'}, null, 2));
    sandbox.exports.apply({input: {valueType: 'text', value: '{"raw":true}'}, clipboard: null});
    assert.equal(editorText, '{"raw":true}');
});

test('deferred clipboard is a distinct source and rejects stale reads or repeated versions', () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    const snapshot = {snapshotId: 'a'.repeat(32), version: '1', text: '', preview: 'head ...... tail', prefix: 'head', suffix: 'tail', length: 10000000};
    assert.equal(store.getState().setAutomaticClipboard(snapshot, 0), true);
    const input = store.getState().input;
    assert.equal(input.kind, 'clipboard-snapshot');
    assert.equal(input.snapshot.id, snapshot.snapshotId);
    assert.equal(input.snapshot.length, snapshot.length);
    assert.equal('context' in input, false, 'an unread source must not impersonate a complete PluginContext');
    assert.equal('value' in input.snapshot, false);
    assert.equal('text' in input.snapshot, false);
    assert.equal(store.getState().isPanelOpen(), true);
    store.getState().clearValue();
    assert.equal(store.getState().setAutomaticClipboard(snapshot, 0), false);
    assert.equal(store.getState().setAutomaticClipboard(snapshot, store.getState().inputRevision), false);
    assert.equal(store.getState().setAutomaticClipboard({...snapshot, version: '2'}, store.getState().inputRevision), true);
    store.getState().setTextValue('typed');
    assert.equal(store.getState().input.kind, 'inline');
    assert.equal(store.getState().input.context.input.value, 'typed');
});

test('large text cannot reach generic search, ranking or legacy matchers', () => {
    const {getSearchQuery} = loadTS('frontend/src/lib/input-text.ts');
    const {createRankingInputContext} = loadTS('frontend/src/lib/command-ranking.ts');
    const {createTextInput, createAutomaticClipboardInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const {filterPluginEntries} = loadTS('frontend/src/lib/plugin-matching.ts');
    const value = 'private body '.repeat(900000);
    assert.equal(getSearchQuery(value), '');
    assert.equal(getSearchQuery(' '.repeat(10000000)), '');
    assert.equal(getSearchQuery(' Visual Studio '), 'Visual Studio');
    const ranking = createRankingInputContext({value, valueType: 'clipboard', clipboardContentType: 'text'});
    assert.equal(ranking.normalizedValue, '');
    assert.equal(ranking.key, 'clipboard:text:');
    const legacy = () => {throw Error('full-text legacy matcher executed')};
    const ui = {type: 'ui', matcher: {kind: 'legacy', match: legacy}};
    const executable = {type: 'executable', matcher: {kind: 'legacy', match: legacy, largeTextPrefixes: ['https://']}};
    const longInline = createTextInput(value);
    assert.equal(filterPluginEntries([ui, executable], longInline)[0], ui, 'legacy UI remains an explicit choice for long text');
    assert.equal(filterPluginEntries([ui, executable], longInline).includes(executable), false);
    const snapshot = createAutomaticClipboardInput({snapshotId: 'fixed-source', length: value.length, prefix: 'plain text', suffix: 'body', preview: 'summary', text: '', version: '1'});
    assert.equal(filterPluginEntries([ui, executable], snapshot).includes(executable), false);
    const urlSnapshot = createAutomaticClipboardInput({snapshotId: 'url-source', length: value.length, prefix: ' HTTPS://example.com', suffix: '/end', preview: 'summary', text: '', version: '2'});
    assert.equal(filterPluginEntries([executable], urlSnapshot)[0], executable, 'declared legacy prefix is evaluated against only the preview');
    const short = {type: 'ui', matcher: {kind: 'legacy', match: c => c.input.value === 'exact'}};
    assert.equal(filterPluginEntries([short], createTextInput('exact'))[0], short);
});

test('automatic requests coalesce and only an explicit launch resolves the fixed snapshot', async () => {
    let finishRead, originReads = 0, automaticReads = 0;
    const origin = '  ' + '😀 body '.repeat(160000) + '\n';
    const host = {
        GetAutomaticClipboardApi: () => {automaticReads++; return new Promise(resolve => {finishRead = resolve})},
        GetClipboardSnapshotTextApi: async id => {assert.equal(id, 'selected-snapshot'); originReads++; return origin},
    };
    const api = loadTS('frontend/src/api/app.ts', {'@/api/host': {host}});
    const {createAutomaticClipboardInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const first = api.getAutomaticClipboard();
    const second = api.getAutomaticClipboard();
    assert.equal(first, second);
    assert.equal(automaticReads, 1);
    assert.equal(originReads, 0);
    const view = {snapshotId: 'selected-snapshot', length: origin.length, text: '', preview: 'summary', prefix: origin.slice(0,128), suffix: origin.slice(-128), version: '1'};
    finishRead(view);
    await first;
    const input = createAutomaticClipboardInput(view);
    assert.equal(originReads, 0);
    const result = await api.resolvePluginContext(input);
    assert.equal(result.input.value, origin);
    assert.equal(result.clipboard.text, origin);
    assert.equal(originReads, 1);
    assert.equal(input.kind, 'clipboard-snapshot', 'resolving does not mutate the captured source');
    assert.equal('context' in input, false);
});

test('inline execution clones context wrappers and files without invoking clipboard APIs', async () => {
    const {createClipboardInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const {resolvePluginContext} = loadTS('frontend/src/api/app.ts', {'@/api/host': {host: {
        GetClipboardSnapshotTextApi() {assert.fail('inline sources need no origin read')},
        GetClipboardContentApi() {assert.fail('inline sources need no live clipboard read')},
    }}});
    const sourceFiles = ['original'];
    const input = createClipboardInput({contentType: 'files', files: sourceFiles});
    sourceFiles.push('outside mutation');
    const execution = await resolvePluginContext(input);
    execution.input.value = 'plugin mutation';
    execution.clipboard.files.push('plugin mutation');
    assert.equal(input.context.input.value, '');
    assert.equal(JSON.stringify(input.context.clipboard.files), '["original"]');
    assert.equal(JSON.stringify(execution.clipboard.files), '["original","plugin mutation"]');
});

test('preview matching receives immutable bounded metadata and execution alone receives the exact origin', async () => {
    const {filterPluginEntries} = loadTS('frontend/src/lib/plugin-matching.ts');
    const {createClipboardInput, createAutomaticClipboardInput, getPluginMatchContext} = loadTS('frontend/src/lib/launcher-input.ts');
    const origin = ' \n{"body":"' + 'private 😀 body '.repeat(100000) + '"}\n ';
    const input = createClipboardInput({contentType: 'text', text: origin, imageBase64: 'must-not-leak', files: ['/must-not-leak']});
    const preview = getPluginMatchContext(input);
    let received;
    const entry = {type: 'ui', matcher: {kind: 'preview', match: value => {
        received = value;
        return value.input.head.trimStart().startsWith('{') && value.input.tail.trimEnd().endsWith('}');
    }}};
    assert.equal(filterPluginEntries([entry], input)[0], entry);
    assert.equal(received.input.length, origin.length);
    assert.equal(received.input.head.length, 128);
    assert.equal(received.input.tail.length, 128);
    assert.equal(received.input.truncated, true);
    assert.ok(JSON.stringify(received).length < 800);
    assert.deepEqual(Object.keys(received).sort(), ['clipboard', 'input']);
    for (const key of ['value', 'originValue', 'snapshotId']) assert.equal(key in received.input, false);
    for (const key of ['text', 'files', 'imageBase64']) assert.equal(key in received.clipboard, false);
    assert.ok(Object.isFrozen(received) && Object.isFrozen(received.input) && Object.isFrozen(received.clipboard));
    const view = {snapshotId: 'fixed-source', length: origin.length, prefix: origin.slice(0,128), suffix: origin.slice(-128), preview: 'summary', text: '', version: '1'};
    const deferred = createAutomaticClipboardInput(view);
    const bridgePreview = getPluginMatchContext(deferred);
    assert.equal(bridgePreview.input.tail, preview.input.tail);
    let originReads = 0;
    const {resolvePluginContext} = loadTS('frontend/src/api/app.ts', {'@/api/host': {host: {GetClipboardSnapshotTextApi: async id => {assert.equal(id, 'fixed-source'); originReads++; return origin}}}});
    assert.equal(originReads, 0);
    const execution = await resolvePluginContext(deferred);
    assert.equal(originReads, 1);
    assert.equal(execution.input.value, origin);
    assert.equal(execution.clipboard.text, origin);
    const falseEntry = {type: 'ui', matcher: {kind: 'preview', match: () => false}, match() {throw Error('fallback ran')}};
    assert.equal(filterPluginEntries([falseEntry], input).length, 0);
});

test('unavailable snapshot resolution fails without reading current clipboard or creating a fake context', async () => {
    const expected = new Error('fixed snapshot unavailable');
    let reads = 0;
    const {resolvePluginContext} = loadTS('frontend/src/api/app.ts', {'@/api/host': {host: {
        GetClipboardSnapshotTextApi: async id => {assert.equal(id, 'expired'); reads++; throw expected},
        GetClipboardContentApi() {assert.fail('expired origins must never fall back to current clipboard')},
    }}});
    const input = {kind: 'clipboard-snapshot', snapshot: {id: 'expired', length: 1000000, head: '{', tail: '}', preview: 'summary'}};
    await assert.rejects(resolvePluginContext(input), error => error === expected);
    assert.equal(reads, 1);
    assert.equal('context' in input, false);
});

test('legacy matcher mutation cannot change source or the next entry context', () => {
    const {filterPluginEntries} = loadTS('frontend/src/lib/plugin-matching.ts');
    const {createClipboardInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const files = ['original'];
    const input = createClipboardInput({contentType: 'files', files});
    const mutator = {type: 'ui', matcher: {kind: 'legacy', match: context => {
        context.input.value = 'changed';
        context.clipboard.files.push('changed');
        return true;
    }}};
    const observer = {type: 'ui', matcher: {kind: 'legacy', match: context => {
        assert.equal(context.input.value, '');
        assert.equal(JSON.stringify(context.clipboard.files), '["original"]');
        return true;
    }}};
    assert.equal(filterPluginEntries([mutator, observer], input).length, 2);
    assert.equal(input.context.input.value, '');
    assert.equal(JSON.stringify(input.context.clipboard.files), '["original"]');
    assert.deepEqual(files, ['original']);
});

test('one frozen preview is shared across a matching pass and matcher errors do not expose origins', () => {
    const diagnostics = [];
    const {filterPluginEntries} = loadTS('frontend/src/lib/plugin-matching.ts', {}, {console: {error: (...args) => diagnostics.push(args)}});
    const {createTextInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const origin = 'private body '.repeat(100000);
    let firstPreview;
    const broken = {type: 'ui', subTitle: 'Broken', matcher: {kind: 'preview', match: preview => {
        firstPreview = preview;
        throw new Error(origin);
    }}};
    const survivor = {type: 'ui', subTitle: 'Survivor', matcher: {kind: 'preview', match: preview => {
        assert.equal(preview, firstPreview, 'one projection is sufficient for an entire pass');
        assert.throws(() => {'use strict'; preview.input.head = 'changed'}, TypeError);
        return true;
    }}};
    const matches = filterPluginEntries([broken, survivor], createTextInput(origin));
    assert.equal(matches.length, 1);
    assert.equal(matches[0], survivor);
    assert.equal(diagnostics.length, 1);
    assert.ok(JSON.stringify(diagnostics).length < 200);
    assert.equal(JSON.stringify(diagnostics).includes(origin), false);
});

test('official lightweight matchers recommend JSON, URL and assets without parsing or evaluating bodies', async () => {
    const {filterPluginEntries} = loadTS('frontend/src/lib/plugin-matching.ts');
    const {createClipboardInput} = loadTS('frontend/src/lib/launcher-input.ts');
    const {sanitizePluginEntries} = loadTS('frontend/src/lib/plugin.ts', {}, {fetch: async () => ({ok: true}), AbortController});
    const loadEntry = async plugin => {
        const definitions = loadTS(`plugins/official/watools.plugin.${plugin}/plugin/app.js`, {
            './calculator-storage.js': {appendHistory() {}},
            './calculator-core.js': {
                ...loadTS('plugins/official/watools.plugin.calculator/plugin/calculator-core.js'),
                calculateExpression() {throw Error('matching evaluated expression')},
                canEvaluateExpression() {throw Error('matching evaluated expression')},
            },
        }, {JSON: {...JSON, parse() {throw Error('matching parsed JSON')}}}).default;
        for (const entry of definitions) {
            assert.equal(typeof entry.matchPreview, 'function');
            assert.equal('match' in entry, false, 'official plugins must expose one matching contract');
        }
        return sanitizePluginEntries({homeUrl: 'https://example.invalid', packageId: `watools.plugin.${plugin}`}, definitions);
    };
    const [json] = await loadEntry('json');
    const accepts = (entry, value, type = 'text') => {
        const input = createClipboardInput({contentType: type, text: value, imageBase64: type === 'image' ? 'image' : null, files: type === 'files' ? ['file'] : null});
        return filterPluginEntries([entry], input).length === 1;
    };
    for (const text of [' {"a":1}\n', '[1,2]', 'json {"a":1}', '{' + 'invalid body '.repeat(200000) + '}', '[' + '1,'.repeat(200000) + ']']) assert.equal(accepts(json, text), true);
    for (const text of ['plain text', '{wrong end]', '[wrong end}', 'https://example.invalid']) assert.equal(accepts(json, text), false);
    const common = await loadEntry('common');
    assert.equal(accepts(common[0], 'https://example.invalid/' + 'a'.repeat(1000000)), true);
    assert.equal(accepts(common[1], '', 'files'), true);
    assert.equal(accepts(common[2], '', 'image'), true);
    const calculator = await loadEntry('calculator');
    assert.equal(accepts(calculator[0], '1 + 2 * 3'), true);
    assert.equal(accepts(calculator[0], 'not math'), false);
    for (const plugin of ['translate', 'qr', 'textstats']) await loadEntry(plugin);
});

test('calculator candidates reject invalid expressions without copying or recording results', async () => {
    const copies = [], history = [];
    const core = loadTS('plugins/official/watools.plugin.calculator/plugin/calculator-core.js');
    const [calculator] = loadTS('plugins/official/watools.plugin.calculator/plugin/app.js', {
        './calculator-core.js': core,
        './calculator-storage.js': {appendHistory: async (...args) => history.push(args)},
    }, {window: {runtime: {ClipboardSetText: async text => copies.push(text)}}}).default;
    const {createTextInput, getPluginMatchContext} = loadTS('frontend/src/lib/launcher-input.ts');
    for (const expression of ['2+', '1/0']) {
        const input = createTextInput(expression);
        assert.equal(calculator.matchPreview(getPluginMatchContext(input)), true, 'lightweight matching is only a candidate check');
        await assert.rejects(calculator.execute(input.context), error => error instanceof Error || Object.prototype.toString.call(error) === '[object Error]');
        assert.equal(input.context.input.value, expression);
    }
    assert.deepEqual(copies, [], 'invalid results must not replace the clipboard');
    assert.deepEqual(history, [], 'invalid results must not be saved as successful calculations');
});

test('a calculator execution failure keeps launcher input and reports an error without hiding', async () => {
    const sideEffects = [];
    const core = loadTS('plugins/official/watools.plugin.calculator/plugin/calculator-core.js');
    const [calculator] = loadTS('plugins/official/watools.plugin.calculator/plugin/app.js', {
        './calculator-core.js': core,
        './calculator-storage.js': {appendHistory: async () => sideEffects.push('history')},
    }, {window: {runtime: {ClipboardSetText: async () => sideEffects.push('copy')}}}).default;
    for (const expression of ['2+', '1/0']) {
        const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
        store.getState().setTextValue(expression);
        const source = store.getState().input;
        const errors = [], events = [];
        const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
            launchPending: {current: false}, focusGeneration: {current: 0}, input: source,
            setLaunching() {}, setExecutionError: error => errors.push(error), navigate() {assert.fail('executable cannot navigate')},
            useAppStore: store, usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
            resolvePluginContext: async captured => captured.context,
            withPluginApi: async (_id, action) => action({}), updatePluginUsage: async () => {},
            clearValue: store.getState().clearValue,
            flushApplicationUsage: async () => events.push('flush'), flushPluginUsage: async () => events.push('flush'),
            host: {HideAppApi: () => events.push('hide')}, Logger: {error() {}},
        }, 'onTriggerPluginCommand');
        await handler({...calculator, packageId: 'watools.plugin.calculator'});
        assert.equal(store.getState().input, source, 'a failed action must retain the original source');
        assert.equal(store.getState().input.context.input.value, expression);
        assert.ok(errors.some(error => typeof error === 'string' && error.includes('插件启动或执行失败')));
        assert.deepEqual(events, [], 'failure cannot clear, flush success, or hide the launcher');
        assert.deepEqual(sideEffects, [], 'failure cannot copy or record history');
    }
});

test('entry sanitization normalizes preview and legacy definitions into one explicit matcher', async () => {
    const {sanitizePluginEntries} = loadTS('frontend/src/lib/plugin.ts', {}, {fetch: async () => ({ok: true}), AbortController});
    const plugin = {homeUrl: 'https://example.invalid', packageId: 'watools.plugin.test'};
    const matchPreview = () => false;
    const match = () => true;
    const execute = async () => {};
    const entries = await sanitizePluginEntries(plugin, [
        {type: 'ui', subTitle: 'Preview UI', icon: null, file: 'index.html', matchPreview, match},
        {type: 'executable', subTitle: 'Preview execute', icon: null, matchPreview, execute},
        {type: 'ui', subTitle: 'Legacy', icon: null, file: 'legacy.html', match},
        {type: 'ui', subTitle: 'Invalid', icon: null, file: 'invalid.html'},
        {type: 'executable', subTitle: 'Missing execute', icon: null, matchPreview},
    ]);
    assert.equal(entries.length, 3);
    assert.equal(entries[0].matcher.kind, 'preview');
    assert.equal(entries[0].matcher.match, matchPreview, 'preview is authoritative even when legacy is also defined');
    assert.equal(entries[1].matcher.kind, 'preview');
    assert.equal(entries[1].execute, execute);
    assert.equal(entries[2].matcher.kind, 'legacy');
    assert.equal(entries[2].matcher.match, match);
    for (const entry of entries) {
        assert.equal('match' in entry, false);
        assert.equal('matchPreview' in entry, false);
        assert.equal('largeTextPrefixes' in entry, false);
    }
});

test('snapshot lease follows input state across route changes and releases on clear', async () => {
    const calls = [];
    const host = {
        RetainClipboardSnapshotApi: async id => calls.push(['retain', id]),
        ReleaseClipboardSnapshotApi: async id => calls.push(['release', id]),
    };
    const {watchClipboardSnapshot} = loadTS('frontend/src/api/app.ts', {'@/api/host': {host}}, {
        setInterval: callback => callback, clearInterval() {},
    });
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    const stop = watchClipboardSnapshot(store, () => assert.fail('lease failed'));
    const view = {version: '1', snapshotId: 'snapshot-1', text: '', preview: 'summary', prefix: 'original head', suffix: 'original tail', length: 1000000};
    store.getState().setAutomaticClipboard(view, 0);
    await tick();
    assert.equal(JSON.stringify(calls), JSON.stringify([['retain', 'snapshot-1']]));
    // Route teardown is independent of the root state lease.
    store.setState({inputRevision: store.getState().inputRevision + 1});
    assert.equal(calls.filter(call => call[0] === 'release').length, 0);
    store.getState().clearValue();
    await tick();
    assert.equal(JSON.stringify(calls), JSON.stringify([['retain', 'snapshot-1'], ['release', 'snapshot-1']]));
    store.getState().setClipboardContent({contentType: 'text', text: 'inline clipboard'});
    await tick();
    assert.equal(calls.length, 2, 'inline sources never acquire a clipboard lease');
    stop();
});

test('a cancelled coalesced focus does not release the snapshot accepted by a later focus', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    let complete;
    const pending = new Promise(resolve => {complete = resolve});
    const released = [];
    const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
        focusGeneration: {current: 0}, useAppStore: store,
        getAutomaticClipboard: () => pending,
        setAutomaticClipboard: store.getState().setAutomaticClipboard,
        inputRef: {current: {select() {}, focus() {}}},
        host: {ReleaseClipboardSnapshotApi: id => released.push(id)},
        flushApplicationUsage() {}, flushPluginUsage() {}, Logger: {error() {}},
    }, 'handleWindowFocus');
    handler(true);
    handler(true);
    complete({version: '1', snapshotId: 'shared', text: '', preview: 'summary', prefix: 'original head', suffix: 'original tail', length: 1000000});
    await tick();
    assert.equal(store.getState().input.snapshot.id, 'shared');
    assert.equal(released.length, 0);
});

test('a delayed clipboard paste after blur stays visible when usage flush completes', async () => {
    for (const newPaste of [true, false]) {
        const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
        const events = [];
        let finishFlush;
        const flushed = new Promise(resolve => {finishFlush = resolve});
        const revision = store.getState().inputRevision;
        const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
            focusGeneration: {current: 0}, useAppStore: store,
            setAutomaticClipboard: store.getState().setAutomaticClipboard,
            flushApplicationUsage: () => flushed, flushPluginUsage: async () => {},
            isDevMode: () => false, host: {HideAppApi: () => events.push('hide')},
        }, 'handleWindowFocus');
        handler(false);
        if (newPaste) store.getState().setClipboardContent({contentType: 'text', text: 'late paste'}, revision);
        finishFlush();
        await tick();
        assert.deepEqual(events, newPaste ? [] : ['hide'], 'a new source prevents the old blur operation from hiding');
        if (newPaste) assert.equal(store.getState().input.context.input.value, 'late paste');
    }
});

test('mock clipboard snapshots deliver exact text, release independently and reject expired IDs', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {crypto: require('node:crypto').webcrypto});
    const text = '  \0' + 'word '.repeat(200000) + '\n';
    const host = createMockHost({clipboard: {contentType: 'text', text}});
    const view = await host.GetAutomaticClipboardApi();
    assert.equal(view.text, '');
    assert.ok(view.preview.length < 140);
    assert.equal(await host.GetClipboardSnapshotTextApi(view.snapshotId), text);
    await host.RetainClipboardSnapshotApi(view.snapshotId);
    await host.ReleaseClipboardSnapshotApi(view.snapshotId);
    await assert.rejects(host.GetClipboardSnapshotTextApi(view.snapshotId), error => error.code === 'UNAVAILABLE');
    const same = await host.GetAutomaticClipboardApi();
    assert.equal(same.version, view.version);
    assert.equal(await host.GetClipboardSnapshotTextApi(same.snapshotId), text);
});

test('mock clipboard writes preserve exact text, change versions and leave captured snapshots fixed', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors}, {crypto: require('node:crypto').webcrypto});
    const oldOrigin = '  \0😀 ' + 'old original '.repeat(1000) + '\n  ';
    const host = createMockHost({clipboard: {contentType: 'text', text: oldOrigin}});
    const captured = await host.GetAutomaticClipboardApi();
    const newOrigin = 'a'.repeat(127) + '😀' + ' \0 new original\n '.repeat(1000) + '😀' + 'z'.repeat(127);
    await host.SetClipboardTextApi(newOrigin);
    assert.equal((await host.GetClipboardContentApi()).text, newOrigin);
    const updated = await host.GetAutomaticClipboardApi();
    assert.notEqual(updated.version, captured.version);
    assert.notEqual(updated.snapshotId, captured.snapshotId);
    assert.equal(updated.length, newOrigin.length);
    assert.equal(updated.text, '', 'long clipboard text remains deferred after a write');
    assert.equal(updated.prefix, 'a'.repeat(127));
    assert.equal(updated.suffix, 'z'.repeat(127));
    for (const edge of [updated.prefix, updated.suffix]) {
        assert.ok(edge.length <= 128);
        assert.ok(edge.isWellFormed(), 'match metadata must not split the original surrogate pair');
    }
    assert.equal(await host.GetClipboardSnapshotTextApi(updated.snapshotId), newOrigin);
    assert.equal(await host.GetClipboardSnapshotTextApi(captured.snapshotId), oldOrigin, 'writing current clipboard cannot change a captured origin');
    const shortOrigin = ' \0😀\n ';
    await host.SetClipboardTextApi(shortOrigin);
    const shortView = await host.GetAutomaticClipboardApi();
    assert.notEqual(shortView.version, updated.version);
    assert.equal(shortView.text, shortOrigin);
    assert.equal(shortView.prefix, shortOrigin);
    assert.equal(shortView.suffix, shortOrigin);
    assert.equal(await host.GetClipboardSnapshotTextApi(captured.snapshotId), oldOrigin);
    await assert.rejects(host.SetClipboardTextApi(null), error => error.code === 'INVALID_ARGUMENT');
    assert.equal((await host.GetAutomaticClipboardApi()).version, shortView.version, 'a rejected write cannot change clipboard version');
});

function windowKeyHandler(file, globals, variable = 'handleHotkey') {
    const source = ts.createSourceFile(file, fs.readFileSync(path.join(root, file), 'utf8'),
        ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    let initializer;
    function visit(node) {
        if (ts.isVariableDeclaration(node) && node.name.getText(source) === variable) {
            initializer = node.initializer.getText(source);
        }
        ts.forEachChild(node, visit);
    }
    visit(source);
    assert.ok(initializer, `callback ${variable} missing in ${file}`);
    const context = {...globals, URLSearchParams, useCallback: callback => callback, exports: {}};
    const output = ts.transpileModule(`exports.handle = ${initializer};`, {
        compilerOptions: {target: ts.ScriptTarget.ES2022},
    }).outputText;
    vm.runInNewContext(output, context);
    return context.exports.handle;
}

test('a stalled usage write does not delay plugin navigation; clearing input cancels resolution', async () => {
    const events = [];
    let revision = 0, finishOrigin;
    const context = {input: {value: 'original'}, clipboard: null};
    const input = {kind: 'inline', context};
    const buildHandler = resolve => windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
        launchPending: {current: false}, focusGeneration: {current: 0}, input,
        setLaunching: value => events.push(['launching', value]), setExecutionError() {},
        useAppStore: {getState: () => ({input, inputRevision: revision})},
        usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
        resolvePluginContext: resolve,
        updatePluginUsage: () => {events.push(['usage']); return new Promise(() => {})},
        persistPluginLaunchContext: context => {assert.equal(context.input.value, 'original'); return 'launch-id'},
        navigate: route => events.push(['navigate', route]),
        clearValue() {}, flushApplicationUsage() {}, flushPluginUsage() {},
        Logger: {error() {}},
    }, 'onTriggerPluginCommand');
    await buildHandler(async captured => {assert.equal(captured, input); return context})({type: 'ui', packageId: 'probe', file: 'index.html'});
    assert.equal(events.filter(e => e[0] === 'navigate').length, 1);
    events.length = 0;
    const pending = buildHandler(() => new Promise(resolve => {finishOrigin = resolve}))({type: 'ui', packageId: 'probe'});
    revision++;
    finishOrigin(context);
    await pending;
    assert.equal(events.filter(e => e[0] === 'navigate' || e[0] === 'usage').length, 0);
});

test('a stale snapshot resolution failure cannot show an error on new input', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    store.getState().setAutomaticClipboard({
        snapshotId: 'old-source', version: '1', text: '', length: 1000000,
        prefix: 'original head', suffix: 'original tail', preview: 'summary',
    }, store.getState().inputRevision);
    let rejectOrigin;
    const errors = [];
    const diagnostics = [];
    const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
        launchPending: {current: false}, focusGeneration: {current: 0}, input: store.getState().input,
        setLaunching() {}, setExecutionError: error => errors.push(error), useAppStore: store,
        usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
        resolvePluginContext: () => new Promise((_resolve, reject) => {rejectOrigin = reject}),
        updatePluginUsage() {assert.fail('a failed resolution must not record usage')},
        navigate() {assert.fail('a failed resolution must not navigate')},
        clearValue: store.getState().clearValue, flushApplicationUsage() {}, flushPluginUsage() {},
        Logger: {error: message => diagnostics.push(message)},
    }, 'onTriggerPluginCommand');
    const pending = handler({type: 'ui', packageId: 'probe', file: 'index.html'});
    store.getState().setTextValue('new input');
    rejectOrigin(new Error('old snapshot unavailable'));
    await pending;
    assert.equal(store.getState().input.context.input.value, 'new input');
    assert.deepEqual(errors.filter(Boolean), [], 'an old request cannot attach its error to new input');
    assert.deepEqual(diagnostics, ['插件启动或执行失败']);
});

test('an executable completing after input changes neither clears nor hides the new input', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    store.getState().setTextValue('original');
    const input = store.getState().input;
    let finishExecution;
    const events = [];
    const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
        launchPending: {current: false}, focusGeneration: {current: 0}, input,
        setLaunching() {}, setExecutionError() {}, navigate() {assert.fail('executable cannot navigate')}, useAppStore: store,
        usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
        resolvePluginContext: async captured => captured.context,
        withPluginApi: async (_id, run) => run({}), updatePluginUsage: async () => {},
        clearValue: store.getState().clearValue,
        flushApplicationUsage: async () => {}, flushPluginUsage: async () => {},
        host: {HideAppApi: () => events.push('hide')}, Logger: {error() {}},
    }, 'onTriggerPluginCommand');
    const pending = handler({type: 'executable', packageId: 'probe', execute: () => new Promise(resolve => {finishExecution = resolve})});
    await tick();
    store.getState().setTextValue('new input');
    finishExecution();
    await pending;
    assert.equal(store.getState().input.context.input.value, 'new input');
    assert.deepEqual(events, []);
});

test('typing during usage flush keeps the new launcher input visible', async () => {
    const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
    store.getState().setTextValue('original');
    let finishFlush;
    const flushed = new Promise(resolve => {finishFlush = resolve});
    const events = [];
    const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
        launchPending: {current: false}, focusGeneration: {current: 0}, input: store.getState().input,
        setLaunching() {}, setExecutionError() {}, navigate() {assert.fail('executable cannot navigate')}, useAppStore: store,
        usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
        resolvePluginContext: async captured => captured.context,
        withPluginApi: async (_id, run) => run({}), updatePluginUsage: async () => {},
        clearValue: store.getState().clearValue,
        flushApplicationUsage: () => flushed, flushPluginUsage: async () => {},
        host: {HideAppApi: () => events.push('hide')}, Logger: {error() {}},
    }, 'onTriggerPluginCommand');
    const pending = handler({type: 'executable', packageId: 'probe', execute: async () => {}});
    await tick();
    assert.equal(store.getState().isPanelOpen(), false, 'successful action clears its own input before flushing');
    store.getState().setTextValue('new input during flush');
    finishFlush();
    await pending;
    assert.equal(store.getState().input.context.input.value, 'new input during flush');
    assert.deepEqual(events, []);
});

test('a queued executable rechecks cancellation before performing its action', async () => {
    for (const cancellation of ['new input', 'blur']) {
        const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
        store.getState().setTextValue('original');
        const generation = {current: 0};
        const events = [];
        let resumeQueue;
        const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
            launchPending: {current: false}, focusGeneration: generation, input: store.getState().input,
            setLaunching() {}, setExecutionError() {}, navigate() {assert.fail('executable cannot navigate')}, useAppStore: store,
            usePluginStore: {getState: () => ({getPluginById: () => ({enabled: true})})},
            resolvePluginContext: async captured => captured.context,
            withPluginApi: (_id, action) => new Promise(resolve => {resumeQueue = () => resolve(action({}))}),
            updatePluginUsage: async () => {}, clearValue: store.getState().clearValue,
            flushApplicationUsage: async () => {}, flushPluginUsage: async () => {},
            host: {HideAppApi: () => events.push('hide')}, Logger: {error() {}},
        }, 'onTriggerPluginCommand');
        const pending = handler({type: 'executable', packageId: 'probe', execute: async () => events.push('execute')});
        await tick();
        assert.equal(typeof resumeQueue, 'function', 'the action must have reached the compatibility queue');
        if (cancellation === 'new input') store.getState().setTextValue('new input');
        else generation.current++;
        resumeQueue();
        await pending;
        assert.deepEqual(events, [], `${cancellation}: a cancelled queued action must neither run nor hide`);
        assert.equal(store.getState().input.context.input.value, cancellation === 'new input' ? 'new input' : 'original');
    }
});

test('empty Escape does not hide after input or focus changes during usage flush', async () => {
    for (const cancellation of ['new input', 'blur', 'none']) {
        const {useAppStore: store} = loadTS('frontend/src/stores/appStore.ts');
        const generation = {current: 0};
        const events = [];
        let finishFlush;
        const flushed = new Promise(resolve => {finishFlush = resolve});
        const handler = windowKeyHandler('frontend/src/components/watools/wa-command.tsx', {
            useAppStore: store, focusGeneration: generation,
            getIsPanelOpen: store.getState().isPanelOpen, clearValue: store.getState().clearValue,
            flushApplicationUsage: () => flushed, flushPluginUsage: async () => {},
            host: {HideOrShowAppApi: () => events.push('hide')},
        });
        handler({key: 'Escape', isComposing: false, keyCode: 27, preventDefault() {}, stopPropagation() {}});
        if (cancellation === 'new input') store.getState().setTextValue('new input');
        else if (cancellation === 'blur') generation.current++;
        finishFlush();
        await tick();
        assert.deepEqual(events, cancellation === 'none' ? ['hide'] : [], `${cancellation}: only the original empty launcher may hide`);
        if (cancellation === 'new input') assert.equal(store.getState().input.context.input.value, 'new input');
    }
});

test('IME candidate keys leave launcher input and plugin/management routes intact', async () => {
    for (const name of ['wa-command', 'wa-plugin', 'wa-plugin-management', 'wa-log-management']) {
        const actions = [];
        const handler = windowKeyHandler(`frontend/src/components/watools/${name}.tsx`, {
            getIsPanelOpen: () => true, getCanClearAssets: () => true,
            clearValue: () => actions.push('clear'), navigate: () => actions.push('navigate'),
            detach: () => actions.push('detach'),
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

test('plugin refresh reports a failed fetch and a later retry loads the new revision', async () => {
    let reads = 0;
    const {usePluginStore: store} = loadTS('frontend/src/stores/pluginStore.ts', {
        '@/api/plugin': {
            getPlugins: async () => {
                reads++;
                if (reads === 2) throw new Error('plugin database temporarily unavailable');
                return [{packageId: 'probe', enabled: true, entry: [], version: reads === 1 ? 'old' : 'new'}];
            },
            disposePlugin: async () => {}, updatePluginUsage: async () => {},
        },
        '@/lib/logger': {Logger: {error() {}}},
    }, {Error});
    await tick();
    await assert.rejects(store.getState().refreshPlugins(), /temporarily unavailable/);
    assert.equal(store.getState().plugins[0].version, 'old');
    assert.equal(store.getState().isLoading, false);
    await store.getState().refreshPlugins();
    assert.equal(store.getState().plugins[0].version, 'new');
    assert.equal(store.getState().error, null);
});

test('management retries a completed installation refresh and clears only its refresh error', async () => {
    let effect, retry, attempts = 0, releaseRefresh;
    const states = [];
    const status = {state: 'succeeded', jobId: 1, configuration: {sourceRoot: '/checkout', goExecutable: '/go', toolPath: '/bin'}, officialPlugins: [], output: []};
    const refreshPlugins = () => {
        attempts++;
        if (attempts === 1) return Promise.reject(new Error('usage temporarily unavailable'));
        return new Promise(resolve => {releaseRefresh = resolve});
    };
    const {WaManagement} = loadTS('frontend/src/components/watools/wa-management.tsx', {
        react: {
            useState: value => {
                const index = states.push(value) - 1;
                return [value, next => {states[index] = next}];
            },
            useRef: value => ({current: value}), useEffect: callback => {effect = callback},
        },
        'react/jsx-runtime': {jsx: () => null, jsxs: () => null},
        wouter: {useLocation: () => ['/', () => {}]}, 'lucide-react': {},
        '@/api/host': {host: {GetLocalInstallationApi: async () => status}},
        '@/components/ui/button': {}, '@/components/ui/input': {}, './wa-plugin-management': {}, './wa-workspace-header': {},
        '@/stores/pluginStore': {usePluginStore: selector => selector({refreshPlugins})},
    }, {setTimeout: callback => {retry = callback; return 1}, clearTimeout() {}});
    WaManagement({});
    const cleanup = effect();
    await tick();
    assert.equal(attempts, 1);
    assert.match(states[6], /usage temporarily unavailable/);
    // A separate action error must survive recovery of the plugin refresh.
    states[5] = 'independent source configuration error';
    const recovering = retry();
    await tick();
    assert.equal(attempts, 2, 'the same completed job retries its failed refresh');
    releaseRefresh();
    await recovering;
    assert.equal(states[6], '');
    assert.equal(states[5], 'independent source configuration error');
    await retry();
    assert.equal(attempts, 2, 'a successfully refreshed job is not refreshed again');
    cleanup();
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
    const {persistPluginLaunchContext,readPluginLaunchContext,releasePluginLaunchContext}=loadTS('frontend/src/lib/plugin-context.ts');
    const initial={input:{value:'private input',valueType:'clipboard',clipboardContentType:'files'},clipboard:{contentType:'files',text:null,imageBase64:null,files:['first']}};
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

test('stale clipboard reads and conditional clears cannot replace or clear newly typed input', () => {
    const {useAppStore:store}=loadTS('frontend/src/stores/appStore.ts');
    const initialRevision=store.getState().inputRevision;
    store.getState().clearValue();
    const automatic = {snapshotId: 'fixed', version: '1', text: 'late focus', prefix: 'late focus', suffix: 'late focus', length: 10, preview: 'late focus'};
    assert.equal(store.getState().setAutomaticClipboard(automatic, initialRevision), false);
    store.getState().setClipboardContent({contentType:'text',text:'late paste'},initialRevision);
    assert.equal(store.getState().input.context.input.value,'');
    const revision=store.getState().inputRevision;
    store.getState().setTextValue('new input');
    store.getState().setClipboardContent({contentType:'image',imageBase64:'stale image'},revision);
    assert.equal(store.getState().input.context.input.value,'new input');
    assert.equal(store.getState().input.context.clipboard,null);
    assert.equal(store.getState().clearValue(revision), false);
    assert.equal(store.getState().input.context.input.value,'new input');
    const current=store.getState().inputRevision;
    store.getState().setClipboardContent({contentType:'text',text:'fresh paste'},current);
    assert.equal(store.getState().input.context.input.value,'fresh paste');
    assert.equal(store.getState().input.context.input.valueType,'clipboard');
    assert.equal(store.getState().clearValue(store.getState().inputRevision), true);
    assert.equal(store.getState().isPanelOpen(), false);
});

test('v2 plugin runtime preserves clipboard booleans, window sizes and variadic event payloads', async () => {
    const callbacks = new Map();
    let failClipboard = false;
    const clipboardWrites = [];
    let target;
    const cancelled = [];
    const register = (name, callback) => {callbacks.set(name, callback); return () => {callbacks.delete(name); cancelled.push(name)}};
    const {legacyRuntime: runtime} = loadTS('frontend/src/lib/legacy-runtime.ts', {
        '@wailsio/runtime': {
            Application: {}, Browser: {}, Screens: {}, System: {},
            Clipboard: {Text: async () => 'probe', SetText: async () => {assert.fail('legacy clipboard writes must use the exact-text Host API')}},
            Window: {Get: name => {target = name; return {Size: async () => ({width: 920, height: 64})}}},
            Events: {On: register, Once: register, OnMultiple: register,
                Emit: async (name, data) => callbacks.get(name)?.({name, data}),
                Off: (...names) => names.forEach(name => callbacks.delete(name)), OffAll: () => callbacks.clear()},
        },
        '@/api/host': {host: {
            LogFrontendApi: async () => {}, GetLauncherSizeApi: async () => ({width:920, height:64}),
            SetClipboardTextApi: async text => {clipboardWrites.push(text); if (failClipboard) throw Error('clipboard unavailable')},
        }},
    });
    assert.equal(target, 'launcher');
    const origin = '  \0😀 original text\n  ';
    assert.equal(await runtime.ClipboardSetText(origin), true);
    assert.deepEqual(clipboardWrites, [origin], 'whitespace, NUL and Unicode must reach the Host API unchanged');
    failClipboard = true;
    await assert.rejects(runtime.ClipboardSetText(origin), /clipboard unavailable/);
    assert.deepEqual(clipboardWrites, [origin, origin]);
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

test('plugin window hooks preserve edited models and generic forms exclude passwords and files', async () => {
    const state = loadTS('frontend/src/lib/plugin-window-state.ts');
    const edited = {text:'  \0😀 edited JSON\n', view:{line:7}};
    let restored;
    const hooked = {watoolsSession:{capture:async()=>edited,restore:async value=>{restored=value}}};
    const captured = await state.capturePluginWindowState(hooked);
    assert.equal(captured.kind,'hooks');
    await state.restorePluginWindowState(hooked,captured);
    assert.equal(restored,edited);
    await assert.rejects(state.restorePluginWindowState({},captured),/状态恢复接口/);
    const events=[];
    const fields=[{tagName:'TEXTAREA',type:'textarea',value:'edited',dispatchEvent:e=>events.push(e.name)},
        {tagName:'INPUT',type:'password',value:'secret'}, {tagName:'INPUT',type:'file',value:'path'}];
    const frame={document:{querySelector:()=>null,querySelectorAll:()=>fields,createEvent:()=>({initEvent(name){this.name=name}})}};
    const form=await state.capturePluginWindowState(frame);
    assert.equal(form.value.length,1);
    fields[0].value='changed';
    await state.restorePluginWindowState(frame,form);
    assert.equal(fields[0].value,'edited');
    assert.deepEqual(events,['input','change']);
    await assert.rejects(state.capturePluginWindowState({document:{querySelector:()=>({})}}),/安装新版插件.*重新打开/);
});

test('plugin runtime targets its window and owns independent event subscriptions', async () => {
    const callbacks=new Map(), actions=[];
    const register=(name, callback)=>{callbacks.set(name,callback);return ()=>callbacks.delete(name)};
    const {createPluginRuntime}=loadTS('frontend/src/lib/legacy-runtime.ts',{
        '@wailsio/runtime':{Application:{Quit:()=>actions.push('quit-app')},Browser:{},Clipboard:{},Screens:{},System:{},
            Window:{Get:name=>({Hide:()=>actions.push(['hide',name]),Close:()=>actions.push(['close',name]),Size:async()=>({width:800,height:600})})},
            Events:{On:register,Once:register,OnMultiple:register,Emit:async(name,data)=>callbacks.get(name)?.({data})}},
        '@/api/host':{host:{GetLauncherSizeApi:async()=>({width:1280,height:64})}},
    });
    const a=createPluginRuntime('plugin-a','a'),b=createPluginRuntime('plugin-b','b');
    const observations=[];
    a.EventsOn('ping',value=>observations.push(['a',value]));b.EventsOn('ping',value=>observations.push(['b',value]));
    await a.EventsEmit('ping','first');
    assert.deepEqual(observations,[['a','first']]);
    a.EventsOffAll(); await b.EventsEmit('ping','second');
    assert.deepEqual(observations,[['a','first'],['b','second']]);
    a.Hide();b.Quit();
    assert.deepEqual(actions,[['hide','plugin-a'],['close','plugin-b']]);
    assert.equal(JSON.stringify(await b.WindowGetSize()),'{"w":800,"h":600}');
    b.dispose();assert.equal(callbacks.size,0);
});

test('iframe HTTP request IDs and cancellation remain owned by one UI session', async () => {
    const requests=[],cancelled=[],listeners=new Map();
    const api={HttpProxy:request=>new Promise(resolve=>requests.push({request,resolve})),CancelHttpProxy:async id=>cancelled.push(id)};
    const {attachPluginSession}=loadTS('frontend/src/lib/plugin-ui-session.ts',{
        '@/api/api':{createWaToolsApi:()=>api}, '@/lib/legacy-runtime':{createPluginRuntime:()=>({dispose(){}})},
    },{crypto:{randomUUID:()=> 'generated'},CustomEvent:class{constructor(name,options){this.name=name;this.detail=options.detail}}});
    const frame=()=>({addEventListener:(name,callback)=>listeners.set(name,callback),removeEventListener:name=>listeners.delete(name),dispatchEvent(){}});
    const a=frame(),b=frame();const context={input:{value:'source',valueType:'text'},clipboard:null};
    const closeA=attachPluginSession(a,'same-plugin','plugin-a','a',context);
    const closeB=attachPluginSession(b,'same-plugin','plugin-b','b',context);
    assert.equal(a.watools.session.windowMode,'detached');
    assert.equal(a.watools.session.id,'a');
    assert.equal(Object.isFrozen(a.watools.session),true);
    const taskA=a.watools.HttpProxy({url:'https://example.invalid',requestId:'same'});
    const taskB=b.watools.HttpProxy({url:'https://example.invalid',requestId:'same'});
    assert.equal(requests[0].request.requestId,'a:same');assert.equal(requests[1].request.requestId,'b:same');
    closeA();assert.deepEqual(cancelled,['a:same']);
    await assert.rejects(a.watools.HttpProxy({url:'https://example.invalid'}),/已关闭/);
    await b.watools.CancelHttpProxy('same');assert.deepEqual(cancelled,['a:same','b:same']);
    requests.forEach(item=>item.resolve({status_code:200}));await Promise.all([taskA,taskB]);closeB();
    assert.equal(context.input.value,'source');a.pluginContext.input.value='mutated';assert.equal(b.pluginContext.input.value,'source');
});

test('detach hides only after restoration; failures and cancelled pages keep the launcher available', async () => {
    for (const outcome of ['success','failure','cancelled']) {
        const failure=outcome==='failure',cancelled=outcome==='cancelled',success=outcome==='success';
        const events=[],body={inert:false};let finishStatus;
        const refs={alive:{current:true},pendingDetach:{current:null},detachBusy:{current:false}};
        const frame={document:{body}};
        const handler=windowKeyHandler('frontend/src/components/watools/wa-plugin.tsx',{
            ...refs,deliveredDocument:{current:frame.document},iframeRef:{current:{contentWindow:frame}},launch:{context:{input:{value:'original'}}},
            plugin:{homeUrl:'/revision'},safeFile:'index.html',packageId:'json',
            setDetaching:flag=>events.push(['busy',flag]),setDetachError:error=>{if(error)events.push(['error'])},
            capturePluginWindowState:async()=>({kind:'hooks',value:{text:'edited'}}),
            host:{DetachPluginApi:async request=>{assert.equal(request.state.value.text,'edited');return 'session'},
                GetPluginWindowStatusApi:()=>new Promise((resolve,reject)=>{finishStatus=()=>failure?reject(Error('failed')):resolve({ready:true})}),
                ClosePluginWindowApi:async id=>events.push(['close',id]),
                HideAppApi:async()=>events.push(['hide'])},navigate:()=>events.push(['navigate']),
            Date,setTimeout,
        },'detach');
        const task=handler();await tick();await handler();
        assert.equal(body.inert,true);assert.equal(events.some(e=>e[0]==='navigate' || e[0]==='hide'),false);
        if (cancelled) refs.alive.current=false;
        finishStatus();await task;await tick();
        assert.equal(body.inert,false);assert.equal(refs.detachBusy.current,false);
        assert.equal(events.some(e=>e[0]==='navigate'),success);
        assert.equal(events.some(e=>e[0]==='hide'),success);
        if (success) assert.ok(events.findIndex(e=>e[0]==='hide') < events.findIndex(e=>e[0]==='navigate'));
        assert.equal(events.some(e=>e[0]==='close'),!success);
        assert.equal(events.some(e=>e[0]==='error'),failure);
    }
});

test('Command+D captures before editor shortcuts and composing or repeated keys do not detach', () => {
    const actions=[];
    const handler=windowKeyHandler('frontend/src/components/watools/wa-plugin.tsx',{navigate(){},detach:()=>actions.push('detach')});
    const event={key:'d',metaKey:true,preventDefault:()=>actions.push('prevent'),stopImmediatePropagation:()=>actions.push('stop')};
    handler({...event,isComposing:true});assert.deepEqual(actions,[]);
    handler({...event,repeat:true});assert.deepEqual(actions,['prevent','stop']);
    actions.length=0;handler(event);assert.deepEqual(actions,['prevent','stop','detach']);
});

test('launcher resize restores its minimum height and ignores delayed plugin measurements', async () => {
    let finishOld;const writes=[];let reads=0;
    const {resizeWindowHeight}=loadTS('frontend/src/hooks/useElementResize.ts',{
        react:{},'@/lib/legacy-runtime':{WindowGetSize:()=>++reads===1?new Promise(resolve=>{finishOld=resolve}):Promise.resolve({w:1280,h:760}),
            WindowSetSize:async(w,h)=>writes.push({w,h})},
    });
    const old=resizeWindowHeight([{target:{clientHeight:760}}]);
    await resizeWindowHeight([{target:{clientHeight:54}}]);
    finishOld({w:1280,h:64});await old;
    assert.deepEqual(writes,[{w:1280,h:64}]);
});

test('browser preview without native window geometry does not request an invalid resize', async () => {
    const writes=[];
    const {resizeWindowHeight}=loadTS('frontend/src/hooks/useElementResize.ts',{
        react:{},'@/lib/legacy-runtime':{WindowGetSize:async()=>({w:0,h:0}),
            WindowSetSize:async(w,h)=>writes.push({w,h})},
    });
    await resizeWindowHeight([{target:{clientHeight:760}}]);
    assert.deepEqual(writes,[]);
});

function officialWindowHooks(plugin, globals) {
    const html = fs.readFileSync(path.join(root, `plugins/official/watools.plugin.${plugin}/plugin/index.html`), 'utf8');
    const start = html.indexOf('    window.watoolsSession = {');
    assert.ok(start >= 0, `${plugin} must provide explicit window state hooks`);
    const end = html.indexOf('\n    };', start) + 7;
    const context = {window: {}, Promise, ...globals};
    vm.runInNewContext(html.slice(start, end), context);
    return context.window.watoolsSession;
}

const windowField = (value) => ({value, textContent:'', className:'', selectionStart:1, selectionEnd:3,
    selectionDirection:'backward', scrollTop:17, focus(){}, setSelectionRange(start,end,direction){
        this.selectionStart=start;this.selectionEnd=end;this.selectionDirection=direction;
    }});

test('official JSON and textstats export edited text, selections and panel state', async () => {
    const editor=windowField('  \0😀 edited\n'),path=windowField('$.edited'),output=windowField('result');
    const model={previewCollapsed:false,editor:{saveViewState:()=>({cursor:9}),restoreViewState:value=>{model.view=value}}};
    let preview;
    const json=officialWindowHooks('json',{editorReady:Promise.resolve(),state:model,pathInput:path,extractOutput:output,
        getEditorValue:()=>editor.value,getModel:()=>({setValue:value=>{editor.value=value}}),normalizeLineEndings:value=>value,
        togglePanel:value=>{preview=value},
        validateNow(){},updateActionState(){},focusEditor(){}});
    const captured=await json.capture();editor.value='new initialization';path.value='';output.value='';
    await json.restore(captured);
    assert.equal(editor.value,'  \0😀 edited\n');assert.equal(path.value,'$.edited');assert.equal(output.value,'result');
    assert.equal(preview,true);assert.equal(model.view.cursor,9);
    const label={textContent:'original label'};let renders=0;
    const stats=officialWindowHooks('textstats',{editor,seedLabel:label,render:()=>renders++});
    const saved=await stats.capture();editor.value='overwritten';editor.selectionStart=0;
    await stats.restore(saved);
    assert.equal(editor.value,saved.text);assert.equal(editor.selectionStart,1);assert.equal(editor.selectionDirection,'backward');
    assert.equal(editor.scrollTop,17);assert.equal(renders,1);
});

test('QR window snapshot waits for imports and restores the existing image without regeneration', async () => {
    let completeImport;const pendingImages=new Set([new Promise(resolve=>{completeImport=resolve})]);
    const state={text:'decoded text',imageDataUrl:'data:image/png;base64,synthetic',imageLabel:'imported image',
        imageInfo:{width:100,height:80},activeSource:'image'};
    const textInput=windowField(state.text),statusNode={textContent:'decoded',className:'status success'};
    let image;
    const qr=officialWindowHooks('qr',{pendingImages,state,textInput,statusNode,
        applyTextValue:(text,source)=>{textInput.value=text;state.activeSource=source},setPreviewImage:(...args)=>{image=args}});
    let captured=false;const task=qr.capture().then(value=>{captured=true;return value});await tick();assert.equal(captured,false);
    completeImport();const saved=await task;textInput.value='fresh';state.activeSource='text';
    await qr.restore(saved);
    assert.equal(textInput.value,'decoded text');assert.equal(state.activeSource,'image');
    assert.equal(image[0],saved.imageDataUrl);assert.equal(image[1],'imported image');assert.equal(image[2].width,100);
});

test('translation window snapshot preserves results, cancels work and excludes credentials', async () => {
    const source=windowField('edited source'),target=windowField('existing translation');
    const status={textContent:'translating',className:'status'},detected={textContent:'检测结果 EN'};
    const calls=[];
    const globals={initialize:async()=>calls.push('init'),autoTranslateTimer:12,isTranslating:true,apiKey:'synthetic-secret',
        clearTimeout:()=>calls.push('stop-timer'),invalidateTranslation:()=>calls.push('cancel'),
        sourceTextNode:source,targetTextNode:target,statusNode:status,detectedLabel:detected,
        settingsPanel:{classList:{contains:()=>false}},setStatus:message=>{status.textContent=message},
        updateDirectionUI:()=>calls.push('direction'),setSettingsVisible:()=>calls.push('settings')};
    const hooks=officialWindowHooks('translate',globals);const saved=await hooks.capture();
    assert.equal(JSON.stringify(saved).includes('synthetic-secret'),false);assert.ok(saved.status.includes('暂停'));
    source.value='seed';target.value='';await hooks.restore(saved);
    assert.equal(source.value,'edited source');assert.equal(target.value,'existing translation');
    assert.ok(calls.includes('cancel'));assert.equal(detected.textContent,'检测结果 EN');
});

test('calculator window restore waits for initialization and does not write shared history or draft', async () => {
    let ready;const expressionInput=windowField('2 + 3');let rendered=0;
    const globals={ready:new Promise(resolve=>{ready=resolve}),expressionInput,
        history:[{expression:'1+1',result:'2'}],latestOutcome:{displayValue:'5'},statusNode:{textContent:'ready',className:'status'},
        renderHistory:()=>rendered++};
    const hooks=officialWindowHooks('calculator',globals);let done=false;
    const task=hooks.capture().then(value=>{done=true;return value});await tick();assert.equal(done,false);
    ready();const saved=await task;expressionInput.value='draft';await hooks.restore(saved);
    assert.equal(expressionInput.value,'2 + 3');assert.equal(rendered,1);assert.equal(expressionInput.selectionEnd,3);
});

test('translation initializes once and detached restore never repeats translation or overwrites edits', async () => {
    const html=fs.readFileSync(path.join(root,'plugins/official/watools.plugin.translate/plugin/index.html'),'utf8');
    const script=html.match(/<script type="module">([\s\S]*?)<\/script>/)[1];
    for (const windowMode of ['inline','detached']) {
        const fields=new Map(),listeners=new Map();let requests=0,keyReads=0;
        const field=id=>{
            if (!fields.has(id)) {
                const classes=new Set();
                fields.set(id,{...windowField(''),addEventListener(){},select(){},classList:{
                    contains:name=>classes.has(name),add:name=>classes.add(name),remove:name=>classes.delete(name),
                    toggle:(name,value)=>{if(value)classes.add(name);else classes.delete(name)},
                }});
            }
            return fields.get(id);
        };
        const window={location:{search:''},runtime:{},pluginContext:{input:{value:'original',valueType:'text'},clipboard:null},
            addEventListener:(name,fn)=>listeners.set(name,fn),localStorage:{removeItem(){},getItem:()=>null},
            watools:{session:{windowMode},SecretGet:async()=>{keyReads++;return 'synthetic-key'},StorageRemove:async()=>{},
                HttpProxy:async()=>{requests++;return {status_code:200,body:JSON.stringify({translations:[{text:'translated',detected_source_language:'EN'}]})}},
                CancelHttpProxy:async()=>{}}};
        vm.runInNewContext(script,{window,document:{getElementById:field},URLSearchParams,setTimeout,clearTimeout,console});
        listeners.get('load')();listeners.get('watools:context-ready')();await tick();await tick();
        assert.equal(keyReads,1);
        if(windowMode==='inline') {assert.equal(requests,1);assert.equal(field('target-text').value,'translated')}
        else {
            assert.equal(requests,0);
            const saved={source:'edited source',target:'preserved result',detected:'检测结果 EN',status:'complete',statusClass:'status success',
                settingsVisible:false,selectionStart:1,selectionEnd:4,sourceScrollTop:11,targetScrollTop:12};
            await window.watoolsSession.restore(saved);
            listeners.get('watools:context-ready')();await tick();
            assert.equal(requests,0);assert.equal(field('source-text').value,saved.source);assert.equal(field('target-text').value,saved.target);
        }
    }
});

test('JSON restoration claims the model before initialization and ignores late launch context', async () => {
    const html=fs.readFileSync(path.join(root,'plugins/official/watools.plugin.json/plugin/index.html'),'utf8');
    let initialized,modelText='';const state={editor:{},windowRestored:false};
    const globals={state,editorReady:new Promise(resolve=>{initialized=resolve}),getModel:()=>({setValue:value=>{modelText=value}}),
        normalizeLineEndings:value=>value,pathInput:windowField(''),extractOutput:windowField(''),
        togglePanel(){},validateNow(){},updateActionState(){},focusEditor(){}};
    const hooks=officialWindowHooks('json',globals);const task=hooks.restore({text:'edited',path:'',output:'',previewCollapsed:true});
    assert.equal(state.windowRestored,true);
    const start=html.indexOf('    const applyHostContext =');const end=html.indexOf('\n    };',start)+7;
    const context={state,window:{watools:{session:{windowMode:'inline'}}}};
    vm.runInNewContext(html.slice(start,end)+'\nthis.apply=applyHostContext;',context);
    context.apply({input:{value:'late original'}}); // No original lookup or parsing is permitted after restore claims the model.
    initialized();await task;assert.equal(modelText,'edited');
    context.apply({input:{value:'another original'}});assert.equal(modelText,'edited');
});

function logQueryHarness() {
    const {createLogQueryController, logQueryReducer, initialLogQueryState} = loadTS('frontend/src/lib/log-query.ts');
    let state = initialLogQueryState;
    const requests = [], actions = [];
    const controller = createLogQueryController(query => new Promise((resolve, reject) => {
        requests.push({query, resolve, reject});
    }), action => {actions.push(action); state = logQueryReducer(state, action)});
    const result = (ids, nextCursor) => ({records: ids.map(id => ({id, message: id})), nextCursor,
        availableSources: ['app-backend'], availablePluginIds: [], totalMatched: ids.length});
    return {controller, requests, actions, result, get state() {return state}};
}

test('log queries ignore old responses without clearing the new request loading state', async () => {
    const h = logQueryHarness();
    const old = h.controller.replace({keyword: 'old'});
    const current = h.controller.replace({keyword: 'new'});
    h.requests[0].resolve(h.result(['old'], 'old-cursor'));
    await old;
    assert.equal(h.state.isLoadingLogs, true);
    assert.equal(h.state.records.length, 0);
    h.requests[1].resolve(h.result(['new'], 'new-cursor'));
    await current;
    assert.equal(h.state.records[0].id, 'new');
    assert.equal(h.state.nextCursor, 'new-cursor');
    assert.equal(h.state.isLoadingLogs, false);
});

test('late log query failures cannot replace the current result or its error', async () => {
    const h = logQueryHarness();
    const old = h.controller.replace({keyword: 'old'});
    const current = h.controller.replace({keyword: 'new'});
    h.requests[1].resolve(h.result(['new']));
    await current;
    const count = h.actions.length;
    h.requests[0].reject(new Error('old failure'));
    await old;
    assert.equal(h.actions.length, count);
    assert.equal(h.state.records[0].id, 'new');
    assert.equal(h.state.error, null);
});

test('log pagination uses its query snapshot and cannot append into changed filters', async () => {
    const h = logQueryHarness();
    const files = ['first.log'];
    const first = h.controller.replace({files, keyword: 'old'});
    files.push('other.log');
    h.requests[0].resolve(h.result(['first'], 'page-two'));
    await first;
    const append = h.controller.append();
    await h.controller.append();
    assert.equal(h.requests.length, 2, 'duplicate pagination is suppressed');
    assert.equal(JSON.stringify(h.requests[1].query.files), '["first.log"]');
    assert.equal(h.requests[1].query.cursor, 'page-two');
    const current = h.controller.replace({keyword: 'new'});
    h.requests[1].resolve(h.result(['old-page'], 'page-three'));
    await append;
    assert.equal(h.state.isLoadingLogs, true);
    assert.equal(h.state.isLoadingMore, false);
    h.requests[2].resolve(h.result(['new']));
    await current;
    assert.equal(JSON.stringify(h.state.records.map(record => record.id)), '["new"]');
    await h.controller.append();
    assert.equal(h.requests.length, 3, 'exhausted query cannot reuse an old cursor');
});

test('invalidated log queries do not publish results, errors or finalizers after unmount', async () => {
    for (const reject of [false, true]) {
        const h = logQueryHarness();
        const pending = h.controller.replace({keyword: 'pending'});
        h.controller.invalidate();
        const count = h.actions.length;
        if (reject) h.requests[0].reject(new Error('unmounted'));
        else h.requests[0].resolve(h.result(['unmounted']));
        await pending;
        assert.equal(h.actions.length, count);
        await h.controller.append();
        assert.equal(h.requests.length, 1);
    }
});

test('log reducer preserves selection by ID and is safe to replay', () => {
    const {logQueryReducer, initialLogQueryState} = loadTS('frontend/src/lib/log-query.ts');
    const first = {records: [{id: 'one'}, {id: 'two'}], availableSources: [], availablePluginIds: [], totalMatched: 2};
    let state = logQueryReducer(initialLogQueryState, {type: 'result', result: first, append: false});
    state = logQueryReducer(state, {type: 'select', id: 'two'});
    const action = {type: 'result', result: {...first, records: [{id: 'two', message: 'updated'}]}, append: false};
    const next = logQueryReducer(state, action);
    assert.equal(next.selectedRecordId, 'two');
    assert.equal(next.records[0].message, 'updated');
    assert.equal(state.records.length, 2, 'input state is not mutated');
    assert.equal(JSON.stringify(next), JSON.stringify(logQueryReducer(state, action)), 'replaying has no side effects');
    assert.equal(logQueryReducer(next, {type: 'result', result: {...first, records: [{id:'fallback'}]}, append: false}).selectedRecordId, 'fallback');
    assert.equal(logQueryReducer(next, {type: 'result', result: {...first, records: []}, append: false}).selectedRecordId, null);
    const failedFiles = logQueryReducer(next, {type: 'error', message: 'file listing failed'});
    assert.equal(logQueryReducer(failedFiles, {type: 'reset', preserveError: true}).error, 'file listing failed');
});

test('frontend warning logger uses the warning bridge', async () => {
    const calls = [];
    const mocks = Object.fromEntries(['LogDebug', 'LogError', 'LogInfo', 'LogWarning'].map(name =>
        [name, async message => {calls.push({name, message})}]));
    const {Logger} = loadTS('frontend/src/lib/logger.ts', {'@/lib/legacy-runtime': mocks});
    await Logger.warn('warning probe');
    assert.equal(calls.length, 1);
    assert.equal(calls[0].name, 'LogWarning');
    assert.equal(calls[0].message, '[Fronted] warning probe');
});

test('failed log pagination retains its data and cursor for a successful retry', async () => {
    const h = logQueryHarness();
    const first = h.controller.replace({keyword: 'same'});
    h.requests[0].resolve(h.result(['one'], 'page-two'));
    await first;
    const failed = h.controller.append();
    h.requests[1].reject(new Error('temporary failure'));
    await failed;
    assert.equal(h.state.records[0].id, 'one');
    assert.equal(h.state.nextCursor, 'page-two');
    assert.ok(h.state.error);
    assert.equal(h.state.isLoadingMore, false);
    const retry = h.controller.append();
    assert.equal(h.requests[2].query.cursor, 'page-two');
    assert.equal(h.state.error, null);
    h.requests[2].resolve(h.result(['two']));
    await retry;
    assert.equal(JSON.stringify(h.state.records.map(record => record.id)), '["one","two"]');
});

test('log effect cleanup and restart leave only the restarted request active', async () => {
    const h = logQueryHarness();
    const firstMount = h.controller.replace({keyword: 'same'});
    h.controller.invalidate();
    const secondMount = h.controller.replace({keyword: 'same'});
    h.requests[0].resolve(h.result(['first-mount']));
    await firstMount;
    assert.equal(h.state.isLoadingLogs, true);
    h.requests[1].resolve(h.result(['second-mount']));
    await secondMount;
    assert.equal(h.state.records[0].id, 'second-mount');
    assert.equal(h.state.isLoadingLogs, false);
});

function applicationStoreHarness({write = async () => {}, log = async () => {}} = {}) {
    const requests = [], timers = [], diagnostics = [];
    let listener = null;
    const {useApplicationCommandStore: store} = loadTS('frontend/src/stores/applicationCommandStore.ts', {
        '@/api/command': {
            getApplicationCommands: () => new Promise((resolve, reject) => {requests.push({resolve, reject})}),
            updateApplicationUsage: write,
        },
        '@/lib/legacy-runtime': {EventsOn: (_name, callback) => {listener = callback; return () => {listener = null}}},
        '@/lib/logger': {Logger: {error: message => {diagnostics.push(message); return log(message)}}},
        'fuse.js': {default: require(path.join(root, 'frontend/node_modules/fuse.js'))},
    }, {setTimeout: callback => {timers.push(callback); return timers.length}, clearTimeout() {}});
    const group = (usedCount = 0) => ({category: 'Application', commands: [{id: 'app', triggerId: 'app', name: 'Editor',
        category: 'Application', description: '', path: '/apps/Editor.app', iconPath: '', isUserApp: true,
        pathName: 'Editor.app', namePinyin: null, nameInitial: null, usedCount, lastUsedAt: null}]});
    return {store, requests, group, timers, diagnostics, emit() {assert.ok(listener, 'listener must survive first failure'); return listener()}};
}

test('application initial query failure is explicit, safely consumed and can recover through its listener', async () => {
    const h = applicationStoreHarness();
    await tick();
    assert.equal(h.store.getState().isLoading, true);
    h.requests[0].reject(new Error('private database path and cause'));
    await tick();
    assert.equal(h.store.getState().isLoading, false);
    assert.equal(h.store.getState().commandGroup, null);
    assert.ok(h.store.getState().loadError.includes('请重试'));
    assert.equal(h.store.getState().loadError.includes('private'), false);
    assert.equal(h.emit(), undefined, 'event callback owns its promise rejection');
    await tick();
    h.requests[1].reject(new Error('event query failed'));
    await tick(); // node:test also fails on any unhandled rejection.
    assert.ok(h.store.getState().loadError);
    const retry = h.store.getState().refreshCommands();
    await tick();
    h.requests[2].resolve(h.group());
    await retry;
    assert.equal(h.store.getState().loadError, null);
    assert.equal(h.store.getState().searchCommands('Editor')[0].id, 'app');
});

test('failed application refresh rejects all coalesced callers and keeps previous searchable commands', async () => {
    const h = applicationStoreHarness();
    await tick();
    h.requests[0].resolve(h.group());
    await tick();
    const previous = h.store.getState().commandGroup, fuse = h.store.getState().fuse;
    const first = h.store.getState().refreshCommands();
    const second = h.store.getState().refreshCommands();
    const outcome = Promise.allSettled([first, second]);
    await tick();
    assert.equal(h.requests.length, 2, 'one query serves both refresh callers');
    assert.equal(h.store.getState().isLoading, true);
    assert.equal(h.store.getState().searchCommands('Editor')[0].id, 'app');
    const failure = new Error('query unavailable');
    h.requests[1].reject(failure);
    const results = await outcome;
    assert.equal(results[0].status, 'rejected');
    assert.equal(results[1].status, 'rejected');
    assert.equal(results[0].reason, failure);
    assert.equal(results[1].reason, failure);
    assert.equal(h.store.getState().commandGroup, previous);
    assert.equal(h.store.getState().fuse, fuse);
    assert.equal(h.store.getState().searchCommands('Editor')[0].id, 'app');
    assert.ok(h.store.getState().loadError);
});

test('a successful empty application query clears stale commands and previous load errors', async () => {
    const h = applicationStoreHarness();
    await tick();
    h.requests[0].resolve(h.group());
    await tick();
    const failed = h.store.getState().loadCommands();
    const assertion = assert.rejects(failed);
    h.requests[1].reject(new Error('temporary query failure'));
    await assertion;
    const recovered = h.store.getState().loadCommands();
    h.requests[2].resolve({category: 'Application', commands: []});
    await recovered;
    assert.equal(h.store.getState().loadError, null);
    assert.equal(h.store.getState().commandGroup.commands.length, 0);
    assert.equal(h.store.getState().searchCommands('Editor').length, 0);
});

test('usage saved during an application reload survives an older successful response', async () => {
    const writes = [];
    const h = applicationStoreHarness({write: async items => {writes.push(items)}});
    await tick();
    h.requests[0].resolve(h.group(2));
    await tick();
    const refreshed = h.store.getState().refreshCommands();
    await tick();
    await h.store.getState().updateCommandUsage('app');
    assert.equal(h.store.getState().updateBuffer.size, 0, 'new usage has already been acknowledged');
    h.requests[1].resolve(h.group(2));
    await refreshed;
    assert.equal(h.store.getState().commandGroup.commands[0].usedCount, 3);
    assert.ok(h.store.getState().commandGroup.commands[0].lastUsedAt instanceof Date);
    assert.equal(writes[0][0].usedCount, 3);
    await h.store.getState().updateCommandUsage('app');
    assert.equal(writes[1][0].usedCount, 4, 'the next usage must not reuse an older count');
});

test('unsaved usage blocks application refresh and remains buffered across direct reload', async () => {
    const h = applicationStoreHarness({write: async () => {throw new Error('private write failure')}});
    await tick();
    h.requests[0].resolve(h.group(2));
    await tick();
    await h.store.getState().updateCommandUsage('app');
    await assert.rejects(h.store.getState().refreshCommands(), /使用记录尚未保存/);
    assert.equal(h.requests.length, 1, 'refresh must not fetch over an unsaved buffer');
    const buffer = h.store.getState().updateBuffer;
    const loaded = h.store.getState().loadCommands();
    h.requests[1].resolve(h.group(2));
    await loaded;
    assert.equal(h.store.getState().updateBuffer, buffer);
    assert.equal(h.store.getState().updateBuffer.get('app').usedCount, 3);
    assert.equal(h.store.getState().commandGroup.commands[0].usedCount, 3);
    assert.ok(h.diagnostics.every(message => !message.includes('private')));
});

test('mock application query distinguishes rejection from empty success and later recovery', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors});
    const {HostError} = errors;
    let phase = 'failure';
    const application = {id:'probe', triggerId:'probe', name:'Probe', category:'Application', description:'', path:'/mock',
        iconPath:'', isUserApp:true, usedCount:0, lastUsedAt:null};
    const host = createMockHost({applicationQuery: async () => {
        if (phase === 'failure') throw new HostError('OPERATION_FAILED', '应用列表加载失败，请重试', 'command.list');
        return phase === 'empty' ? [] : [application];
    }});
    await assert.rejects(host.GetApplicationCommandsApi(), error => error.code === 'OPERATION_FAILED');
    phase = 'empty';
    assert.equal((await host.GetApplicationCommandsApi()).length, 0);
    phase = 'success';
    const result = await host.GetApplicationCommandsApi();
    assert.equal(result[0].id, 'probe');
    result[0].name = 'mutated';
    assert.equal((await host.GetApplicationCommandsApi())[0].name, 'Probe');
});

test('application reload accepts authoritative usage when no local update occurred during the query', async () => {
    const h = applicationStoreHarness();
    await tick();
    h.requests[0].resolve(h.group(10));
    await tick();
    await h.store.getState().updateCommandUsage('app');
    const reload = h.store.getState().loadCommands();
    h.requests[1].resolve(h.group(4));
    await reload;
    assert.equal(h.store.getState().commandGroup.commands[0].usedCount, 4,
        'previously saved local statistics must not indefinitely override new host data');
});

test('Wails application adapter normalizes query rejection and preserves successful empty lists', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    let reject = true;
    const bindings = Object.fromEntries(['app', 'command', 'plugin', 'storage', 'system', 'update', 'log'].map(name =>
        [`../../../bindings/watools/internal/host/${name}service`, {}]));
    bindings['../../../bindings/watools/internal/host/commandservice'] = {
        GetApplicationCommandsApi: async () => {
            if (reject) throw {cause: {code: 'OPERATION_FAILED', message: '应用列表加载失败，请重试', operation: 'command.list'}};
            return [];
        },
    };
    const {wailsHost} = loadTS('frontend/src/api/host/wails.ts', {...bindings, './errors': errors});
    await assert.rejects(wailsHost.GetApplicationCommandsApi(), error =>
        error.code === 'OPERATION_FAILED' && error.operation === 'command.list' && error.message === '应用列表加载失败，请重试');
    reject = false;
    assert.equal((await wailsHost.GetApplicationCommandsApi()).length, 0);
});

test('application item hook keeps existing results visible while a refresh is loading', () => {
    const command = {id:'app', triggerId:'app', name:'Editor', path:'/apps/Editor.app', iconPath:'', usedCount:2, lastUsedAt:null};
    const state = {commandGroup:{commands:[command]}, isLoading:true, searchCommands: () => [command]};
    const {useApplicationItems} = loadTS('frontend/src/components/watools/wa-application-item.tsx', {
        react: {useMemo: factory => factory()},
        'react/jsx-runtime': {jsx: (type, props) => ({type, props})},
        '@/components/watools/wa-icon': {WaIcon: () => null},
        '@/stores/applicationCommandStore': {useApplicationCommandStore: () => state},
        '@/lib/command-ranking': {compareRankableItems: () => 0},
    });
    const items = useApplicationItems({searchKey:'Editor', rankingContext:{}, rankingHistory:[], onTriggerCommand() {}});
    assert.equal(items.length, 1);
    assert.equal(items[0].title, 'Editor');
});

test('application changes during a query fetch a fresh snapshot and coalesce event bursts', async () => {
    const h = applicationStoreHarness();
    await tick();
    h.emit();
    h.emit();
    assert.equal(h.requests.length, 1, 'events wait for the pending query');
    h.requests[0].resolve(h.group(1));
    await tick();
    assert.equal(h.requests.length, 2, 'pending events require one fresh query');
    h.emit();
    h.emit();
    h.requests[1].resolve(h.group(2));
    await tick();
    assert.equal(h.requests.length, 3, 'an event during the follow-up requires the latest snapshot');
    h.requests[2].resolve(h.group(3));
    await tick();
    await tick();
    assert.equal(h.store.getState().commandGroup.commands[0].usedCount, 3);
    assert.equal(h.requests.length, 3, 'completed bursts do not cause an automatic refresh loop');
});

test('mock queried applications become the trigger and usage snapshot; failed queries retain it', async () => {
    const errors = loadTS('frontend/src/api/host/errors.ts');
    const {createMockHost} = loadTS('frontend/src/api/host/mock.ts', {'./errors': errors});
    const record = {id:'queried', triggerId:'queried', name:'Queried', category:'Application', description:'', path:'/mock',
        iconPath:'', isUserApp:true, usedCount:0, lastUsedAt:null};
    const options = {applications: [], applicationQuery: async () => [record]};
    const host = createMockHost(options);
    const queried = await host.GetApplicationCommandsApi();
    queried[0].triggerId = 'mutated';
    record.triggerId = 'provider-mutated';
    await host.TriggerCommandApi('queried', 'Application');
    options.applicationQuery = async () => {throw new Error('query failure')};
    await assert.rejects(host.GetApplicationCommandsApi());
    await host.TriggerCommandApi('queried', 'Application');
    await host.UpdateApplicationUsageApi([{id:'queried', lastUsedAt:'2026-10-07T12:00:00Z', usedCount:4}]);
    options.applicationQuery = undefined;
    const retained = await host.GetApplicationCommandsApi();
    assert.equal(retained[0].triggerId, 'queried');
    assert.equal(retained[0].usedCount, 4);
    assert.equal(retained[0].lastUsedAt, '2026-10-07T12:00:00Z');
});

test('synchronous diagnostic failure cannot discard usage or skip its retry', async () => {
    const h = applicationStoreHarness({write: async () => {throw new Error('write failure')},
        log: () => {throw new Error('synchronous logger failure')}});
    await tick();
    h.requests[0].resolve(h.group());
    await tick();
    await h.store.getState().updateCommandUsage('app');
    await tick();
    assert.equal(h.store.getState().updateBuffer.get('app').usedCount, 1);
    assert.equal(h.timers.length, 1, 'retry is scheduled independently of diagnostic logging');
    await assert.rejects(h.store.getState().refreshCommands(), /使用记录尚未保存/);
    await tick();
    assert.ok(h.store.getState().loadError);
    assert.equal(h.requests.length, 1);
});
