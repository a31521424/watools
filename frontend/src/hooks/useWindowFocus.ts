import {useEffect} from "react";

export const useWindowFocus = (onFocusChange: (focus: boolean) => void, deps?: React.DependencyList) => {
    useEffect(() => {
        const handleFocus = () => {
            onFocusChange(true)
        }
        const handleBlur = () => {
            onFocusChange(false)
        }
        window.addEventListener('focus', handleFocus)
        window.addEventListener('blur', handleBlur)
        return () => {
            window.removeEventListener('focus', handleFocus)
            window.removeEventListener('blur', handleBlur)
        }
    }, deps)
}