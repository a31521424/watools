import {useCallback, useEffect, useRef, useState} from "react";
import {useLocation, useSearchParams} from "wouter";
import {useAppStore, usePluginStore} from "@/stores";
import {createWaToolsApi} from "@/api/api";
import {normalizePluginAssetPath} from "@/lib/plugin";

export const WaPlugin = () => {
    const [searchParams] = useSearchParams()
    const iframeRef = useRef<HTMLIFrameElement | null>(null)
    const iframeCleanupRef = useRef<(() => void) | null>(null)
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
            setIframeHeight(null)
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
            setIframeHeight(null)
        }
    }, [packageId, file, seed, getPluginById]);

    useEffect(() => {
        return () => {
            iframeCleanupRef.current?.()
            iframeCleanupRef.current = null
        }
    }, []);

    const syncIframeHeight = useCallback(() => {
        const iframeWindow = iframeRef.current?.contentWindow
        const iframeDocument = iframeWindow?.document
        if (!iframeDocument) {
            return
        }

        const bodyHeight = iframeDocument.body?.scrollHeight || 0
        const bodyOffsetHeight = iframeDocument.body?.offsetHeight || 0
        const docHeight = iframeDocument.documentElement?.scrollHeight || 0
        const docOffsetHeight = iframeDocument.documentElement?.offsetHeight || 0
        const nextHeight = Math.max(bodyHeight, bodyOffsetHeight, docHeight, docOffsetHeight)

        if (nextHeight > 0) {
            setIframeHeight(prevHeight => prevHeight === nextHeight ? prevHeight : nextHeight)
        }
    }, [])

    const handleIframeLoad = useCallback(() => {
        iframeCleanupRef.current?.()
        iframeCleanupRef.current = null

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

        const iframeDocument = iframeWindow.document
        const resizeObserver = new ResizeObserver(() => {
            syncIframeHeight()
        })

        if (iframeDocument.body) {
            resizeObserver.observe(iframeDocument.body)
        }
        if (iframeDocument.documentElement) {
            resizeObserver.observe(iframeDocument.documentElement)
        }

        const rafId = iframeWindow.requestAnimationFrame(() => {
            syncIframeHeight()
        })
        const timeoutId = iframeWindow.setTimeout(() => {
            syncIframeHeight()
        }, 120)

        iframeCleanupRef.current = () => {
            iframeWindow.removeEventListener('keydown', handleHotkey)
            resizeObserver.disconnect()
            iframeWindow.cancelAnimationFrame(rafId)
            iframeWindow.clearTimeout(timeoutId)
        }
    }, [packageId, inputValue, clearInputValue, syncIframeHeight]);

    return <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {pluginUrl && <iframe
            ref={iframeRef}
            style={{
                height: iframeHeight ? `${iframeHeight}px` : '720px',
            }}
            className="block min-h-0 w-full border-0"
            src={pluginUrl} onLoad={handleIframeLoad}
        />}
        {!pluginUrl && 'loading...'}
    </div>
}
