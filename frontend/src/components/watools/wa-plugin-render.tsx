import {usePluginActions} from "@/store/pluginStore";
import {useLocation, useParams} from "wouter";
import {useEffect, useRef} from "react";

export const WaPluginRender = () => {
    const containerRef = useRef<HTMLDivElement>(null)
    const {getPluginEntry} = usePluginActions()
    const params: { entryID: string } = useParams()
    const [location, navigate] = useLocation()

    const entry = getPluginEntry(params.entryID)

    const input: string = history.state.input

    const handlerEscape = () => {
        navigate("/")
    }

    useEffect(() => {
        const hotkeyHandler = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                e.preventDefault()
                e.stopPropagation()
                handlerEscape()
            }
        }
        window.addEventListener("keydown", hotkeyHandler)
        return () => {
            window.removeEventListener("keydown", hotkeyHandler)
        }
    }, [])


    useEffect(() => {
        const container = containerRef.current
        if (!entry) {
            return
        }
        if (!container) {
            return
        }
        entry.render && entry.render(containerRef.current, input)
        return () => {
            container.innerHTML = ""
        }
    }, [containerRef]);


    if (!entry) {
        navigate("/")
        return null
    }
    return <div ref={containerRef} className="min-h-[400px]"></div>
}