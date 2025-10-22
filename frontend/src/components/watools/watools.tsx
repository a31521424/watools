import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {HideAppApi} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";
import {Route} from "wouter";
import {WaPlugin} from "@/components/watools/wa-plugin";

const Watools = () => {
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
    })

    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide border-0">
        <Route path='/'>
            <WaCommand/>
        </Route>
        <Route path='/plugin'>
            <WaPlugin />
        </Route>
    </div>
}


export default Watools