import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {EventsEmit} from "../../../wailsjs/runtime/runtime";

const Main = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })

    // Enable panel-like behavior by emitting focus events to backend
    useWindowFocus((focused) => {
        if (!focused) {
            EventsEmit("window-blur");
        }
    });

    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        <WaCommand/>
    </div>
}


export default Main