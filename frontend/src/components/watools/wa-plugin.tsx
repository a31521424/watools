import {useLocation, useSearchParams} from "wouter";
import {useCallback, useEffect, useRef, useState} from "react";
import {useAppStore, usePluginStore} from "@/stores";
import {createWaToolsApi} from "@/api/api";

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
        if (!plugin) {
            return
        }
        const url = `${plugin.homeUrl}/${file}?t=${Date.now()}`
        setPluginUrl(url)
        return () => {
            setPluginUrl(null)
        }
    }, [packageId, file]);


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

