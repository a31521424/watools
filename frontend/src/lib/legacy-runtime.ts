import {Application, Browser, Clipboard, Events, Screens, System, Window} from '@wailsio/runtime'
import {host} from "@/api/host"

// The v2 runtime shape used by existing trusted plugins. Calls are anchored to
// the launcher; iframe consumers must not resolve an arbitrary current window.
const launcher = Window.Get('launcher')
export const BrowserOpenURL = (url: string) => Browser.OpenURL(url)
export const ClipboardGetText = () => Clipboard.Text()
export const ClipboardSetText = async (text: string): Promise<boolean> => { await host.SetClipboardTextApi(text); return true }
export const WindowGetSize = async () => { const size = await host.GetLauncherSizeApi(); return {w: size.width, h: size.height} }
export const WindowSetSize = (width: number, height: number) => host.SetLauncherSizeApi(width, height)
const callbackData = (callback: (...data: any[]) => void) => (event: Events.WailsEvent) => callback(...(event.data == null ? [] : Array.isArray(event.data) ? event.data : [event.data]))
export const EventsOn = (name: string, callback: (...data: any[]) => void) => Events.On(name, callbackData(callback))
export const EventsOnce = (name: string, callback: (...data: any[]) => void) => Events.Once(name, callbackData(callback))
export const EventsOnMultiple = (name: string, callback: (...data: any[]) => void, count: number) => Events.OnMultiple(name, callbackData(callback), count)
export const EventsEmit = (name: string, ...data: any[]) => Events.Emit(name, data)
export const EventsOff = (name: string, ...names: string[]) => Events.Off(name, ...names)
export const EventsOffAll = () => Events.OffAll()
export const LogDebug = (message: string) => host.LogFrontendApi('debug', message)
export const LogInfo = (message: string) => host.LogFrontendApi('info', message)
export const LogError = (message: string) => host.LogFrontendApi('error', message)
export const LogWarning = (message: string) => host.LogFrontendApi('warn', message)

const windowMethods = (windowName: string) => {
 const target = Window.Get(windowName);
 return {
   WindowGetSize: windowName === 'launcher' ? WindowGetSize : async () => {const size = await target.Size(); return {w:size.width,h:size.height}},
   WindowSetSize: windowName === 'launcher' ? WindowSetSize : (width:number,height:number) => target.SetSize(width,height),
   WindowReload: () => target.Reload(), WindowReloadApp: () => target.ForceReload(),
   WindowCenter: () => target.Center(), WindowSetTitle: (title:string) => target.SetTitle(title),
   WindowSetPosition: (x:number,y:number) => target.SetPosition(x,y), WindowGetPosition: () => target.Position(),
   WindowHide: () => target.Hide(), WindowShow: () => target.Show(), Hide: () => target.Hide(), Show: () => target.Show(),
   WindowSetAlwaysOnTop: (value:boolean) => target.SetAlwaysOnTop(value),
   WindowSetMinSize: (width:number,height:number) => target.SetMinSize(width,height), WindowSetMaxSize: (width:number,height:number) => target.SetMaxSize(width,height),
   WindowFullscreen: () => target.Fullscreen(), WindowUnfullscreen: () => target.UnFullscreen(), WindowIsFullscreen: () => target.IsFullscreen(),
   WindowMaximise: () => target.Maximise(), WindowUnmaximise: () => target.UnMaximise(), WindowToggleMaximise: () => target.ToggleMaximise(), WindowIsMaximised: () => target.IsMaximised(),
   WindowMinimise: () => target.Minimise(), WindowUnminimise: () => target.UnMinimise(), WindowIsMinimised: () => target.IsMinimised(),
   WindowIsNormal: async () => !(await target.IsFullscreen()) && !(await target.IsMaximised()) && !(await target.IsMinimised()),
   WindowSetBackgroundColour: (r:number,g:number,b:number,a:number) => target.SetBackgroundColour(r,g,b,a),
 };
};
export const legacyRuntime = {
 ...windowMethods('launcher'), BrowserOpenURL, ClipboardGetText, ClipboardSetText,
 EventsOn, EventsOnce, EventsOnMultiple, EventsEmit, EventsOff, EventsOffAll,
 LogDebug, LogInfo, LogError, LogWarning, LogPrint: LogInfo, LogTrace: LogDebug, LogFatal: LogError,
 Quit: () => Application.Quit(),
 Environment: async () => { const env = await System.Environment(); return {buildType: env.Debug ? 'dev' : 'production', platform: env.OS, arch: env.Arch} },
 ScreenGetAll: async () => {
   const [screens,current] = await Promise.all([Screens.GetAll(),Screens.GetCurrent()]);
   return screens.map(screen => ({isPrimary:screen.IsPrimary,isCurrent:screen.ID===current.ID,width:screen.Size.Width,height:screen.Size.Height}));
 },
};

// Each iframe owns its subscriptions and targets its containing native window.
export const createPluginRuntime = (windowName: string, instanceId: string) => {
 const target = Window.Get(windowName);
 const listeners = new Map<string, Set<() => void>>();
 const name = (event: string) => `plugin:${instanceId}:${event}`;
 const subscribe = (event: string, callback: (...data: any[]) => void, once = false, count?: number) => {
   let remaining = count ?? (once ? 1 : Infinity);
   const wrapped = (...data: any[]) => {try {callback(...data)} finally {if (--remaining<=0) remove()}};
   const off = count !== undefined ? EventsOnMultiple(name(event), wrapped, count) : once ? EventsOnce(name(event), wrapped) : EventsOn(name(event), wrapped);
   const set = listeners.get(event) ?? new Set(); set.add(off); listeners.set(event, set);
   const remove = () => {off(); set.delete(off); if (!set.size) listeners.delete(event)};
   return remove;
 };
 const dispose = () => {for (const set of listeners.values()) for (const off of set) off(); listeners.clear()};
 return {
   ...legacyRuntime,
   ...windowMethods(windowName),
   Quit: windowName === 'launcher' ? legacyRuntime.Quit : () => target.Close(),
   EventsOn: (event:string,callback:(...data:any[])=>void) => subscribe(event,callback),
   EventsOnce: (event:string,callback:(...data:any[])=>void) => subscribe(event,callback,true),
   EventsOnMultiple: (event:string,callback:(...data:any[])=>void,count:number) => subscribe(event,callback,false,count),
   EventsEmit: (event:string,...data:any[]) => EventsEmit(name(event),...data),
   EventsOff: (...events:string[]) => {for (const event of events) {for (const off of listeners.get(event) ?? []) off(); listeners.delete(event)}},
   EventsOffAll: dispose, dispose,
 };
};
