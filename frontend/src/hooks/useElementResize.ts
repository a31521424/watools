import {RefObject, useEffect, useRef} from 'react'
import {WindowGetSize, WindowSetSize} from "@/lib/legacy-runtime";


export const useElementResize = <T extends HTMLElement>(
    params: { onResize: (entries: ResizeObserverEntry[]) => void }
): RefObject<T | null> => {
    const elementRef = useRef<T>(null)

    useEffect(() => {
        const node = elementRef.current

        if (!node) {
            return
        }
        const observer = new ResizeObserver(params.onResize)

        observer.observe(node)
        return () => {
            observer.disconnect()
        }
    }, [params.onResize])

    return elementRef
}

let resizeRevision = 0
export const resizeWindowHeight = async (entries: ResizeObserverEntry[]) => {
    const revision = ++resizeRevision
    let height: number | null = null
    for (const entry of entries) {
        if (entry.target.clientHeight) {
            height = entry.target.clientHeight
        }
    }
    if (height) height = Math.max(64, height)
    const currentSize = await WindowGetSize()
    if (revision === resizeRevision && height && currentSize.h !== height) {
        await WindowSetSize(currentSize.w, height)
    }
}
