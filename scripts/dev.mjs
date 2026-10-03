#!/usr/bin/env node
import {spawn, spawnSync} from 'node:child_process';
import {chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const frontend = path.join(root, 'frontend');
const goMod = readFileSync(path.join(root, 'go.mod'), 'utf8');
const toolchain = goMod.match(/^toolchain (go\S+)$/m)?.[1];
const wailsVersion = goMod.match(/^(?:require\s+)?\s*github\.com\/wailsapp\/wails\/v2 (v\S+)/m)?.[1];
const packageJSON = JSON.parse(readFileSync(path.join(frontend, 'package.json'), 'utf8'));
const pnpmVersion = packageJSON.packageManager?.match(/^pnpm@([\d.]+)$/)?.[1];
const state = path.join(root, '.cache', 'dev');
const tools = path.join(root, '.cache', 'tools');
const [task = 'help', ...args] = process.argv.slice(2);
const commands = ['doctor', 'setup', 'dev', 'browser', 'bindings', 'check', 'check-race', 'build', 'plugins'];

const help = `WaTools macOS development (run from any directory)

  node scripts/dev.mjs doctor       Check Node, pinned Go/pnpm, macOS SDK and paths
  node scripts/dev.mjs setup        Download Go modules; install the frontend lockfile
  node scripts/dev.mjs dev          Start Wails with isolated development data
  node scripts/dev.mjs browser      Start Wails and open its browser debugging URL
  node scripts/dev.mjs bindings     Build assets and regenerate Wails bindings
  node scripts/dev.mjs check        Workflow tests, frontend tests/build, Go tests/vet/build
  node scripts/dev.mjs check-race   Build assets and run Go race tests
  node scripts/dev.mjs build        Build a native application in build/bin
  node scripts/dev.mjs plugins list|package|install [package-id...]

Development data: ${path.join(state, 'data')}
Development cache/logs: ${path.join(state, 'cache')}
Plugin packages: ${path.join(state, 'packages')}
Checks use disposable data/cache directories. No task resets user data.
dev/browser/build accept additional Wails flags; other tasks accept no flags.
`;

const env = {
    ...process.env,
    GOTOOLCHAIN: toolchain,
    GOWORK: 'off',
    GOCACHE: process.env.GOCACHE || path.join(tools, 'go-build'),
    GOMODCACHE: process.env.GOMODCACHE || path.join(tools, 'go-mod'),
    GOPATH: process.env.GOPATH || path.join(tools, 'gopath'),
    WATOOLS_DATA_DIR: path.join(state, 'data'),
    WATOOLS_CACHE_DIR: path.join(state, 'cache'),
    WATOOLS_LOG_LEVEL: process.env.WATOOLS_LOG_LEVEL || 'debug',
};
let interrupted;
let cleanupSafe = true;

async function waitForProcessGroup(pid) {
    const deadline = Date.now() + 5000;
    while (true) {
        try {
            process.kill(-pid, 0);
        } catch (error) {
            if (error.code === 'ESRCH') return;
            throw error;
        }
        if (Date.now() >= deadline) {
            cleanupSafe = false;
            throw new Error(`Task process group ${pid} did not exit within 5 seconds. Temporary data is preserved while processes may still be using it.`);
        }
        await new Promise(resolve => setTimeout(resolve, 25));
    }
}

function probe(command, commandArgs, cwd = root) {
    const result = spawnSync(command, commandArgs, {cwd, env, encoding: 'utf8'});
    if (result.error || result.status !== 0) {
        throw new Error(`${command} ${commandArgs.join(' ')}: ${result.error?.message || result.stderr?.trim() || `exit ${result.status}`}`);
    }
    return result.stdout.trim();
}

function preflight() {
    if (!toolchain || !wailsVersion || !pnpmVersion) {
        throw new Error('Missing pinned toolchain/Wails in go.mod or pnpm in frontend/package.json.');
    }
    const [major, minor] = process.versions.node.split('.').map(Number);
    if (major < 22 || (major === 22 && minor < 12)) {
        throw new Error(`Node ${process.versions.node} is unsupported; use Node >=22.12 (CI uses Node 22).`);
    }
    if (process.platform !== 'darwin') {
        throw new Error('This development entry currently targets macOS. Existing Windows CI remains separate.');
    }
    const goVersion = probe('go', ['version']);
    if (!goVersion.startsWith(`go version ${toolchain} `)) {
        throw new Error(`Expected ${toolchain}; received ${goVersion}. Check your Go launcher and toolchain download.`);
    }
    if (task !== 'plugins') {
        const actual = probe('pnpm', ['--version'], frontend);
        if (actual !== pnpmVersion) {
            throw new Error(`Expected pnpm ${pnpmVersion}; received ${actual}. Enable Corepack or install the pinned pnpm version.`);
        }
    }
    const sdk = probe('xcrun', ['--show-sdk-path']);
    if (!sdk) throw new Error('macOS SDK unavailable. Install/select Xcode Command Line Tools.');
    let macOSVersion;
    if (['doctor', 'dev', 'browser', 'build'].includes(task)) {
        try {
            macOSVersion = probe('sysctl', ['-n', 'kern.osproductversion']);
        } catch (error) {
            throw new Error(`Wails v2 needs a read-only macOS version query. If an agent sandbox blocks sysctl, run this native task with permission to query system information. ${error.message}`);
        }
    }
    if (!['doctor', 'setup', 'plugins'].includes(task) && !existsSync(path.join(frontend, 'node_modules', 'typescript'))) {
        throw new Error('Frontend dependencies missing. Run: node scripts/dev.mjs setup');
    }
    if (task === 'doctor') {
        console.log(`Node: ${process.version}\nGo: ${goVersion}\npnpm: ${pnpmVersion}\nWails CLI/module: ${wailsVersion}\nmacOS: ${macOSVersion} (${process.arch})\nmacOS SDK: ${sdk}`);
        console.log(`Frontend dependencies: ${existsSync(path.join(frontend, 'node_modules', 'typescript')) ? 'installed' : 'missing; run setup'}`);
        console.log(`Data: ${env.WATOOLS_DATA_DIR}\nCache/logs: ${env.WATOOLS_CACHE_DIR}\nGo cache: ${env.GOCACHE}\nGo modules: ${env.GOMODCACHE}`);
    }
}

async function run(command, commandArgs, cwd = root) {
    if (interrupted) throw new Error(`Task interrupted by ${interrupted}.`);
    console.log(`\n> ${command} ${commandArgs.join(' ')}`);
    await new Promise((resolve, reject) => {
        // go run ignores SIGINT itself. Give the task its own process group so
        // the actual CLI also receives cancellation and can clean up Vite.
        const child = spawn(command, commandArgs, {cwd, env, stdio: 'inherit', detached: true});
        const stop = signal => {
            interrupted ||= signal;
            if (!child.pid) return;
            try {
                process.kill(-child.pid, signal);
            } catch (error) {
                if (error.code !== 'ESRCH') console.error(`Unable to stop task process group: ${error.message}`);
            }
        };
        const interrupt = () => stop('SIGINT');
        const terminate = () => stop('SIGTERM');
        process.on('SIGINT', interrupt);
        process.on('SIGTERM', terminate);
        const cleanup = () => {
            process.off('SIGINT', interrupt);
            process.off('SIGTERM', terminate);
        };
        child.on('error', error => { cleanup(); reject(error); });
        child.on('close', async (code, signal) => {
            try {
                // SIGTERM may end go before the CLI's deferred cleanup. Wait
                // for the task group before deleting its data or touching files.
                if (interrupted) await waitForProcessGroup(child.pid);
                if (code === 0 && !interrupted) resolve();
                else reject(new Error(`${command} failed (${interrupted || signal || `exit ${code}`}).`));
            } catch (error) {
                reject(error);
            } finally {
                cleanup();
            }
        });
    });
}

const pnpm = (...commandArgs) => run('pnpm', commandArgs, frontend);
const go = (...commandArgs) => run('go', commandArgs);
// Wails v2 writes some runtime assets with executable bits. These generated
// JS/TS/JSON files are source modules, so keep regeneration free of mode churn.
function normalizeBindings(directory = path.join(frontend, 'wailsjs')) {
    if (!existsSync(directory)) return;
    for (const entry of readdirSync(directory, {withFileTypes: true})) {
        const file = path.join(directory, entry.name);
        if (entry.isDirectory()) normalizeBindings(file);
        else if (entry.isFile() && /\.(js|ts|json)$/.test(entry.name)) chmodSync(file, 0o644);
    }
}
async function wails(...commandArgs) {
    try {
        await go('run', `github.com/wailsapp/wails/v2/cmd/wails@${wailsVersion}`, ...commandArgs);
    } finally {
        if (cleanupSafe) normalizeBindings();
    }
}

async function main() {
    if (['help', '--help', '-h'].includes(task)) return console.log(help);
    if (!commands.includes(task)) throw new Error(`Unknown task: ${task}. Run: node scripts/dev.mjs help`);
    if (task === 'plugins') {
        if (!['list', 'package', 'install'].includes(args[0]) || args.slice(1).some(arg => arg.startsWith('-')) || (args[0] === 'list' && args.length !== 1)) {
            throw new Error('Usage: node scripts/dev.mjs plugins list|package|install [package-id...]');
        }
    } else if (!['dev', 'browser', 'build'].includes(task) && args.length) {
        throw new Error(`${task} accepts no arguments.`);
    }
    preflight();
    if (task === 'doctor') return;

    let temporary;
    if (['check', 'check-race', 'bindings', 'build'].includes(task)) {
        temporary = mkdtempSync(path.join(tmpdir(), 'watools-check-'));
        env.WATOOLS_DATA_DIR = path.join(temporary, 'data');
        env.WATOOLS_CACHE_DIR = path.join(temporary, 'cache');
    }
    try {
        for (const directory of [env.WATOOLS_DATA_DIR, env.WATOOLS_CACHE_DIR]) mkdirSync(directory, {recursive: true, mode: 0o700});
        switch (task) {
            case 'setup':
                await go('mod', 'download');
                await pnpm('install', '--frozen-lockfile');
                break;
            case 'dev':
            case 'browser':
                await wails('dev', '-m', ...(task === 'browser' ? ['-browser'] : []), ...args);
                break;
            case 'bindings':
                await pnpm('build');
                await wails('generate', 'module');
                break;
            case 'check':
                await run(process.execPath, ['--test', 'scripts/dev-workflow.test.mjs']);
                await pnpm('test');
                await pnpm('build');
                await go('test', './...');
                await go('vet', './...');
                await go('build', '-tags', 'desktop,production', './...');
                break;
            case 'check-race':
                await pnpm('build');
                await go('test', '-race', './...');
                break;
            case 'build':
                await wails('build', '-m', ...args);
                break;
            case 'plugins': {
                const [action, ...ids] = args;
                await go('run', './cmd/pluginctl', action, ...(action === 'list' ? [] : ['--output', path.join(state, 'packages')]), ...ids);
                break;
            }
        }
    } finally {
        if (temporary && cleanupSafe) rmSync(temporary, {recursive: true, force: true});
        else if (temporary) console.error(`Preserved temporary data: ${temporary}`);
    }
}

main().catch(error => {
    console.error(`\nWaTools: ${error.message}`);
    process.exitCode = 1;
});
