import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route} from "wouter";
import {WaPlugin} from "@/components/watools/wa-plugin";

const Watools = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })


    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide border-0">
        <Route path='/'>
            <WaCommand/>
        </Route>
        <Route path='/plugin'>
            <WaPlugin/>
        </Route>
    </div>
}


export default Watools