import {useLocation, useSearchParams} from "wouter";
import {useCallback, useEffect, useRef, useState} from "react";
import {useAppStore, usePluginStore} from "@/stores";
import {createWaToolsApi} from "@/api/api";
import {normalizePluginAssetPath} from "@/lib/plugin";

export const WaPlugin = () => {
    const [searchParams] = useSearchParams()
    const iframeRef = useRef<HTMLIFrameElement | null>(null)
    const {getPluginById} = usePluginStore()
    const [pluginUrl, setPluginUrl] = useState<string | null>(null)
    const [, navigate] = useLocation()
    const [iframeHeight, setIframeHeight] = useState<number | null>(null)
    const inputValue = useAppStore(state => state.value)
    const clearInputValue = useAppStore(state => state.clearValue)

    const packageId = searchParams.get('packageId') || ''
    const file = searchParams.get('file')
    const seed = searchParams.get('seed') || ''

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
        if (seed) {
            params.set('seed', seed)
        }
        const url = `${plugin.homeUrl}/${safeFile}?${params.toString()}`
        setPluginUrl(url)
        return () => {
            setPluginUrl(null)
        }
    }, [packageId, file, seed, getPluginById]);


    const handleIframeLoad = useCallback(() => {
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
        iframeWindow.inputValue = inputValue

        clearInputValue()

        const height = iframeWindow.document.body.scrollHeight
        if (height) {
            setIframeHeight(height)
        }
    }, [iframeRef.current, packageId, inputValue, clearInputValue]);

    return <div className="flex-1 overflow-hidden">
        {pluginUrl && <iframe
            ref={iframeRef}
            style={{
                height: iframeHeight ? `${iframeHeight}px` : '100%',
            }}
            className="w-svw h-svh min-h-[500px]"
            src={pluginUrl} onLoad={handleIframeLoad}
        />}
        {!pluginUrl && 'loading...'}
    </div>
}
