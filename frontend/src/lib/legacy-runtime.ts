import {Application, Browser, Clipboard, Events, Screens, System, Window} from '@wailsio/runtime'
import {LogFrontendApi} from '../../bindings/watools/internal/coordinator/waappcoordinator'

// The v2 runtime shape used by existing trusted plugins. Calls are anchored to
// the launcher; iframe consumers must not resolve an arbitrary current window.
const launcher = Window.Get('launcher')
export const BrowserOpenURL = (url: string) => Browser.OpenURL(url)
export const ClipboardGetText = () => Clipboard.Text()
export const ClipboardSetText = async (text: string): Promise<boolean> => { await Clipboard.SetText(text); return true }
export const WindowGetSize = async () => { const size = await launcher.Size(); return {w: size.width, h: size.height} }
export const WindowSetSize = (width: number, height: number) => launcher.SetSize(width, height)
const callbackData = (callback: (...data: any[]) => void) => (event: Events.WailsEvent) => callback(...(event.data == null ? [] : Array.isArray(event.data) ? event.data : [event.data]))
export const EventsOn = (name: string, callback: (...data: any[]) => void) => Events.On(name, callbackData(callback))
export const EventsOnce = (name: string, callback: (...data: any[]) => void) => Events.Once(name, callbackData(callback))
export const EventsOnMultiple = (name: string, callback: (...data: any[]) => void, count: number) => Events.OnMultiple(name, callbackData(callback), count)
export const EventsEmit = (name: string, ...data: any[]) => Events.Emit(name, data)
export const EventsOff = (name: string, ...names: string[]) => Events.Off(name, ...names)
export const EventsOffAll = () => Events.OffAll()
export const LogDebug = (message: string) => LogFrontendApi('debug', message)
export const LogInfo = (message: string) => LogFrontendApi('info', message)
export const LogError = (message: string) => LogFrontendApi('error', message)
export const LogWarning = (message: string) => LogFrontendApi('warn', message)

export const legacyRuntime = {
 BrowserOpenURL, ClipboardGetText, ClipboardSetText, WindowGetSize, WindowSetSize,
 EventsOn, EventsOnce, EventsOnMultiple, EventsEmit, EventsOff, EventsOffAll,
 LogDebug, LogInfo, LogError, LogWarning, LogPrint: LogInfo, LogTrace: LogDebug, LogFatal: LogError,
 WindowReload: () => launcher.Reload(), WindowReloadApp: () => launcher.ForceReload(),
 WindowCenter: () => launcher.Center(), WindowSetTitle: (title: string) => launcher.SetTitle(title),
 WindowSetPosition: (x: number, y: number) => launcher.SetPosition(x, y), WindowGetPosition: () => launcher.Position(),
 WindowHide: () => launcher.Hide(), WindowShow: () => launcher.Show(),
 WindowSetAlwaysOnTop: (value: boolean) => launcher.SetAlwaysOnTop(value),
 WindowSetMinSize: (width: number, height: number) => launcher.SetMinSize(width, height),
 WindowSetMaxSize: (width: number, height: number) => launcher.SetMaxSize(width, height),
 WindowFullscreen: () => launcher.Fullscreen(), WindowUnfullscreen: () => launcher.UnFullscreen(),
 WindowIsFullscreen: () => launcher.IsFullscreen(),
 WindowMaximise: () => launcher.Maximise(), WindowUnmaximise: () => launcher.UnMaximise(),
 WindowToggleMaximise: () => launcher.ToggleMaximise(), WindowIsMaximised: () => launcher.IsMaximised(),
 WindowMinimise: () => launcher.Minimise(), WindowUnminimise: () => launcher.UnMinimise(),
 WindowIsMinimised: () => launcher.IsMinimised(),
 WindowIsNormal: async () => !(await launcher.IsFullscreen()) && !(await launcher.IsMaximised()) && !(await launcher.IsMinimised()),
 WindowSetBackgroundColour: (r: number, g: number, b: number, a: number) => launcher.SetBackgroundColour(r, g, b, a),
 Quit: () => Application.Quit(), Hide: () => Application.Hide(), Show: () => Application.Show(),
 Environment: async () => { const env = await System.Environment(); return {buildType: env.Debug ? 'dev' : 'production', platform: env.OS, arch: env.Arch} },
 ScreenGetAll: async () => {
   const [screens, current] = await Promise.all([Screens.GetAll(), Screens.GetCurrent()])
   return screens.map(screen => ({isPrimary: screen.IsPrimary, isCurrent: screen.ID === current.ID, width: screen.Size.Width, height: screen.Size.Height}))
 },
}
