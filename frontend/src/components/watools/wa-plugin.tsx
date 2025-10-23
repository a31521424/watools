import {useLocation, useSearchParams} from "wouter";
import {useEffect, useRef, useState} from "react";
import {usePluginStore} from "@/stores";

export const WaPlugin = () => {
    const [searchParams] = useSearchParams()
    const iframeRef = useRef<HTMLIFrameElement | null>(null)
    const {getPluginById} = usePluginStore()
    const [pluginUrl, setPluginUrl] = useState<string | null>(null)
    const [, navigate] = useLocation()

    const packageId = searchParams.get('packageId') || ''
    const file = searchParams.get('file')

    const handleHotkey = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            console.log('WaPlugin handleHotkey Escape')
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
        const url = `${plugin.homeUrl}/${file}`
        setPluginUrl(url)
        return () => {
            setPluginUrl(null)
        }
    }, [packageId, file]);

    const handleIframeLoad = () => {
        if (!iframeRef.current) {
            return
        }
        const iframeWindow = iframeRef.current.contentWindow
        if (!iframeWindow) {
            return
        }
        iframeWindow.addEventListener('keydown', handleHotkey)
    }

    return <div className="flex-1 overflow-hidden">
        {pluginUrl && <iframe
            ref={iframeRef}
            className="w-svw h-svh min-h-[500px]"
            src={pluginUrl} onLoad={handleIframeLoad}
        />}
        {!pluginUrl && 'loading...'}
    </div>
}

