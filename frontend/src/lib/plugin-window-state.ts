export type PluginStateHooks = {
    capture: () => unknown | Promise<unknown>
    restore: (state: unknown) => void | Promise<void>
};
export type PluginWindowState = {kind: 'hooks' | 'form'; value: unknown};
type PluginFrame = Window & {watoolsSession?: PluginStateHooks};
type Field = {index: number; tag: string; type: string; value: string; checked?: boolean; selected?: boolean[]};

// Existing plugins retain ordinary form controls. Editors with their own models
// provide explicit hooks; no DOM or executable objects cross WebViews.
export const capturePluginWindowState = async (frame: Window): Promise<PluginWindowState> => {
    const hooks = (frame as PluginFrame).watoolsSession;
    if (hooks) {
        let timeout: ReturnType<typeof setTimeout> | undefined;
        try {
            const value = await Promise.race([Promise.resolve().then(()=>hooks.capture()), new Promise<never>((_resolve,reject)=>{
                timeout=setTimeout(()=>reject(new Error('插件状态导出超时，请重试')),30_000);
            })]);
            return {kind:'hooks',value:value ?? null};
        } finally {if (timeout!==undefined) clearTimeout(timeout)}
    }
    if (frame.document.querySelector('.monaco-editor,[contenteditable="true"]')) throw new Error('当前插件版本不支持编辑器状态迁移。请安装新版插件，再返回启动器重新打开；已有编辑内容请先复制保存。');
    const fields: Field[] = [];
    frame.document.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>('input,textarea,select').forEach((field, index) => {
        if (field.type === 'password' || field.type === 'file') return;
        fields.push({index, tag: field.tagName, type: field.type, value: field.value,
            checked: field.tagName === 'INPUT' ? (field as HTMLInputElement).checked : undefined,
            selected: field.tagName === 'SELECT' ? Array.from((field as HTMLSelectElement).options, option => option.selected) : undefined});
    });
    return {kind: 'form', value: fields};
};
export const restorePluginWindowState = async (frame: Window, state: PluginWindowState): Promise<void> => {
    if (state.kind === 'hooks') {
        const hooks = (frame as PluginFrame).watoolsSession;
        if (!hooks) throw new Error('插件未提供窗口状态恢复接口');
        await hooks.restore(state.value);
        return;
    }
    const controls = frame.document.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>('input,textarea,select');
    for (const saved of state.value as Field[]) {
        const field = controls[saved.index];
        if (!field || field.tagName !== saved.tag || field.type !== saved.type) continue;
        field.value = saved.value;
        if (saved.checked !== undefined) (field as HTMLInputElement).checked = saved.checked;
        if (saved.selected) Array.from((field as HTMLSelectElement).options).forEach((option, index) => {option.selected = saved.selected![index] ?? false});
        // Construct events in the target realm for WebKit.
        for (const name of ['input', 'change']) { const event = frame.document.createEvent('Event'); event.initEvent(name, true, false); field.dispatchEvent(event); }
    }
};
