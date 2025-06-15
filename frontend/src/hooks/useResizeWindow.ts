import {RefObject, useEffect, useRef} from 'react'
import {WindowGetSize, WindowSetSize} from "../../wailsjs/runtime";


const useResizeWindow = <T extends HTMLElement>(): RefObject<T> => {
    const elementRef = useRef<T>(null)
    useEffect(() => {
        const node = elementRef.current

        if (!node) {
            return
        }
        const observer = new ResizeObserver(entries => {
            let height: number | null = null
            for (const entry of entries) {
                if (entry.target.clientHeight) {
                    height = entry.target.clientHeight
                }
            }
            resizeWindow(height, null).then(_ => _)
        })

        observer.observe(node)
        return () => {
            observer.unobserve(node)
        }
    }, [elementRef])
    return elementRef
}

const resizeWindow = async (height: number | null, width: number | null) => {
    const currentSize = await WindowGetSize()
    if (height && currentSize.h !== height) {
        WindowSetSize(currentSize.w, height)
    }
    if (width && currentSize.w !== width) {
        WindowSetSize(width, currentSize.h)
    }

}

export default useResizeWindow