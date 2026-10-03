import {test} from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, statSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const script = fileURLToPath(new URL('./dev.mjs', import.meta.url));
// These integration tests exercise the macOS entry with fake external tools,
// including environment leakage, paths containing spaces and child failures.
const macOS = {skip: process.platform !== 'darwin'};

function fixture(t, extraEnv = {}) {
    const temporary = realpathSync(mkdtempSync(path.join(tmpdir(), 'watools workflow ')));
    const root = path.join(temporary, 'checkout with spaces');
    const bin = path.join(temporary, 'bin');
    const record = path.join(temporary, 'calls.jsonl');
    const pids = path.join(temporary, 'owned-pids.jsonl');
    const stopped = path.join(temporary, 'host-stopped');
    t.after(() => {
        // Only groups created and recorded by this fixture are eligible for
        // cleanup, including when a deliberately broken wrapper times out.
        try {
            if (!existsSync(pids)) return;
            for (const line of readFileSync(pids, 'utf8').trim().split('\n')) {
                const {pid, group} = JSON.parse(line);
                try { process.kill(group ? -pid : pid, 'SIGKILL'); }
                catch (error) { if (error.code !== 'ESRCH') throw error; }
            }
        } finally {
            rmSync(temporary, {recursive: true, force: true});
        }
    });
    const userData = path.join(temporary, 'user-data');
    const userCache = path.join(temporary, 'user-cache');
    for (const dir of ['scripts', 'frontend/node_modules/typescript']) mkdirSync(path.join(root, dir), {recursive: true});
    mkdirSync(bin);
    for (const dir of [userData, userCache]) {
        mkdirSync(dir);
        writeFileSync(path.join(dir, 'sentinel'), 'private-data');
    }
    copyFileSync(script, path.join(root, 'scripts/dev.mjs'));
    writeFileSync(path.join(root, 'go.mod'), 'module fixture\n\ngo 1.26.0\n\ntoolchain go1.26.8\n\nrequire (\n github.com/wailsapp/wails/v2 v2.10.2\n)\n');
    writeFileSync(path.join(root, 'frontend/package.json'), '{"packageManager":"pnpm@10.21.0"}');
    const host = path.join(temporary, 'host.cjs');
    writeFileSync(host, `
const {spawn} = require('node:child_process');
const fs = require('node:fs');
const env = process.env;
const record = (pid, group) => fs.appendFileSync(env.WATOOLS_TEST_PIDS, JSON.stringify({pid, group}) + '\\n');
record(process.pid, false);
const service = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {detached: true, stdio: 'ignore'});
record(service.pid, true);
let stopping = false;
const stop = () => {
  if (stopping) return;
  stopping = true;
  setTimeout(() => {
    service.once('close', () => {
      fs.writeFileSync(env.WATOOLS_TEST_STOPPED, 'cleaned');
      process.exit(0);
    });
    process.kill(-service.pid, 'SIGTERM');
  }, 200);
};
process.on('SIGINT', stop);
process.on('SIGTERM', stop);
process.send('ready');
`);
    const fakeTool = `#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const tool = path.basename(process.argv[1]);
const args = process.argv.slice(2);
const env = process.env;
fs.appendFileSync(env.WATOOLS_TEST_RECORD, JSON.stringify({tool, args, cwd: process.cwd(),
  data: env.WATOOLS_DATA_DIR, cache: env.WATOOLS_CACHE_DIR, toolchain: env.GOTOOLCHAIN,
  goCache: env.GOCACHE, goModules: env.GOMODCACHE, goPath: env.GOPATH}) + '\\n');
if (tool === 'go' && args[0] === 'version') console.log('go version ' + env.GOTOOLCHAIN + ' darwin/arm64');
if (tool === 'pnpm' && args[0] === '--version') console.log(env.WATOOLS_TEST_PNPM || '10.21.0');
if (tool === 'xcrun') console.log('/fake/macOS.sdk');
if (tool === 'sysctl') console.log('26.6.2');
if (tool === 'go' && args[0] === 'test' && env.WATOOLS_TEST_FAIL) process.exit(7);
if (tool === 'go' && args[0] === 'run' && env.WATOOLS_TEST_INTERRUPT) {
  setTimeout(() => process.kill(process.ppid, 'SIGINT'), 50);
  setInterval(() => {}, 1000);
}
if (tool === 'go' && args[0] === 'run' && env.WATOOLS_TEST_TREE_SIGNAL) {
  const {spawn} = require('node:child_process');
  fs.appendFileSync(env.WATOOLS_TEST_PIDS, JSON.stringify({pid: process.pid, group: true}) + '\\n');
  // Like go run: ignore INT on the wrapper and wait for the actual program.
  // TERM uses the default action, so the host must receive the group signal.
  process.on('SIGINT', () => {});
  const host = spawn(process.execPath, [env.WATOOLS_TEST_HOST], {stdio: ['ignore', 'ignore', 'ignore', 'ipc']});
  host.once('message', () => process.kill(process.ppid, env.WATOOLS_TEST_TREE_SIGNAL));
  host.once('close', code => process.exit(code || 0));
}
if (tool === 'go' && args[0] === 'mod' && env.WATOOLS_TEST_CANCEL_SUCCESS) {
  process.on(env.WATOOLS_TEST_CANCEL_SUCCESS, () => process.exit(0));
  setTimeout(() => process.kill(process.ppid, env.WATOOLS_TEST_CANCEL_SUCCESS), 50);
}
`;
    for (const tool of ['go', 'pnpm', 'xcrun', 'sysctl']) writeFileSync(path.join(bin, tool), fakeTool, {mode: 0o755});
    const env = {...process.env, PATH: `${bin}${path.delimiter}${process.env.PATH}`,
        WATOOLS_TEST_RECORD: record, WATOOLS_TEST_PIDS: pids, WATOOLS_TEST_HOST: host,
        WATOOLS_TEST_STOPPED: stopped, WATOOLS_DATA_DIR: userData, WATOOLS_CACHE_DIR: userCache,
        GOTOOLCHAIN: 'go1.99.0', GOCACHE: '', GOMODCACHE: '', GOPATH: '', ...extraEnv};
    return {
        root,
        stopped,
        run(...args) {
            const result = spawnSync(process.execPath, [path.join(root, 'scripts/dev.mjs'), ...args], {cwd: temporary, env, encoding: 'utf8', timeout: 10000});
            for (const dir of [userData, userCache]) {
                assert.equal(readFileSync(path.join(dir, 'sentinel'), 'utf8'), 'private-data');
                assert.deepEqual(readdirSync(dir), ['sentinel']);
            }
            return {...result, calls: existsSync(record) ? readFileSync(record, 'utf8').trim().split('\n').map(line => JSON.parse(line)) : []};
        },
    };
}

test('doctor resolves the checkout from its script and pins tools without creating development data', macOS, t => {
    const f = fixture(t);
    const result = f.run('doctor');
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /Wails CLI\/module: v2\.10\.2/);
    assert.equal(existsSync(path.join(f.root, '.cache/dev')), false);
    for (const call of result.calls) {
        assert.equal(call.toolchain, 'go1.26.8');
        assert.equal(call.data, path.join(f.root, '.cache/dev/data'));
        assert.equal(call.cache, path.join(f.root, '.cache/dev/cache'));
        assert.equal(call.goCache, path.join(f.root, '.cache/tools/go-build'));
    }
    assert.equal(result.calls.find(call => call.tool === 'pnpm').cwd, path.join(f.root, 'frontend'));
});

test('dev uses the module CLI version and forwards Wails flags as separate arguments', macOS, t => {
    const f = fixture(t);
    const result = f.run('dev', '-appargs', 'two words; $(never-run)');
    assert.equal(result.status, 0, result.stderr);
    const call = result.calls.at(-1);
    assert.deepEqual(call.args, ['run', 'github.com/wailsapp/wails/v2/cmd/wails@v2.10.2', 'dev', '-m', '-appargs', 'two words; $(never-run)']);
    assert.equal(call.cwd, f.root);
    assert.equal(call.data, path.join(f.root, '.cache/dev/data'));
    assert.equal(existsSync(call.data), true);
});

test('browser connects through Wails rather than starting a hostless Vite server', macOS, t => {
    const f = fixture(t);
    const result = f.run('browser');
    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(result.calls.at(-1).args.slice(2), ['dev', '-m', '-browser']);
});

test('plugin installation shares the development installation and directs packages away from source', macOS, t => {
    const f = fixture(t);
    const result = f.run('plugins', 'install', 'watools.plugin.calculator');
    assert.equal(result.status, 0, result.stderr);
    const call = result.calls.at(-1);
    assert.deepEqual(call.args, ['run', './cmd/pluginctl', 'install', '--output', path.join(f.root, '.cache/dev/packages'), 'watools.plugin.calculator']);
    assert.equal(call.data, path.join(f.root, '.cache/dev/data'));
});

test('race checks use disposable data and clean up even when the Go command fails', macOS, t => {
    const f = fixture(t, {WATOOLS_TEST_FAIL: '1'});
    const result = f.run('check-race');
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /exit 7/);
    const call = result.calls.at(-1);
    assert.deepEqual(call.args, ['test', '-race', './...']);
    assert.notEqual(call.data, path.join(f.root, '.cache/dev/data'));
    assert.equal(existsSync(path.dirname(call.data)), false);
    assert.equal(existsSync(path.join(f.root, '.cache/dev')), false);
});

test('build uses disposable data and preserves development data', macOS, t => {
    const f = fixture(t);
    const result = f.run('build');
    assert.equal(result.status, 0, result.stderr);
    assert.equal(existsSync(path.dirname(result.calls.at(-1).data)), false);
    assert.equal(existsSync(path.join(f.root, '.cache/dev')), false);
});

test('binding generation builds assets first and removes executable bits without changing generated content', macOS, t => {
    const f = fixture(t);
    const directory = path.join(f.root, 'frontend/wailsjs/runtime');
    mkdirSync(directory, {recursive: true});
    const file = path.join(directory, 'runtime.js');
    writeFileSync(file, 'export const generated = true;', {mode: 0o755});
    const result = f.run('bindings');
    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(result.calls.slice(-2).map(call => call.args), [
        ['build'], ['run', 'github.com/wailsapp/wails/v2/cmd/wails@v2.10.2', 'generate', 'module'],
    ]);
    assert.equal(statSync(file).mode & 0o777, 0o644);
    assert.equal(readFileSync(file, 'utf8'), 'export const generated = true;');
    assert.equal(existsSync(path.dirname(result.calls.at(-1).data)), false);
});

test('interrupting the wrapper stops its child and still cleans temporary data and generated modes', macOS, t => {
    const f = fixture(t, {WATOOLS_TEST_INTERRUPT: '1'});
    const directory = path.join(f.root, 'frontend/wailsjs');
    mkdirSync(directory, {recursive: true});
    const file = path.join(directory, 'runtime.js');
    writeFileSync(file, 'generated', {mode: 0o755});
    const result = f.run('build');
    assert.equal(result.error, undefined, 'interrupted child must not hang');
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /SIGINT/);
    assert.equal(existsSync(path.dirname(result.calls.at(-1).data)), false);
    assert.equal(statSync(file).mode & 0o777, 0o644);
});

for (const signal of ['SIGINT', 'SIGTERM']) {
    test(`${signal} reaches the actual host and waits for delayed cleanup of its separate service group`, macOS, t => {
        const f = fixture(t, {WATOOLS_TEST_TREE_SIGNAL: signal});
        const result = f.run('build');
        assert.equal(result.error, undefined, 'task group cancellation must not hang');
        assert.notEqual(result.status, 0);
        assert.match(result.stderr, new RegExp(signal));
        assert.equal(readFileSync(f.stopped, 'utf8'), 'cleaned');
        assert.equal(existsSync(path.dirname(result.calls.at(-1).data)), false);
    });

    test(`${signal} cancels later setup commands even when the interrupted command exits successfully`, macOS, t => {
        const f = fixture(t, {WATOOLS_TEST_CANCEL_SUCCESS: signal});
        const result = f.run('setup');
        assert.equal(result.error, undefined);
        assert.notEqual(result.status, 0);
        assert.match(result.stderr, new RegExp(signal));
        assert.equal(result.calls.some(call => call.tool === 'pnpm' && call.args.includes('install')), false);
    });
}

test('an incompatible pnpm fails before installing or starting the application', macOS, t => {
    const f = fixture(t, {WATOOLS_TEST_PNPM: '11.2.2'});
    const result = f.run('setup');
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /Expected pnpm 10\.21\.0/);
    assert.equal(result.calls.some(call => call.args.includes('download') || call.args.includes('install')), false);
});

test('unknown tasks and plugin output overrides fail before invoking tools', macOS, t => {
    for (const args of [['unknown'], ['plugins', 'install', '--output', '/user-data'], ['bindings', '-save']]) {
        const f = fixture(t);
        const result = f.run(...args);
        assert.notEqual(result.status, 0);
        assert.deepEqual(result.calls, []);
    }
});
