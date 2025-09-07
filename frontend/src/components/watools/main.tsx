import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";
import {HideAppApi, ReloadApi, ReloadAppApi} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {useEffect} from "react";

const Main = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })

    useWindowFocus((focused) => {
        if (!focused) {
            if (isDevMode()) {
                return
            }
            HideAppApi()
        }
    });


    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            const ctrlKey = e.metaKey || e.ctrlKey
            const shiftKey = e.shiftKey
            const key = e.key

            if (ctrlKey && shiftKey && key === "R") {
                ReloadAppApi()
            } else if (ctrlKey && key === "r") {
                ReloadApi()
            }
        }
        window.addEventListener("keydown", handler)
        return () => {
            window.removeEventListener("keydown", handler)
        }
    }, [])

    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        <WaCommand/>
    </div>
}


export default Main