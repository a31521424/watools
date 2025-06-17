import useResizeWindow from "@/hooks/useResizeWindow";
import {WaCommand} from "./wa-command";

const Main = () => {
    const windowRef = useResizeWindow<HTMLDivElement>()

    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        <WaCommand/>
    </div>
}


export default Main