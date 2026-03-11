import {useEffect, useMemo, useRef, useState} from "react";
import {useLocation, useSearchParams} from "wouter";
import {useAppStore, usePluginStore} from "@/stores";
import {createWaToolsApi} from "@/api/api";
import {normalizePluginAssetPath} from "@/lib/plugin";
import {buildPluginContext, getLegacySeedValue, resolvePluginLaunchContext} from "@/lib/plugin-context";

export const WaPlugin = () => {
    const [searchParams] = useSearchParams()
    const iframeRef = useRef<HTMLIFrameElement | null>(null)
    const {getPluginById} = usePluginStore()
    const [pluginUrl, setPluginUrl] = useState<string | null>(null)
    const [, navigate] = useLocation()
    const inputValue = useAppStore(state => state.value)
    const inputValueType = useAppStore(state => state.valueType)
    const clipboardContentType = useAppStore(state => state.clipboardContentType)
    const clipboardImageBase64 = useAppStore(state => state.imageBase64)
    const clipboardFiles = useAppStore(state => state.files)
    const clearInputValue = useAppStore(state => state.clearValue)

    const packageId = searchParams.get('packageId') || ''
    const file = searchParams.get('file')
    const launchId = searchParams.get('launchId')
    const seed = searchParams.get('seed') || ''
    const liveContext = useMemo(() => buildPluginContext(
        inputValue,
        inputValueType,
        clipboardContentType || undefined,
        clipboardImageBase64,
        clipboardFiles,
    ), [inputValue, inputValueType, clipboardContentType, clipboardImageBase64, clipboardFiles])
    const launchContext = useMemo(() => resolvePluginLaunchContext({
        launchId,
        seed,
        liveContext,
    }), [launchId, seed, liveContext])

    const handleHotkey = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            e.preventDefault()
            e.stopPropagation()
            navigate('/')
        }
    }
    useEffect(() => {
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, []);

    useEffect(() => {
        const plugin = getPluginById(packageId)
        const safeFile = normalizePluginAssetPath(file)
        const matchedEntry = plugin?.enabled ? plugin.entry.find(entry => entry.type === "ui" && entry.file === safeFile) : undefined

        if (!plugin || !plugin.enabled || !safeFile || !matchedEntry) {
            setPluginUrl(null)
            return
        }
        const params = new URLSearchParams({
            t: Date.now().toString(),
        })
        const legacySeed = getLegacySeedValue(launchContext)
        if (legacySeed) {
            params.set('seed', legacySeed)
        }
        const url = `${plugin.homeUrl}/${safeFile}?${params.toString()}`
        setPluginUrl(url)
        return () => {
            setPluginUrl(null)
        }
    }, [packageId, file, launchContext, getPluginById]);

    const handleIframeLoad = () => {
        if (!iframeRef.current) {
            return
        }
        const iframeWindow = iframeRef.current.contentWindow
        if (!iframeWindow) {
            return
        }
        iframeWindow.addEventListener('keydown', handleHotkey)

        // TODO: better way to expose runtime api to iframe
        // @ts-ignore
        iframeWindow.runtime = window.runtime
        // @ts-ignore
        iframeWindow.watools = createWaToolsApi(packageId)
        // @ts-ignore
        iframeWindow.inputValue = launchContext.input.value
        // @ts-ignore
        iframeWindow.pluginContext = launchContext
        iframeWindow.dispatchEvent(new CustomEvent('watools:context-ready', {
            // @ts-ignore
            detail: (iframeWindow as any).pluginContext,
        }))

        clearInputValue()
    }

    useEffect(() => {
        return () => {
            const iframeWindow = iframeRef.current?.contentWindow
            iframeWindow?.removeEventListener('keydown', handleHotkey)
        }
    }, [])

    return <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
        {pluginUrl && <iframe
            ref={iframeRef}
            className="block h-full min-h-0 w-full flex-1 border-0"
            src={pluginUrl} onLoad={handleIframeLoad}
        />}
        {!pluginUrl && 'loading...'}
    </div>
}
