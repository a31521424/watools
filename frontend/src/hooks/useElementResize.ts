import {RefObject, useEffect, useRef} from 'react'
import {WindowGetSize, WindowSetSize} from "../../wailsjs/runtime";


export const useElementResize = <T extends HTMLElement>(
    params: { onResize: (entries: ResizeObserverEntry[]) => void }
): RefObject<T> => {
    const elementRef = useRef<T>(null)
    useEffect(() => {
        const node = elementRef.current

        if (!node) {
            return
        }
        const observer = new ResizeObserver(params.onResize)

        observer.observe(node)
        return () => {
            observer.unobserve(node)
        }
    }, [elementRef])
    return elementRef
}

export const resizeWindowHeight = async (entries: ResizeObserverEntry[]) => {
    let height: number | null = null
    for (const entry of entries) {
        if (entry.target.clientHeight) {
            height = entry.target.clientHeight
        }
    }
    const currentSize = await WindowGetSize()
    if (height && currentSize.h !== height) {
        WindowSetSize(currentSize.w, height)
    }
}
